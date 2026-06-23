package gateway

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// InternalRoute represents an internal service route configuration
type InternalRoute struct {
	PathPrefix   string            `json:"path_prefix"`   // e.g., "/api/internal/skills"
	ServiceName  string            `json:"service_name"`  // e.g., "skills-service"
	RewriteRule  string            `json:"rewrite_rule"`  // Optional path rewrite pattern
	Timeout      time.Duration     `json:"timeout"`       // Request timeout
	RetryCount   int               `json:"retry_count"`   // Number of retries on failure
	AllowedRoles []string           `json:"allowed_roles"` // Required roles for access
	Metadata     map[string]string `json:"metadata,omitempty"`
}

// InternalRouter handles routing to internal services
type InternalRouter struct {
	registry   *ServiceRegistry
	routes     map[string]InternalRoute
	routeOrder []string // Ordered list of route prefixes for matching
	metrics    *RouterMetrics
	client     *http.Client
}

// RouterMetrics holds Prometheus metrics for the router
type RouterMetrics struct {
	RequestsTotal    *prometheus.CounterVec
	RequestDuration  *prometheus.HistogramVec
	ActiveRequests   prometheus.Gauge
	UpstreamErrors   *prometheus.CounterVec
	RetryAttempts    prometheus.Counter
}

func newRouterMetrics(namespace string) *RouterMetrics {
	return &RouterMetrics{
		RequestsTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "internal_router_requests_total",
				Help:      "Total number of internal router requests",
			},
			[]string{"service", "status"},
		),
		RequestDuration: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: namespace,
				Name:      "internal_router_request_duration_seconds",
				Help:      "Request duration in seconds",
				Buckets:   prometheus.DefBuckets,
			},
			[]string{"service"},
		),
		ActiveRequests: promauto.NewGauge(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "internal_router_active_requests",
				Help:      "Number of active requests",
			},
		),
		UpstreamErrors: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "internal_router_upstream_errors_total",
				Help:      "Total number of upstream errors",
			},
			[]string{"service", "error_type"},
		),
		RetryAttempts: promauto.NewCounter(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "internal_router_retry_attempts_total",
				Help:      "Total number of retry attempts",
			},
		),
	}
}

// NewInternalRouter creates a new internal router
func NewInternalRouter(registry *ServiceRegistry) *InternalRouter {
	r := &InternalRouter{
		registry:   registry,
		routes:    make(map[string]InternalRoute),
		routeOrder: make([]string, 0),
		metrics:    newRouterMetrics("nexusai"),
		client: &http.Client{
			Timeout: 30 * time.Second,
			Transport: &http.Transport{
				MaxIdleConns:        100,
				MaxIdleConnsPerHost: 10,
				IdleConnTimeout:     90 * time.Second,
			},
		},
	}
	return r
}

// RegisterRoute adds a new route configuration
func (r *InternalRouter) RegisterRoute(route InternalRoute) error {
	if route.PathPrefix == "" {
		return fmt.Errorf("path prefix is required")
	}
	if route.ServiceName == "" {
		return fmt.Errorf("service name is required")
	}
	if route.Timeout == 0 {
		route.Timeout = 30 * time.Second
	}
	if route.RetryCount == 0 {
		route.RetryCount = 1
	}

	// Validate path prefix format
	if !strings.HasPrefix(route.PathPrefix, "/") {
		route.PathPrefix = "/" + route.PathPrefix
	}

	r.routes[route.PathPrefix] = route
	r.routeOrder = append(r.routeOrder, route.PathPrefix)
	
	return nil
}

// UnregisterRoute removes a route
func (r *InternalRouter) UnregisterRoute(pathPrefix string) error {
	if _, ok := r.routes[pathPrefix]; !ok {
		return fmt.Errorf("route %s not found", pathPrefix)
	}
	delete(r.routes, pathPrefix)
	
	// Update route order
	newOrder := make([]string, 0)
	for _, p := range r.routeOrder {
		if p != pathPrefix {
			newOrder = append(newOrder, p)
		}
	}
	r.routeOrder = newOrder
	
	return nil
}

// matchRoute finds the most specific matching route for a path
func (r *InternalRouter) matchRoute(path string) (*InternalRoute, string, error) {
	var longestMatch *InternalRoute
	var matchedPrefix string

	// Sort routes by length (most specific first) - longest prefix match
	for _, prefix := range r.routeOrder {
		if strings.HasPrefix(path, prefix) {
			if longestMatch == nil || len(prefix) > len(matchedPrefix) {
				route := r.routes[prefix]
				longestMatch = &route
				matchedPrefix = prefix
			}
		}
	}

	if longestMatch == nil {
		return nil, "", fmt.Errorf("no route found for path: %s", path)
	}

	// Extract the remaining path after the prefix
	remainingPath := strings.TrimPrefix(path, matchedPrefix)
	if remainingPath == "" {
		remainingPath = "/"
	}

	return longestMatch, remainingPath, nil
}

// RouteRequest routes an HTTP request to the appropriate internal service
func (r *InternalRouter) RouteRequest(ctx context.Context, path string, method string, headers http.Header, body io.Reader) (*http.Response, error) {
	route, remainingPath, err := r.matchRoute(path)
	if err != nil {
		return nil, err
	}

	// Get healthy endpoints for the service
	endpoints, err := r.registry.GetEndpoints(route.ServiceName)
	if err != nil {
		r.metrics.UpstreamErrors.WithLabelValues(route.ServiceName, "no_endpoints").Inc()
		return nil, fmt.Errorf("failed to get endpoints for %s: %w", route.ServiceName, err)
	}

	// Select endpoint using weighted round-robin
	endpoint := r.selectEndpoint(endpoints)
	
	// Build upstream URL
	upstreamPath := remainingPath
	if route.RewriteRule != "" {
		upstreamPath = r.applyRewriteRule(route.RewriteRule, path, remainingPath)
	}

	upstreamURL := fmt.Sprintf("http://%s:%d%s", endpoint.Address, endpoint.Port, upstreamPath)

	return r.executeRequestWithRetry(ctx, upstreamURL, method, route, headers, body)
}

// selectEndpoint implements weighted round-robin selection
func (r *InternalRouter) selectEndpoint(endpoints []ServiceEndpoint) ServiceEndpoint {
	totalWeight := 0
	for _, ep := range endpoints {
		totalWeight += ep.Weight
	}

	// Random selection weighted by endpoint weight
	rng := rand.Intn(totalWeight)
	runningWeight := 0
	for _, ep := range endpoints {
		runningWeight += ep.Weight
		if runningWeight > rng {
			return ep
		}
	}

	return endpoints[0]
}

// applyRewriteRule applies a path rewrite rule
func (r *InternalRouter) applyRewriteRule(rule, originalPath, remainingPath string) string {
	// Simple rewrite rule support: $1, $2 placeholders
	// Rule format: "/api/v1" -> "/internal/v1"
	parts := strings.Split(rule, "->")
	if len(parts) == 2 {
		sourcePrefix := strings.TrimSpace(parts[0])
		targetPrefix := strings.TrimSpace(parts[1])
		return strings.Replace(originalPath, sourcePrefix, targetPrefix, 1)
	}
	return remainingPath
}

// executeRequestWithRetry executes request with retry logic
func (r *InternalRouter) executeRequestWithRetry(ctx context.Context, url, method string, route *InternalRoute, headers http.Header, body io.Reader) (*http.Response, error) {
	var lastErr error

	for attempt := 0; attempt <= route.RetryCount; attempt++ {
		if attempt > 0 {
			r.metrics.RetryAttempts.Inc()
			time.Sleep(time.Duration(attempt*100) * time.Millisecond) // Simple backoff
		}

		req, err := http.NewRequestWithContext(ctx, method, url, body)
		if err != nil {
			return nil, fmt.Errorf("failed to create request: %w", err)
		}

		// Copy headers
		for k, v := range headers {
			req.Header[k] = v
		}

		// Set timeout
		client := &http.Client{Timeout: route.Timeout}

		r.metrics.ActiveRequests.Inc()
		start := time.Now()

		resp, err := client.Do(req)
		
		r.metrics.ActiveRequests.Dec()
		r.metrics.RequestDuration.WithLabelValues(route.ServiceName).Observe(time.Since(start).Seconds())

		if err != nil {
			lastErr = err
			r.metrics.UpstreamErrors.WithLabelValues(route.ServiceName, "connection_error").Inc()
			continue
		}

		// Check if response is successful or should retry
		if resp.StatusCode >= 500 && attempt < route.RetryCount {
			resp.Body.Close()
			lastErr = fmt.Errorf("upstream error: %d", resp.StatusCode)
			r.metrics.UpstreamErrors.WithLabelValues(route.ServiceName, "upstream_error").Inc()
			continue
		}

		r.metrics.RequestsTotal.WithLabelValues(route.ServiceName, fmt.Sprintf("%d", resp.StatusCode)).Inc()
		return resp, nil
	}

	return nil, fmt.Errorf("all retry attempts failed: %w", lastErr)
}

// GetRegisteredRoutes returns all registered routes
func (r *InternalRouter) GetRegisteredRoutes() []InternalRoute {
	routes := make([]InternalRoute, 0, len(r.routes))
	for _, route := range r.routes {
		routes = append(routes, route)
	}
	return routes
}

// GetRouteStats returns statistics for all routes
func (r *InternalRouter) GetRouteStats() map[string]interface{} {
	stats := make(map[string]interface{})
	registryStats := r.registry.GetStats()

	for path, route := range r.routes {
		endpointStats, ok := registryStats[route.ServiceName]
		if !ok {
			endpointStats = ServiceStats{}
		}
		stats[path] = map[string]interface{}{
			"service":      route.ServiceName,
			"endpoint_stats": endpointStats,
			"timeout":      route.Timeout.String(),
			"retry_count": route.RetryCount,
		}
	}

	return stats
}

// HealthCheck performs health checks on all registered services
func (r *InternalRouter) HealthCheck() map[string]bool {
	results := make(map[string]bool)
	
	for _, route := range r.routes {
		endpoints, err := r.registry.GetEndpoints(route.ServiceName)
		if err != nil {
			results[route.ServiceName] = false
			continue
		}
		results[route.ServiceName] = len(endpoints) > 0
	}

	return results
}

// MarshalRoutesJSON returns routes as JSON
func (r *InternalRouter) MarshalRoutesJSON() ([]byte, error) {
	return json.MarshalIndent(r.routes, "", "  ")
}
