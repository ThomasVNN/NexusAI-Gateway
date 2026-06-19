package httpclient

import (
	"crypto/tls"
	"net"
	"net/http"
	"sync"
	"time"
)

// MultiplexedTransport manages connection multiplexing for HTTP requests
type MultiplexedTransport struct {
	mu       sync.RWMutex
	pools    map[string]*connPool
	poolSize int
	defaults *http.Transport
}

// connPool represents a connection pool for a specific host
type connPool struct {
	mu       sync.Mutex
	conns    []*persistentConn
	maxConns int
	host     string
}

// persistentConn represents a persistent connection
type persistentConn struct {
	conn    net.Conn
	reqs    int64
	created time.Time
	mu      sync.Mutex
	inUse   bool
}

// NewMultiplexedTransport creates a new multiplexed transport
func NewMultiplexedTransport(poolSize int) *MultiplexedTransport {
	return &MultiplexedTransport{
		pools:    make(map[string]*connPool),
		poolSize: poolSize,
		defaults: &http.Transport{
			MaxIdleConns:        poolSize * 10,
			MaxIdleConnsPerHost: poolSize,
			IdleConnTimeout:     90 * time.Second,
		},
	}
}

// GetTransport returns a transport for the given request
func (t *MultiplexedTransport) GetTransport(req *http.Request) *http.Transport {
	return t.defaults
}

// AcquireConnection acquires a connection from the pool
func (t *MultiplexedTransport) AcquireConnection(host string) (*persistentConn, bool) {
	t.mu.RLock()
	pool, exists := t.pools[host]
	t.mu.RUnlock()

	if !exists {
		return nil, false
	}

	pool.mu.Lock()
	defer pool.mu.Unlock()

	for _, conn := range pool.conns {
		conn.mu.Lock()
		if !conn.inUse {
			conn.inUse = true
			conn.reqs++
			conn.mu.Unlock()
			return conn, true
		}
		conn.mu.Unlock()
	}

	return nil, false
}

// ReleaseConnection releases a connection back to the pool
func (t *MultiplexedTransport) ReleaseConnection(host string, conn *persistentConn) {
	if conn == nil {
		return
	}

	conn.mu.Lock()
	conn.inUse = false
	conn.mu.Unlock()
}

// AddConnection adds a new connection to the pool
func (t *MultiplexedTransport) AddConnection(host string, conn net.Conn) *persistentConn {
	t.mu.RLock()
	pool, exists := t.pools[host]
	t.mu.RUnlock()

	if !exists {
		t.mu.Lock()
		pool, exists = t.pools[host]
		if !exists {
			pool = &connPool{
				maxConns: t.poolSize,
				host:     host,
			}
			t.pools[host] = pool
		}
		t.mu.Unlock()
	}

	pool.mu.Lock()
	defer pool.mu.Unlock()

	// Check if pool is full
	if len(pool.conns) >= pool.maxConns {
		// Remove oldest connection
		if len(pool.conns) > 0 {
			oldest := pool.conns[0]
			pool.conns = pool.conns[1:]
			oldest.conn.Close()
		}
	}

	pConn := &persistentConn{
		conn:    conn,
		created: time.Now(),
		inUse:   true,
	}
	pool.conns = append(pool.conns, pConn)
	return pConn
}

// CloseIdleConnections closes all idle connections
func (t *MultiplexedTransport) CloseIdleConnections() {
	t.mu.Lock()
	defer t.mu.Unlock()

	for _, pool := range t.pools {
		pool.mu.Lock()
		for _, conn := range pool.conns {
			conn.mu.Lock()
			if !conn.inUse {
				conn.conn.Close()
			}
			conn.mu.Unlock()
		}
		// Remove closed connections
		var active []*persistentConn
		for _, conn := range pool.conns {
			conn.mu.Lock()
			if conn.conn != nil {
				active = append(active, conn)
			}
			conn.mu.Unlock()
		}
		pool.conns = active
		pool.mu.Unlock()
	}
}

// Stats returns connection pool statistics
func (t *MultiplexedTransport) Stats() MultiplexStats {
	t.mu.RLock()
	defer t.mu.RUnlock()

	stats := MultiplexStats{
		Pools: make(map[string]PoolStats),
	}

	for host, pool := range t.pools {
		pool.mu.Lock()
		var total, inUse int64
		for _, conn := range pool.conns {
			conn.mu.Lock()
			total++
			if conn.inUse {
				inUse++
			}
			conn.mu.Unlock()
		}
		stats.Pools[host] = PoolStats{
			TotalConnections: int(total),
			InUseConnections: int(inUse),
			IdleConnections:  int(total - inUse),
		}
		pool.mu.Unlock()
	}

	return stats
}

// MultiplexStats holds statistics for all connection pools
type MultiplexStats struct {
	Pools map[string]PoolStats
}

// PoolStats holds statistics for a single connection pool
type PoolStats struct {
	TotalConnections int
	InUseConnections int
	IdleConnections  int
}

// NewTLSConfig creates a new TLS configuration
func NewTLSConfig(insecureSkipVerify bool) *tls.Config {
	return &tls.Config{
		InsecureSkipVerify: insecureSkipVerify,
		MinVersion:         tls.VersionTLS12,
	}
}

// DialContext creates a connection with the given dialer
func DialContext(network, addr string, timeout time.Duration) (net.Conn, error) {
	dialer := &net.Dialer{
		Timeout: timeout,
	}
	return dialer.Dial(network, addr)
}
