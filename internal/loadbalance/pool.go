package loadbalance

import (
	"sync"
	"sync/atomic"
)

// Strategy defines load balancing strategy
type Strategy string

const (
	// RoundRobin distributes requests evenly
	RoundRobin Strategy = "round_robin"
	// LeastConnections routes to server with least connections
	LeastConnections Strategy = "least_connections"
	// Random randomly selects a server
	Random Strategy = "random"
	// Weighted distributes based on weight
	Weighted Strategy = "weighted"
	// LeastLatency routes to server with lowest latency
	LeastLatency Strategy = "least_latency"
)

// Server represents a backend server
type Server struct {
	Address      string
	Weight       int
	Connections  int64
	LatencyMs    int64
	Healthy      bool
	FailureCount int64
	LastFailure  int64
}

// Pool manages a pool of servers
type Pool struct {
	mu       sync.RWMutex
	servers  []*Server
	strategy Strategy
	index    uint32
}

// NewPool creates a new server pool
func NewPool(strategy Strategy) *Pool {
	return &Pool{
		servers:  make([]*Server, 0),
		strategy: strategy,
		index:    0,
	}
}

// AddServer adds a server to the pool
func (p *Pool) AddServer(address string, weight int) *Server {
	p.mu.Lock()
	defer p.mu.Unlock()

	server := &Server{
		Address: address,
		Weight:  weight,
		Healthy: true,
	}

	if weight == 0 {
		server.Weight = 1
	}

	p.servers = append(p.servers, server)
	return server
}

// RemoveServer removes a server from the pool
func (p *Pool) RemoveServer(address string) {
	p.mu.Lock()
	defer p.mu.Unlock()

	newServers := make([]*Server, 0)
	for _, s := range p.servers {
		if s.Address != address {
			newServers = append(newServers, s)
		}
	}
	p.servers = newServers
}

// Select selects a server based on the strategy
func (p *Pool) Select() *Server {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if len(p.servers) == 0 {
		return nil
	}

	var server *Server

	switch p.strategy {
	case RoundRobin:
		server = p.selectRoundRobin()
	case LeastConnections:
		server = p.selectLeastConnections()
	case Random:
		server = p.selectRandom()
	case Weighted:
		server = p.selectWeighted()
	case LeastLatency:
		server = p.selectLeastLatency()
	default:
		server = p.selectRoundRobin()
	}

	if server != nil {
		atomic.AddInt64(&server.Connections, 1)
	}

	return server
}

// Release releases a server connection
func (p *Pool) Release(server *Server) {
	if server != nil {
		atomic.AddInt64(&server.Connections, -1)
	}
}

// selectRoundRobin selects the next server in rotation
func (p *Pool) selectRoundRobin() *Server {
	if len(p.servers) == 0 {
		return nil
	}

	idx := atomic.AddUint32(&p.index, 1) % uint32(len(p.servers))
	return p.servers[idx]
}

// selectLeastConnections selects server with least connections
func (p *Pool) selectLeastConnections() *Server {
	var minConnections int64 = -1
	var selected *Server

	for _, s := range p.servers {
		if !s.Healthy {
			continue
		}

		conns := atomic.LoadInt64(&s.Connections)
		if minConnections == -1 || conns < minConnections {
			minConnections = conns
			selected = s
		}
	}

	return selected
}

// selectRandom randomly selects a server
func (p *Pool) selectRandom() *Server {
	// Simple hash-based random
	idx := int(atomic.AddUint32(&p.index, 1)) % len(p.servers)
	return p.servers[idx]
}

// selectWeighted selects based on weight
func (p *Pool) selectWeighted() *Server {
	totalWeight := 0
	for _, s := range p.servers {
		if s.Healthy {
			totalWeight += s.Weight
		}
	}

	if totalWeight == 0 {
		return nil
	}

	// Pick random weight
	idx := int(atomic.AddUint32(&p.index, 1)) % totalWeight

	current := 0
	for _, s := range p.servers {
		if !s.Healthy {
			continue
		}

		current += s.Weight
		if idx < current {
			return s
		}
	}

	return p.servers[0]
}

// selectLeastLatency selects server with lowest latency
func (p *Pool) selectLeastLatency() *Server {
	var minLatency int64 = -1
	var selected *Server

	for _, s := range p.servers {
		if !s.Healthy {
			continue
		}

		latency := atomic.LoadInt64(&s.LatencyMs)
		if minLatency == -1 || latency < minLatency {
			minLatency = latency
			selected = s
		}
	}

	return selected
}

// UpdateLatency updates server latency
func (p *Pool) UpdateLatency(address string, latencyMs int64) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	for _, s := range p.servers {
		if s.Address == address {
			atomic.StoreInt64(&s.LatencyMs, latencyMs)
			break
		}
	}
}

// RecordFailure records a failure for a server
func (p *Pool) RecordFailure(address string) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	for _, s := range p.servers {
		if s.Address == address {
			failures := atomic.AddInt64(&s.FailureCount, 1)
			if failures >= 3 {
				s.Healthy = false
			}
			break
		}
	}
}

// RecordSuccess records a success for a server
func (p *Pool) RecordSuccess(address string) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	for _, s := range p.servers {
		if s.Address == address {
			atomic.StoreInt64(&s.FailureCount, 0)
			if !s.Healthy {
				s.Healthy = true
			}
			break
		}
	}
}

// GetServers returns all servers
func (p *Pool) GetServers() []*Server {
	p.mu.RLock()
	defer p.mu.RUnlock()

	servers := make([]*Server, len(p.servers))
	copy(servers, p.servers)
	return servers
}

// GetHealthyServers returns only healthy servers
func (p *Pool) GetHealthyServers() []*Server {
	p.mu.RLock()
	defer p.mu.RUnlock()

	var healthy []*Server
	for _, s := range p.servers {
		if s.Healthy {
			healthy = append(healthy, s)
		}
	}
	return healthy
}

// ServerStats returns statistics for all servers
func (p *Pool) ServerStats() []map[string]interface{} {
	p.mu.RLock()
	defer p.mu.RUnlock()

	stats := make([]map[string]interface{}, len(p.servers))
	for i, s := range p.servers {
		stats[i] = map[string]interface{}{
			"address":       s.Address,
			"weight":        s.Weight,
			"connections":   atomic.LoadInt64(&s.Connections),
			"latency_ms":    atomic.LoadInt64(&s.LatencyMs),
			"healthy":       s.Healthy,
			"failure_count": atomic.LoadInt64(&s.FailureCount),
		}
	}
	return stats
}
