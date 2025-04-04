package http

import (
	"net/http"
	"sync"
)

// Mux represents an HTTP muxer that can handle HTTP methods.
type Mux interface {
	Handle(pattern string, handler http.Handler)
	HandleFunc(pattern string, handler func(http.ResponseWriter, *http.Request))
	ServeHTTP(w http.ResponseWriter, r *http.Request)
}

// MWMux applies middleware when registering HTTP handlers.
type MWMux struct {
	mux Mux

	middlewares []func(http.Handler) http.Handler
	mu          sync.RWMutex
}

// NewMWMux creates a new MWMux using mux.
func NewMWMux(mux Mux) *MWMux {
	return &MWMux{mux: mux}
}

// Use adds middlewares to be applied when registering HTTP handlers.
func (m *MWMux) Use(middlewares ...func(http.Handler) http.Handler) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.middlewares = append(m.middlewares, middlewares...)
}

// Handle layers middlewares around handler and registers them with pattern.
func (m *MWMux) Handle(pattern string, handler http.Handler) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// assemble middleware in descending order
	for i := len(m.middlewares) - 1; i >= 0; i-- {
		handler = m.middlewares[i](handler)
	}

	m.mux.Handle(pattern, handler)
}

// HandleFunc layers middlewares around handler and registers them with pattern.
func (m *MWMux) HandleFunc(pattern string, handler func(http.ResponseWriter, *http.Request)) {
	m.Handle(pattern, http.HandlerFunc(handler))
}

// ServeHTTP is a convenience wrapper to use m itself as an HTTP handler.
func (m *MWMux) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	m.mux.ServeHTTP(w, r)
}
