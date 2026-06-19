package graceful

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

// Server manages graceful HTTP server shutdown
type Server struct {
	server  *http.Server
	timeout time.Duration
}

// NewServer creates a new graceful server
func NewServer(addr string, handler http.Handler, timeout time.Duration) *Server {
	if timeout == 0 {
		timeout = 30 * time.Second
	}

	return &Server{
		server: &http.Server{
			Addr:         addr,
			Handler:      handler,
			ReadTimeout:  30 * time.Second,
			WriteTimeout: 30 * time.Second,
			IdleTimeout:  60 * time.Second,
		},
		timeout: timeout,
	}
}

// Start starts the server and handles graceful shutdown
func (s *Server) Start() error {
	// Channel to receive OS signals
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	// Start server in goroutine
	go func() {
		if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			panic("Server error: " + err.Error())
		}
	}()

	// Wait for shutdown signal
	<-stop
	return s.Shutdown()
}

// Shutdown gracefully shuts down the server
func (s *Server) Shutdown() error {
	ctx, cancel := context.WithTimeout(context.Background(), s.timeout)
	defer cancel()

	if err := s.server.Shutdown(ctx); err != nil {
		return err
	}
	return nil
}

// WithTimeout wraps an HTTP handler with timeout handling
func WithTimeout(handler http.Handler, timeout time.Duration) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), timeout)
		defer cancel()

		// Create a response writer that handles timeout
		done := make(chan struct{})
		tw := &timeoutWriter{ResponseWriter: w, done: done}

		go func() {
			handler.ServeHTTP(tw, r.WithContext(ctx))
			close(done)
		}()

		select {
		case <-done:
			// Request completed normally
		case <-ctx.Done():
			// Timeout occurred
			tw.WriteHeader(http.StatusRequestTimeout)
		}
	})
}

// timeoutWriter wraps http.ResponseWriter to handle timeouts
type timeoutWriter struct {
	http.ResponseWriter
	done chan struct{}
}

func (tw *timeoutWriter) WriteHeader(code int) {
	select {
	case <-tw.done:
		// Already wrote header
	default:
		tw.ResponseWriter.WriteHeader(code)
	}
}

func (tw *timeoutWriter) Write(b []byte) (int, error) {
	select {
	case <-tw.done:
		return 0, nil
	default:
		return tw.ResponseWriter.Write(b)
	}
}

// Pool manages a pool of graceful servers
type Pool struct {
	servers []*Server
}

// NewPool creates a new server pool
func NewPool() *Pool {
	return &Pool{
		servers: make([]*Server, 0),
	}
}

// Add adds a server to the pool
func (p *Pool) Add(s *Server) {
	p.servers = append(p.servers, s)
}

// StartAll starts all servers in the pool
func (p *Pool) StartAll() error {
	for _, s := range p.servers {
		if err := s.Start(); err != nil {
			return err
		}
	}
	return nil
}

// ShutdownAll shuts down all servers in the pool
func (p *Pool) ShutdownAll() error {
	for _, s := range p.servers {
		if err := s.Shutdown(); err != nil {
			return err
		}
	}
	return nil
}

// HealthCheck creates a health check handler
func HealthCheck(handler func() bool) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if handler() {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"status":"ok"}`))
		} else {
			w.WriteHeader(http.StatusServiceUnavailable)
			w.Write([]byte(`{"status":"unavailable"}`))
		}
	})
}

// ReadinessCheck creates a readiness check handler
func ReadinessCheck(ready func() bool) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if ready() {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"ready":true}`))
		} else {
			w.WriteHeader(http.StatusServiceUnavailable)
			w.Write([]byte(`{"ready":false}`))
		}
	})
}
