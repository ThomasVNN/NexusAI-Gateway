package httpclient

import (
	"net/http"
	"sync"
	"time"
)

// ConnectionPool manages reusable HTTP transports
type ConnectionPool struct {
	mu         sync.RWMutex
	transports map[string]*http.Transport
	poolSize   int
}

// NewConnectionPool creates a new connection pool
func NewConnectionPool(poolSize int) *ConnectionPool {
	return &ConnectionPool{
		transports: make(map[string]*http.Transport),
		poolSize:   poolSize,
	}
}

// GetTransport returns a transport for the given host
func (p *ConnectionPool) GetTransport(host string) *http.Transport {
	p.mu.RLock()
	transport, exists := p.transports[host]
	p.mu.RUnlock()

	if exists {
		return transport
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	// Double-check after acquiring write lock
	if transport, exists = p.transports[host]; exists {
		return transport
	}

	transport = &http.Transport{
		MaxIdleConns:        p.poolSize,
		MaxIdleConnsPerHost: p.poolSize,
		IdleConnTimeout:     90 * time.Second,
		DisableKeepAlives:   false,
		MaxConnsPerHost:     p.poolSize * 2,
	}

	p.transports[host] = transport
	return transport
}

// Close closes all transports in the pool
func (p *ConnectionPool) Close() {
	p.mu.Lock()
	defer p.mu.Unlock()

	for _, transport := range p.transports {
		transport.CloseIdleConnections()
	}
	p.transports = make(map[string]*http.Transport)
}

// PoolStats returns statistics about the pool
func (p *ConnectionPool) Stats() map[string]int {
	p.mu.RLock()
	defer p.mu.RUnlock()

	return map[string]int{
		"total_transports": len(p.transports),
		"pool_size":        p.poolSize,
	}
}
