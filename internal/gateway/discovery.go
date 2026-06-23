package gateway

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"
)

// ServiceEndpoint represents a single instance of a service
type ServiceEndpoint struct {
	ID        string    `json:"id"`
	Address   string    `json:"address"`
	Port      int       `json:"port"`
	Weight    int       `json:"weight"` // For load balancing
	Healthy   bool      `json:"healthy"`
	Version   string    `json:"version"`
	Metadata  map[string]string `json:"metadata,omitempty"`
	LastPing  time.Time `json:"last_ping"`
}

// ServiceRegistry maintains the list of available services
type ServiceRegistry struct {
	mu        sync.RWMutex
	services  map[string][]ServiceEndpoint
	heartbeat map[string]time.Time
}

// NewServiceRegistry creates a new service registry
func NewServiceRegistry() *ServiceRegistry {
	return &ServiceRegistry{
		services:  make(map[string][]ServiceEndpoint),
		heartbeat: make(map[string]time.Time),
	}
}

// Register adds a new service endpoint to the registry
func (r *ServiceRegistry) Register(serviceName string, endpoint ServiceEndpoint) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Validate endpoint
	if endpoint.Address == "" {
		return fmt.Errorf("endpoint address is required")
	}
	if endpoint.Port <= 0 || endpoint.Port > 65535 {
		return fmt.Errorf("invalid port: %d", endpoint.Port)
	}

	// Set defaults
	if endpoint.ID == "" {
		endpoint.ID = fmt.Sprintf("%s-%s-%d", serviceName, endpoint.Address, endpoint.Port)
	}
	if endpoint.Weight == 0 {
		endpoint.Weight = 100
	}
	endpoint.Healthy = true
	endpoint.LastPing = time.Now()

	// Check if endpoint already exists
	for i, ep := range r.services[serviceName] {
		if ep.Address == endpoint.Address && ep.Port == endpoint.Port {
			r.services[serviceName][i] = endpoint
			return nil
		}
	}

	r.services[serviceName] = append(r.services[serviceName], endpoint)
	r.heartbeat[endpoint.ID] = time.Now()
	
	return nil
}

// Deregister removes a service endpoint from the registry
func (r *ServiceRegistry) Deregister(serviceName, endpointID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	endpoints := r.services[serviceName]
	for i, ep := range endpoints {
		if ep.ID == endpointID {
			r.services[serviceName] = append(endpoints[:i], endpoints[i+1:]...)
			delete(r.heartbeat, endpointID)
			return nil
		}
	}

	return fmt.Errorf("endpoint %s not found for service %s", endpointID, serviceName)
}

// GetEndpoints returns all healthy endpoints for a service
func (r *ServiceRegistry) GetEndpoints(serviceName string) ([]ServiceEndpoint, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	endpoints, ok := r.services[serviceName]
	if !ok {
		return nil, fmt.Errorf("service %s not found", serviceName)
	}

	// Filter healthy endpoints
	healthy := make([]ServiceEndpoint, 0)
	for _, ep := range endpoints {
		if ep.Healthy {
			healthy = append(healthy, ep)
		}
	}

	if len(healthy) == 0 {
		return nil, fmt.Errorf("no healthy endpoints for service %s", serviceName)
	}

	return healthy, nil
}

// GetEndpointByID returns a specific endpoint by ID
func (r *ServiceRegistry) GetEndpointByID(serviceName, endpointID string) (*ServiceEndpoint, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	endpoints, ok := r.services[serviceName]
	if !ok {
		return nil, fmt.Errorf("service %s not found", serviceName)
	}

	for _, ep := range endpoints {
		if ep.ID == endpointID {
			return &ep, nil
		}
	}

	return nil, fmt.Errorf("endpoint %s not found", endpointID)
}

// UpdateHealth updates the health status of an endpoint
func (r *ServiceRegistry) UpdateHealth(serviceName, endpointID string, healthy bool) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	endpoints := r.services[serviceName]
	for i, ep := range endpoints {
		if ep.ID == endpointID {
			endpoints[i].Healthy = healthy
			endpoints[i].LastPing = time.Now()
			return nil
		}
	}

	return fmt.Errorf("endpoint %s not found", endpointID)
}

// RefreshHeartbeat updates the heartbeat timestamp for an endpoint
func (r *ServiceRegistry) RefreshHeartbeat(serviceName, endpointID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.heartbeat[endpointID] = time.Now()
	return nil
}

// ListServices returns all registered service names
func (r *ServiceRegistry) ListServices() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	services := make([]string, 0, len(r.services))
	for name := range r.services {
		services = append(services, name)
	}
	return services
}

// GetStats returns statistics about registered services
func (r *ServiceRegistry) GetStats() map[string]ServiceStats {
	r.mu.RLock()
	defer r.mu.RUnlock()

	stats := make(map[string]ServiceStats)
	for name, endpoints := range r.services {
		total := len(endpoints)
		healthy := 0
		for _, ep := range endpoints {
			if ep.Healthy {
				healthy++
			}
		}
		stats[name] = ServiceStats{
			TotalEndpoints:   total,
			HealthyEndpoints: healthy,
			UnhealthyEndpoints: total - healthy,
		}
	}
	return stats
}

type ServiceStats struct {
	TotalEndpoints     int `json:"total_endpoints"`
	HealthyEndpoints  int `json:"healthy_endpoints"`
	UnhealthyEndpoints int `json:"unhealthy_endpoints"`
}

// HealthCheck performs health checks on all endpoints
func (r *ServiceRegistry) HealthCheck(timeout time.Duration) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	client := &http.Client{Timeout: timeout}

	for serviceName, endpoints := range r.services {
		for i, ep := range endpoints {
			url := fmt.Sprintf("http://%s:%d/health", ep.Address, ep.Port)
			
			resp, err := client.Get(url)
			if err != nil || resp.StatusCode != http.StatusOK {
				r.mu.RUnlock()
				r.UpdateHealth(serviceName, ep.ID, false)
				r.mu.RLock()
				continue
			}
			resp.Body.Close()

			// Update heartbeat
			r.mu.RUnlock()
			r.RefreshHeartbeat(serviceName, ep.ID)
			r.mu.RLock()
			
			// Mark as healthy if was unhealthy
			if !endpoints[i].Healthy {
				r.mu.RUnlock()
				r.UpdateHealth(serviceName, ep.ID, true)
				r.mu.RLock()
			}
		}
	}
}

// MarshalJSON implements json.Marshaler for ServiceRegistry
func (r *ServiceRegistry) MarshalJSON() ([]byte, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	type endpointInfo struct {
		Endpoints []ServiceEndpoint `json:"endpoints"`
		Stats    ServiceStats      `json:"stats"`
	}

	result := make(map[string]endpointInfo)
	for name, endpoints := range r.services {
		healthy := 0
		for _, ep := range endpoints {
			if ep.Healthy {
				healthy++
			}
		}
		result[name] = endpointInfo{
			Endpoints: endpoints,
			Stats: ServiceStats{
				TotalEndpoints:     len(endpoints),
				HealthyEndpoints:  healthy,
				UnhealthyEndpoints: len(endpoints) - healthy,
			},
		}
	}

	return json.Marshal(result)
}
