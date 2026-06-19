package proxy

import (
	"fmt"
	"math/rand"
	"sync"
	"time"
)

// ProxyProtocol defines the type of proxy
type ProxyProtocol string

const (
	ProtocolHTTP   ProxyProtocol = "http"
	ProtocolHTTPS  ProxyProtocol = "https"
	ProtocolSOCKS5 ProxyProtocol = "socks5"
)

// ProxyTier defines the tier of the proxy
type ProxyTier string

const (
	TierFree      ProxyTier = "free"
	TierShared    ProxyTier = "shared"
	TierDedicated ProxyTier = "dedicated"
)

// ProxyStatus represents the current status of a proxy
type ProxyStatus string

const (
	StatusHealthy   ProxyStatus = "healthy"
	StatusUnhealthy ProxyStatus = "unhealthy"
	StatusUnknown   ProxyStatus = "unknown"
)

// ProxyConfig represents a proxy configuration
type ProxyConfig struct {
	ID          string        `json:"id"`
	URL         string        `json:"url"`
	Protocol    ProxyProtocol `json:"protocol"`
	Username    string        `json:"username,omitempty"`
	Password    string        `json:"password,omitempty"`
	Enabled     bool          `json:"enabled"`
	Healthy     bool          `json:"healthy"`
	LastChecked time.Time     `json:"last_checked,omitempty"`
	SuccessRate float64       `json:"success_rate"`
	AvgLatency  int           `json:"avg_latency_ms"`
	Region      string        `json:"region,omitempty"`
	Tier        ProxyTier     `json:"tier"`
	Tags        []string      `json:"tags,omitempty"`
	CreatedAt   time.Time     `json:"created_at"`
	UpdatedAt   time.Time     `json:"updated_at"`
}

// ProxyChain represents a chain of proxies
type ProxyChain struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	ProxyIDs  []string  `json:"proxy_ids"`
	Enabled   bool      `json:"enabled"`
	CreatedAt time.Time `json:"created_at"`
}

// TLSStealthConfig represents TLS stealth mode configuration
type TLSStealthConfig struct {
	Enabled               bool     `json:"enabled"`
	RandomizeTLSVersion   bool     `json:"randomize_tls_version"`
	RandomizeCipherSuites bool     `json:"randomize_cipher_suites"`
	MaskClientHello       bool     `json:"mask_client_hello"`
	UserAgentRotation     bool     `json:"user_agent_rotation"`
	UserAgents            []string `json:"user_agents,omitempty"`
}

// ProxyEngine manages the proxy pool
type ProxyEngine struct {
	mu            sync.RWMutex
	proxies       map[string]*ProxyConfig
	chains        map[string]*ProxyChain
	tlsStealth    *TLSStealthConfig
	requestCounts map[string]int64
	lastRotation  time.Time
}

var (
	proxyEngine *ProxyEngine
	engineOnce  sync.Once
)

// GetProxyEngine returns the singleton proxy engine
func GetProxyEngine() *ProxyEngine {
	engineOnce.Do(func() {
		proxyEngine = &ProxyEngine{
			proxies:       make(map[string]*ProxyConfig),
			chains:        make(map[string]*ProxyChain),
			tlsStealth:    &TLSStealthConfig{Enabled: false},
			requestCounts: make(map[string]int64),
		}
		proxyEngine.initDefaultProxies()
	})
	return proxyEngine
}

func (p *ProxyEngine) initDefaultProxies() {
	defaultProxies := []*ProxyConfig{
		{ID: "proxy-us-east-1", URL: "http://proxy-us-east.example.com:8080", Protocol: ProtocolHTTP, Region: "us-east-1", Tier: TierShared, Enabled: true, Healthy: true, SuccessRate: 0.98, AvgLatency: 45, CreatedAt: time.Now(), UpdatedAt: time.Now()},
		{ID: "proxy-eu-west-1", URL: "http://proxy-eu-west.example.com:8080", Protocol: ProtocolHTTP, Region: "eu-west-1", Tier: TierShared, Enabled: true, Healthy: true, SuccessRate: 0.97, AvgLatency: 62, CreatedAt: time.Now(), UpdatedAt: time.Now()},
		{ID: "proxy-ap-south-1", URL: "http://proxy-ap-south.example.com:8080", Protocol: ProtocolHTTP, Region: "ap-south-1", Tier: TierShared, Enabled: false, Healthy: false, SuccessRate: 0.85, AvgLatency: 120, CreatedAt: time.Now(), UpdatedAt: time.Now()},
	}
	for _, proxy := range defaultProxies {
		p.proxies[proxy.ID] = proxy
	}
}

// ListProxies returns all proxies
func (p *ProxyEngine) ListProxies() []*ProxyConfig {
	p.mu.RLock()
	defer p.mu.RUnlock()

	proxies := make([]*ProxyConfig, 0, len(p.proxies))
	for _, proxy := range p.proxies {
		proxies = append(proxies, proxy)
	}
	return proxies
}

// GetProxy returns a specific proxy by ID
func (p *ProxyEngine) GetProxy(id string) *ProxyConfig {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.proxies[id]
}

// CreateProxy adds a new proxy
func (p *ProxyEngine) CreateProxy(proxy *ProxyConfig) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if proxy.ID == "" {
		proxy.ID = fmt.Sprintf("proxy-%d", time.Now().UnixNano())
	}
	proxy.CreatedAt = time.Now()
	proxy.UpdatedAt = proxy.CreatedAt
	proxy.Healthy = false
	proxy.SuccessRate = 0.0

	p.proxies[proxy.ID] = proxy
	return nil
}

// UpdateProxy updates an existing proxy
func (p *ProxyEngine) UpdateProxy(id string, updates *ProxyConfig) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	proxy, exists := p.proxies[id]
	if !exists {
		return fmt.Errorf("proxy not found")
	}

	if updates.URL != "" {
		proxy.URL = updates.URL
	}
	if updates.Protocol != "" {
		proxy.Protocol = updates.Protocol
	}
	if updates.Username != "" {
		proxy.Username = updates.Username
	}
	if updates.Password != "" {
		proxy.Password = updates.Password
	}
	if updates.Region != "" {
		proxy.Region = updates.Region
	}
	if updates.Tier != "" {
		proxy.Tier = updates.Tier
	}
	if updates.Tags != nil {
		proxy.Tags = updates.Tags
	}
	proxy.Enabled = updates.Enabled
	proxy.UpdatedAt = time.Now()

	return nil
}

// DeleteProxy removes a proxy
func (p *ProxyEngine) DeleteProxy(id string) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if _, exists := p.proxies[id]; !exists {
		return fmt.Errorf("proxy not found")
	}

	delete(p.proxies, id)
	return nil
}

// EnableProxy enables a proxy
func (p *ProxyEngine) EnableProxy(id string) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	proxy, exists := p.proxies[id]
	if !exists {
		return fmt.Errorf("proxy not found")
	}
	proxy.Enabled = true
	proxy.UpdatedAt = time.Now()
	return nil
}

// DisableProxy disables a proxy
func (p *ProxyEngine) DisableProxy(id string) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	proxy, exists := p.proxies[id]
	if !exists {
		return fmt.Errorf("proxy not found")
	}
	proxy.Enabled = false
	proxy.UpdatedAt = time.Now()
	return nil
}

// TestProxy tests a proxy and updates its health status
func (p *ProxyEngine) TestProxy(id string) (*ProxyConfig, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	proxy, exists := p.proxies[id]
	if !exists {
		return nil, fmt.Errorf("proxy not found")
	}

	// Simulate health check
	proxy.LastChecked = time.Now()
	proxy.Healthy = rand.Float32() > 0.1 // 90% success rate simulation
	if proxy.Healthy {
		proxy.SuccessRate = min(1.0, proxy.SuccessRate+0.01)
		proxy.AvgLatency = int((float64(proxy.AvgLatency)*0.9 + float64(rand.Intn(50)+10)*0.1))
	} else {
		proxy.SuccessRate = max(0.0, proxy.SuccessRate-0.05)
	}

	return proxy, nil
}

// GetHealthyProxy returns a random healthy proxy, optionally filtered by region
func (p *ProxyEngine) GetHealthyProxy(region string) *ProxyConfig {
	p.mu.RLock()
	defer p.mu.RUnlock()

	var candidates []*ProxyConfig
	for _, proxy := range p.proxies {
		if proxy.Enabled && proxy.Healthy {
			if region == "" || proxy.Region == region {
				candidates = append(candidates, proxy)
			}
		}
	}

	if len(candidates) == 0 {
		return nil
	}

	return candidates[rand.Intn(len(candidates))]
}

// GetPoolHealth returns the health status of the proxy pool
func (p *ProxyEngine) GetPoolHealth() map[string]interface{} {
	p.mu.RLock()
	defer p.mu.RUnlock()

	total := len(p.proxies)
	healthy := 0
	enabled := 0

	avgSuccessRate := 0.0
	avgLatency := 0

	for _, proxy := range p.proxies {
		if proxy.Healthy {
			healthy++
		}
		if proxy.Enabled {
			enabled++
			avgSuccessRate += proxy.SuccessRate
			avgLatency += proxy.AvgLatency
		}
	}

	if enabled > 0 {
		avgSuccessRate /= float64(enabled)
		avgLatency /= enabled
	}

	return map[string]interface{}{
		"total_proxies":    total,
		"healthy_proxies":  healthy,
		"enabled_proxies":  enabled,
		"avg_success_rate": avgSuccessRate,
		"avg_latency_ms":   avgLatency,
		"timestamp":        time.Now().UTC().Format(time.RFC3339),
	}
}

// RotateProxy performs proxy rotation
func (p *ProxyEngine) RotateProxy() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.lastRotation = time.Now()

	// Simulate rotating through all proxies
	for _, proxy := range p.proxies {
		proxy.LastChecked = time.Now()
	}

	return nil
}

// Chain management

func (p *ProxyEngine) ListChains() []*ProxyChain {
	p.mu.RLock()
	defer p.mu.RUnlock()

	chains := make([]*ProxyChain, 0, len(p.chains))
	for _, chain := range p.chains {
		chains = append(chains, chain)
	}
	return chains
}

func (p *ProxyEngine) CreateChain(chain *ProxyChain) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if chain.ID == "" {
		chain.ID = fmt.Sprintf("chain-%d", time.Now().UnixNano())
	}
	chain.CreatedAt = time.Now()

	p.chains[chain.ID] = chain
	return nil
}

func (p *ProxyEngine) DeleteChain(id string) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if _, exists := p.chains[id]; !exists {
		return fmt.Errorf("chain not found")
	}

	delete(p.chains, id)
	return nil
}

// TLS Stealth management

func (p *ProxyEngine) GetTLSConfig() *TLSStealthConfig {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.tlsStealth
}

func (p *ProxyEngine) SetTLSConfig(config *TLSStealthConfig) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.tlsStealth = config
	return nil
}

func min(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}

func max(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}
