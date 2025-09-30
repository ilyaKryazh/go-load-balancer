// from article https://kasvith.me/posts/lets-create-a-simple-lb-go/
package main

import (
	"fmt"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"sync"
	"sync/atomic"
)

type Backend struct {
	URL          *url.URL
	Alive        bool
	mux          sync.RWMutex
	ReverseProxy *httputil.ReverseProxy
}

type ServerPool struct {
	backends []*Backend
	current  uint64
}

var serverPool ServerPool

func main() {
	u, _ := url.Parse("http://localhost:8080")
	rp := httputil.NewSingleHostReverseProxy(u)
	http.HandleFunc("/", rp.ServeHTTP)
	http.ListenAndServe(":3000", nil)

	targets := []string{
		"http://localhost:8081",
		"http://localhost:8082",
		"http://localhost:8083",
	}

	initializeServerPool(targets)

	// register handler
	http.HandleFunc("/", lb)

	// start server
	port := 3000
	server := &http.Server{
		Addr: fmt.Sprintf(":%d", port),
	}

	fmt.Printf("Load balancer started at %s\n", server.Addr)
	if err := server.ListenAndServe(); err != nil {
		log.Fatalf("server failed: %v", err)
	}
}

func initializeServerPool(targets []string) {
	for _, t := range targets {
		u, _ := url.Parse(t)
		proxy := httputil.NewSingleHostReverseProxy(u)
		backend := &Backend{
			URL:          u,
			Alive:        true,
			ReverseProxy: proxy,
		}
		serverPool.AddBackend(backend)
	}
}

func lb(w http.ResponseWriter, r *http.Request) {
	peer := serverPool.GetNextPeer()
	if peer != nil {
		peer.ReverseProxy.ServeHTTP(w, r)
		return
	}
	http.Error(w, "Service not available", http.StatusServiceUnavailable)
}

func (s *ServerPool) AddBackend(backend *Backend) {
	s.backends = append(s.backends, backend)
}

func (s *ServerPool) NextIndex() int {
	return int(atomic.AddUint64(&s.current, uint64(1)%uint64(len(s.backends))))
}

func (s *ServerPool) GetNextPeer() *Backend {
	next := s.NextIndex()
	l := len(s.backends) + next
	for i := next; i < l; i++ {
		idx := i % len(s.backends)

		if s.backends[idx].isAlive() {
			if i != next {
				atomic.StoreUint64(&s.current, uint64(idx))
			}
			return s.backends[idx]
		}
	}
	return nil
}

func (b *Backend) setAlive(alive bool) {
	b.mux.Lock()
	b.Alive = alive
	b.mux.Unlock()
}

func (b *Backend) isAlive() (alive bool) {
	b.mux.Lock()
	alive = b.Alive
	b.mux.RUnlock()
	return alive
}
