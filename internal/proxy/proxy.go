package proxy

import (
	"context"
	"crypto/tls"
	"fmt"
	"math/rand"
	"net"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/google/uuid"
)

// ProxyType defines the type of proxy
type ProxyType string

const (
	ProxyTypeHTTP     ProxyType = "http"
	ProxyTypeHTTPS    ProxyType = "https"
	ProxyTypeSOCKS5   ProxyType = "socks5"
	ProxyTypeHTTPConnect ProxyType = "http_connect"
)

// TLSFingerprint represents TLS fingerprinting options
type TLSFingerprint struct {
	JA3           string `json:"ja3"`
	JA4           string `json:"ja4"`
	JA4Short      string `json:"ja4_short"`
	IsJA4         bool   `json:"is_ja4"`
}

// BrowserProfile represents a browser fingerprint profile
type BrowserProfile struct {
	Name        string            `json:"name"`
	JA3         string            `json:"ja3"`
	JA4         string            `json:"ja4"`
	TLSVersion  string            `json:"tls_version"`
	CipherSuites []string         `json:"cipher_suites"`
	Extensions   []int            `json:"extensions"`
	ALPN        []string          `json:"alpn"`
	UserAgent   string            `json:"user_agent"`
	Accept      string            `json:"accept"`
	AcceptLang  string            `json:"accept_language"`
	AcceptEnc   string            `json:"accept_encoding"`
}

// Proxy represents a proxy configuration
type Proxy struct {
	ID           string      `json:"id"`
	Type         ProxyType   `json:"type"`
	Host         string      `json:"host"`
	Port         int        `json:"port"`
	Username     string      `json:"username,omitempty"`
	Password     string      `json:"password,omitempty"`
	Protocol     string      `json:"protocol"`
	Country      string      `json:"country,omitempty"`
	City         string      `json:"city,omitempty"`
	ISP          string      `json:"isp,omitempty"`
	LatencyMs    int64      `json:"latency_ms"`
	SuccessRate  float64     `json:"success_rate"`
	BandwidthMbps float64   `json:"bandwidth_mbps"`
	IsStale     bool        `json:"is_stale"`
	LastChecked  time.Time   `json:"last_checked"`
	CreatedAt    time.Time   `json:"created_at"`
	UpdatedAt    time.Time   `json:"updated_at"`
}

// ProxyPool manages a pool of proxies
type ProxyPool struct {
	mu          sync.RWMutex
	proxies     map[string]*Proxy
	httpClient  *HTTPClient
	stealthMode bool
}

// HTTPClient is a custom HTTP client with proxy and TLS support
type HTTPClient struct {
	Transport   *http.Transport
	Client      *http.Client
	Profile     *BrowserProfile
	StealthMode bool
}

// ProxyConfig holds proxy configuration
type ProxyConfig struct {
	ProxyURL        string        `json:"proxy_url"`
	ProxyType       ProxyType     `json:"proxy_type"`
	StealthMode     bool          `json:"stealth_mode"`
	BrowserProfile  string        `json:"browser_profile"`
	Timeout         time.Duration `json:"timeout"`
	MaxRetries      int          `json:"max_retries"`
	RotateOnError   bool         `json:"rotate_on_error"`
}

// Common browser profiles for TLS fingerprinting
var BrowserProfiles = map[string]*BrowserProfile{
	"chrome": {
		Name:        "Chrome 126",
		JA3:         "771,4865-4866-4865-49195-49199-49196-49171-156-157-47-53,0-23-65281-10-11-35-16-5-13-45-28-21,29-23-24-25-30-27-32-31-33-34-35-36-37-38-39-40-41-42-43-44-45-46-51-52-53-57-58-65-13-45-16-18-22-19,0-10-11",
		TLSVersion:  "1.3",
		CipherSuites: []string{
			"0x1301", "0x1302", "0x1303", "0xc02b", "0xc02f", "0xc02c", "0xc030",
		},
		Extensions: []int{0, 10, 13, 16, 23, 43, 45, 51, 65037},
		ALPN:       []string{"h2", "http/1.1"},
		UserAgent:   "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/126.0.0.0 Safari/537.36",
		Accept:      "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8",
		AcceptLang:  "en-US,en;q=0.9",
		AcceptEnc:   "gzip, deflate, br",
	},
	"firefox": {
		Name:        "Firefox 128",
		JA3:         "771,4865-4866-4867-49195-49199-52393-52392-49196-49200-49195-49199-49162-49172-35-162-51-57-156-157-47-53,0-23-65281-10-11-35-16-5-13-45-28-21-43-27-24-25,29-23-24-25-30-31-33-34-35-37-38-39-40-41-42-43-44-45-46-51-52-53-57-58-65-13-45-16-18-22-19-26-27-32-33-35-40-41-42-43-50-51-45-46-49-53-65,0-10-11-13-15-21-23-27-28-33-41-45-51-57-58-59-64-85-107-129-130-131-133-134-135-136-137",
		TLSVersion:  "1.3",
		CipherSuites: []string{
			"0x1301", "0x1302", "0x1303", "0xc02c", "0xc030", "0xc02b", "0xc02f",
		},
		Extensions: []int{0, 5, 10, 11, 13, 15, 16, 21, 23, 27, 28, 33, 34, 35, 40, 41, 43, 44, 45, 50, 51, 53, 57, 58, 59, 64, 65, 85, 107, 129, 130, 131, 133, 134, 135, 136, 137},
		ALPN:       []string{"h2", "http/1.1"},
		UserAgent:   "Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:128.0) Gecko/20100101 Firefox/128.0",
		Accept:      "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,*/*;q=0.8",
		AcceptLang:  "en-US,en;q=0.5",
		AcceptEnc:   "gzip, deflate, br",
	},
	"safari": {
		Name:        "Safari 18",
		JA3:         "771,4865-4866-4867-49195-49199-52393-49196-49200-49195-49199-49162-49172-10,0-23-65281-10-11-16-5-13-45-28-21-43-27-24-25,29-23-24-25-30-31-33-34-35-37-38-39-40-41-42-43-44-45-46-51-52-53-57-58-65-13-21-26-45-50-51-53-65-66-67-68-69-70-71-72-73-74-75-76-77-78-79-80-81,0-10-11-13-15-16-18-19-20-21-22-23-24-25-26-27-28-29-30-31-32-33-34-35-36-37-38-39-40-41-42-43-44-45-46-47-48-49-50-51-52-53-55-56-57-58-59-60-61-62-63-64-65-66-67-68-69-70-71-72-73-74-75-76-77-78-79-80-81-82-83-84-85-86-87-88-89-90-91-92-93-94-95-96-97-98-99-100-101-102-103-104-105-106",
		TLSVersion:  "1.3",
		CipherSuites: []string{
			"0x1301", "0x1302", "0x1303", "0xc02b", "0xc02f", "0xc02c", "0xc030",
		},
		Extensions: []int{0, 5, 10, 11, 13, 16, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 32, 33, 34, 35, 36, 37, 38, 39, 40, 41, 42, 43, 44, 45, 46, 47, 48, 49, 50, 51, 52, 53, 55, 56, 57, 58, 59, 60, 61, 62, 63, 64, 65, 66, 67, 68, 69, 70, 71, 72, 73, 74, 75, 76, 77, 78, 79, 80, 81, 82, 83, 84, 85, 86, 87, 88, 89, 90, 91, 92, 93, 94, 95, 96, 97, 98, 99, 100, 101, 102, 103, 104, 105, 106},
		ALPN:       []string{"h2", "http/1.1"},
		UserAgent:   "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/18.0 Safari/605.1.15",
		Accept:      "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8",
		AcceptLang:  "en-US,en;q=0.9",
		AcceptEnc:   "gzip, deflate, br",
	},
	"edge": {
		Name:        "Edge 126",
		JA3:         "771,4865-4866-4865-49195-49199-49196-49171-156-157-47-53,0-23-65281-10-11-35-16-5-13-45-28-21,29-23-24-25-30-27-32-31-33-34-35-36-37-38-39-40-41-42-43-44-45-46-51-52-53-57-58-65-13-45-16-18-22-19,0-10-11",
		TLSVersion:  "1.3",
		CipherSuites: []string{
			"0x1301", "0x1302", "0x1303", "0xc02b", "0xc02f", "0xc02c", "0xc030",
		},
		Extensions: []int{0, 10, 13, 16, 23, 43, 45, 51, 65037},
		ALPN:       []string{"h2", "http/1.1"},
		UserAgent:   "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/126.0.0.0 Safari/537.36 Edg/126.0.0.0",
		Accept:      "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,image/apng,*/*;q=0.8",
		AcceptLang:  "en-US,en;q=0.9",
		AcceptEnc:   "gzip, deflate, br",
	},
}

// JA4 fingerprint generator
type JA4Generator struct {
	profiles map[string]*BrowserProfile
}

// NewJA4Generator creates a new JA4 generator
func NewJA4Generator() *JA4Generator {
	return &JA4Generator{
		profiles: BrowserProfiles,
	}
}

// GenerateJA4 generates a JA4 fingerprint from a browser profile
func (g *JA4Generator) GenerateJA4(profile *BrowserProfile) string {
	// JA4 format: t13d1516h2_443_xxxx
	// Simplified implementation
	version := "t13d" // TLS 1.3, draft disabled
	
	// Cipher count (2 digits)
	cipherCount := fmt.Sprintf("%02d", len(profile.CipherSuites))
	
	// TLS version
	tlsVer := "d" // 1.3
	
	// SNI present (13)
	sni := "d"
	
	// ALPN first value (15)
	alpn := "16" // h2
	
	// Sort ciphers
	cipherStr := g.sortAndJoinCiphers(profile.CipherSuites)
	
	// Hash of cipher string (16 chars)
	cipherHash := g.hashString(cipherStr)[:16]
	
	return fmt.Sprintf("%s%s%s%s_%s_%s", version, cipherCount, tlsVer, sni, alpn, cipherHash)
}

// sortAndJoinCiphers sorts and joins cipher suites
func (g *JA4Generator) sortAndJoinCiphers(ciphers []string) string {
	result := ""
	for _, c := range ciphers {
		result += c + "_"
	}
	return result
}

// hashString creates a quick hash
func (g *JA4Generator) hashString(s string) string {
	h := uint32(0)
	for i, c := range s {
		h += uint32(c) * uint32(i+1)
	}
	return fmt.Sprintf("%08x", h)
}

// GenerateJA3 generates a JA3 fingerprint from a browser profile
func (g *JA4Generator) GenerateJA3(profile *BrowserProfile) string {
	// JA3 format: version,ciphers,extensions,elliptic_curves,elliptic_curve_point_formats
	return profile.JA3
}

// NewProxyPool creates a new proxy pool
func NewProxyPool() *ProxyPool {
	return &ProxyPool{
		proxies:     make(map[string]*Proxy),
		stealthMode: false,
	}
}

// AddProxy adds a proxy to the pool
func (p *ProxyPool) AddProxy(proxy *Proxy) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if proxy.ID == "" {
		proxy.ID = uuid.New().String()
	}
	proxy.CreatedAt = time.Now()
	proxy.UpdatedAt = time.Now()

	p.proxies[proxy.ID] = proxy
}

// RemoveProxy removes a proxy from the pool
func (p *ProxyPool) RemoveProxy(id string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	delete(p.proxies, id)
}

// GetProxy returns a proxy by ID
func (p *ProxyPool) GetProxy(id string) (*Proxy, bool) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	proxy, exists := p.proxies[id]
	return proxy, exists
}

// GetAllProxies returns all proxies
func (p *ProxyPool) GetAllProxies() []*Proxy {
	p.mu.RLock()
	defer p.mu.RUnlock()

	result := make([]*Proxy, 0, len(p.proxies))
	for _, proxy := range p.proxies {
		result = append(result, proxy)
	}
	return result
}

// GetRandomProxy returns a random proxy from the pool
func (p *ProxyPool) GetRandomProxy() (*Proxy, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	proxies := make([]*Proxy, 0)
	for _, proxy := range p.proxies {
		if !proxy.IsStale {
			proxies = append(proxies, proxy)
		}
	}

	if len(proxies) == 0 {
		return nil, fmt.Errorf("no available proxies")
	}

	return proxies[rand.Intn(len(proxies))], nil
}

// GetProxyByCountry returns a proxy from a specific country
func (p *ProxyPool) GetProxyByCountry(country string) (*Proxy, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	for _, proxy := range p.proxies {
		if proxy.Country == country && !proxy.IsStale {
			return proxy, nil
		}
	}

	return nil, fmt.Errorf("no proxy found for country: %s", country)
}

// MarkProxyStale marks a proxy as stale (unavailable)
func (p *ProxyPool) MarkProxyStale(id string) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if proxy, exists := p.proxies[id]; exists {
		proxy.IsStale = true
		proxy.UpdatedAt = time.Now()
	}
}

// SetStealthMode enables or disables stealth mode
func (p *ProxyPool) SetStealthMode(enabled bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.stealthMode = enabled
}

// CreateHTTPClient creates an HTTP client with proxy and TLS settings
func (p *ProxyPool) CreateHTTPClient(profileName string) (*HTTPClient, error) {
	profile, exists := BrowserProfiles[profileName]
	if !exists {
		profile = BrowserProfiles["chrome"] // Default
	}

	transport := &http.Transport{
		MaxIdleConns:        100,
		MaxIdleConnsPerHost: 10,
		IdleConnTimeout:     90 * time.Second,
		TLSHandshakeTimeout: 10 * time.Second,
	}

	client := &http.Client{
		Transport: transport,
		Timeout:   30 * time.Second,
	}

	return &HTTPClient{
		Transport:   transport,
		Client:      client,
		Profile:     profile,
		StealthMode: p.stealthMode,
	}, nil
}

// CreateStealthTransport creates a stealth TLS config
func CreateStealthTransport(profile *BrowserProfile) *http.Transport {
	// Create custom TLS config based on browser profile
	tlsConfig := &tls.Config{
		MinVersion:       tls.VersionTLS13,
		MaxVersion:       tls.VersionTLS13,
		CurvePreferences: []tls.CurveID{tls.X25519, tls.CurveP256, tls.CurveP384},
		CipherSuites:      parseCipherSuites(profile.CipherSuites),
	}

	// Set ALPN
	if len(profile.ALPN) > 0 {
		tlsConfig.NextProtos = profile.ALPN
	}

	// Enable HTTP/2
	tlsConfig.SessionTicketsDisabled = false

	return &http.Transport{
		TLSClientConfig: tlsConfig,
		MaxIdleConns:    100,
		IdleConnTimeout: 90 * time.Second,
		// Proxy settings would be configured here
	}
}

// parseCipherSuites converts cipher suite strings to tls.CipherSuites
func parseCipherSuites(suites []string) []uint16 {
	var result []uint16
	for _, s := range suites {
		// Remove 0x prefix if present
		s = strip0x(s)
		var id uint16
		fmt.Sscanf(s, "%d", &id)
		if id != 0 {
			result = append(result, id)
		}
	}
	return result
}

// strip0x removes 0x prefix from hex string
func strip0x(s string) string {
	if len(s) > 2 && s[:2] == "0x" {
		return s[2:]
	}
	return s
}

// CheckProxy checks if a proxy is working
func (p *Proxy) CheckProxy(ctx context.Context) error {
	start := time.Now()

	proxyAddr := fmt.Sprintf("%s:%d", p.Host, p.Port)
	
	switch p.Type {
	case ProxyTypeSOCKS5:
		// SOCKS5 proxy check
		conn, err := net.DialTimeout("tcp", proxyAddr, 5*time.Second)
		if err != nil {
			return err
		}
		defer conn.Close()
	case ProxyTypeHTTP, ProxyTypeHTTPS, ProxyTypeHTTPConnect:
		// HTTP/HTTPS proxy check
		conn, err := net.DialTimeout("tcp", proxyAddr, 5*time.Second)
		if err != nil {
			return err
		}
		defer conn.Close()
	default:
		return fmt.Errorf("unsupported proxy type: %s", p.Type)
	}

	p.LatencyMs = time.Since(start).Milliseconds()
	p.LastChecked = time.Now()
	p.SuccessRate = 1.0 // Success if no error

	return nil
}

// StealthHTTPRequest performs an HTTP request with TLS stealth
func (c *HTTPClient) StealthHTTPRequest(ctx context.Context, method, url string, headers map[string]string) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, method, url, nil)
	if err != nil {
		return nil, err
	}

	// Apply browser profile headers
	if c.Profile != nil {
		req.Header.Set("User-Agent", c.Profile.UserAgent)
		req.Header.Set("Accept", c.Profile.Accept)
		req.Header.Set("Accept-Language", c.Profile.AcceptLang)
		req.Header.Set("Accept-Encoding", c.Profile.AcceptEnc)
		req.Header.Set("Sec-Ch-Ua", `"Chromium";v="126", "Not.A/Brand";v="24"`)
		req.Header.Set("Sec-Ch-Ua-Mobile", "?0")
		req.Header.Set("Sec-Ch-Ua-Platform", `"Windows"`)
		req.Header.Set("Sec-Fetch-Dest", "empty")
		req.Header.Set("Sec-Fetch-Mode", "cors")
		req.Header.Set("Sec-Fetch-Site", "same-origin")
	}

	// Apply custom headers
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	// Randomize some headers for stealth
	if c.StealthMode {
		c.randomizeHeaders(req)
	}

	return c.Client.Do(req)
}

// randomizeHeaders adds slight randomization to headers
func (c *HTTPClient) randomizeHeaders(req *http.Request) {
	// Add slight variations to User-Agent
	ua := req.UserAgent()
	if ua != "" {
		// Minor version variation
		ua = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/126.0." + 
			fmt.Sprintf("%d", 6400+rand.Intn(100)) + " Safari/537.36"
		req.Header.Set("User-Agent", ua)
	}
}

// Stats returns proxy pool statistics
type ProxyPoolStats struct {
	TotalProxies  int     `json:"total_proxies"`
	ActiveProxies int     `json:"active_proxies"`
	StaleProxies  int     `json:"stale_proxies"`
	AverageLatency float64 `json:"average_latency_ms"`
	AverageSuccess float64 `json:"average_success_rate"`
}

// GetStats returns proxy pool statistics
func (p *ProxyPool) GetStats() ProxyPoolStats {
	p.mu.RLock()
	defer p.mu.RUnlock()

	stats := ProxyPoolStats{
		TotalProxies: len(p.proxies),
	}

	var totalLatency int64
	var totalSuccess float64
	var activeCount int

	for _, proxy := range p.proxies {
		if proxy.IsStale {
			stats.StaleProxies++
		} else {
			activeCount++
			totalLatency += proxy.LatencyMs
			totalSuccess += proxy.SuccessRate
		}
	}

	stats.ActiveProxies = activeCount

	if activeCount > 0 {
		stats.AverageLatency = float64(totalLatency) / float64(activeCount)
		stats.AverageSuccess = totalSuccess / float64(activeCount)
	}

	return stats
}

// LoadFromURL parses a proxy URL and creates a proxy
func ParseProxyURL(proxyURL string) (*Proxy, error) {
	u, err := url.Parse(proxyURL)
	if err != nil {
		return nil, err
	}

	proxy := &Proxy{
		ID:    uuid.New().String(),
		Host:  u.Hostname(),
		Port:  0,
		Protocol: u.Scheme,
	}

	// Parse port
	if portStr := u.Port(); portStr != "" {
		fmt.Sscanf(portStr, "%d", &proxy.Port)
	} else {
		// Default ports
		switch u.Scheme {
		case "http", "http_connect":
			proxy.Port = 8080
			proxy.Type = ProxyTypeHTTP
		case "https":
			proxy.Port = 8443
			proxy.Type = ProxyTypeHTTPS
		case "socks5":
			proxy.Port = 1080
			proxy.Type = ProxyTypeSOCKS5
		}
	}

	// Parse credentials
	if u.User != nil {
		proxy.Username = u.User.Username()
		proxy.Password, _ = u.User.Password()
	}

	return proxy, nil
}

// Ensure types are used
var _ = tls.Config{}
