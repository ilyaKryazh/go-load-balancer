// from article https://kasvith.me/posts/lets-create-a-simple-lb-go/
package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"sync"
	"sync/atomic"
	"time"
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
	rp.ErrorHandler = func(writer http.ResponseWriter, request *http.Request, e error) {
		log.Printf("[%s] %s\n", serverUrl.Host, e.Error())
		retries := GetRetryFromContext(request)
		if retries < 3 {
			select {
			case <-time.After(10 * time.Millisecond):
				ctx := context.WithValue(request.Context(), Retry, retries+1)
				proxy.ServeHTTP(writer, request.WithContext(ctx))
			}
			return
		}

		// after 3 retries, mark this backend as down
		serverPool.MarkBackendStatus(serverUrl, false)

		// if the same request routing for few attempts with different backends, increase the count
		attempts := GetAttemptsFromContext(request)
		log.Printf("%s(%s) Attempting retry %d\n", request.RemoteAddr, request.URL.Path, attempts)
		ctx := context.WithValue(request.Context(), Attempts, attempts+1)
		lb(writer, request.WithContext(ctx))
	}
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

const (
	Attempts int = iota
	Retry
)

// GetAttemptsFromContext returns the attempts for request
func GetRetryFromContext(r *http.Request) int {
	if retry, ok := r.Context().Value(Retry).(int); ok {
		return retry
	}
	return 0
}
