package main

import (
	"context"
	"fmt"
	"log"
	"net"
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

type ctxKey int

const (
	Attempts ctxKey = iota
	Retry
)

func main() {
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

	go healthCheck()

	fmt.Printf("Load balancer started at %s\n", server.Addr)
	if err := server.ListenAndServe(); err != nil {
		log.Fatalf("server failed: %v", err)
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

func initializeServerPool(targets []string) {
	for _, t := range targets {
		u, _ := url.Parse(t)
		proxy := httputil.NewSingleHostReverseProxy(u)
		backend := &Backend{
			URL:          u,
			ReverseProxy: proxy,
		}
		b := backend
		p := proxy
		proxy.ErrorHandler = func(writter http.ResponseWriter, req *http.Request, e error) {
			log.Printf("[%s] %s\n", b.URL, e.Error())
			retries := GetRetryFromContext(req)
			attempts := GetAttemptsFromContext(req)
			if retries < 3 {
				time.Sleep(10 * time.Millisecond)
				ctx := context.WithValue(req.Context(), Retry, retries+1)
				p.ServeHTTP(writter, req.WithContext(ctx))
				return
			}

			serverPool.MarkServiceDown(backend.URL)
			// if the same rN(next
			ctx := context.WithValue(req.Context(), Attempts, attempts+1)
			lb(writter, req.WithContext(ctx))
		}
		backend.setAlive(true)
		serverPool.AddBackend(backend)
	}
}

func GetRetryFromContext(req *http.Request) int {
	if retries, ok := req.Context().Value(Retry).(int); ok {
		return retries
	}
	return 0
}

func GetAttemptsFromContext(req *http.Request) int {
	if attemps, ok := req.Context().Value(Attempts).(int); ok {
		return attemps
	}
	return 0
}

func (s *ServerPool) MarkServiceDown(u *url.URL) {
	for i := range s.backends {
		if s.backends[i].URL.String() == u.String() {
			s.backends[i].setAlive(false)
		}
	}
}

func (s *ServerPool) NextIndex() int {
	return int(atomic.AddUint64(&s.current, uint64(1)) % uint64(len(s.backends)))
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

func (s *ServerPool) HealthCheck() {
	for _, b := range s.backends {
		status := "up"
		alive := isBackendAlive(b.URL)
		b.setAlive(alive)
		if !alive {
			status = "down"
		}
		log.Printf("%s [%s]\n", b.URL, status)
	}
}

func (b *Backend) setAlive(alive bool) {
	b.mux.Lock()
	b.Alive = alive
	b.mux.Unlock()
}

func (b *Backend) isAlive() (alive bool) {
	b.mux.Lock()
	alive = b.Alive
	b.mux.Unlock()
	return alive
}

func isBackendAlive(url *url.URL) bool {
	timeout := 2 * time.Second
	conn, err := net.DialTimeout("tcp", url.Host, timeout)
	if err != nil {
		log.Println("Server unreachable, error: ", err)
		return false
	}
	conn.Close()
	return true
}

func healthCheck() {
	t := time.NewTicker(time.Second * 20)
	for range t.C {
		log.Println("Start healthcheck")
		serverPool.HealthCheck()
		log.Println("Healthcheck completed")
	}
}
