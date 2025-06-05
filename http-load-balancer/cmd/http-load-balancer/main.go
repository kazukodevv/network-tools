package main

import (
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
	mu           sync.RWMutex
	ReverseProxy *httputil.ReverseProxy
}

func (b *Backend) IsAlive() bool {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.Alive
}

func (b *Backend) SetAlive(alive bool) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.Alive = alive
}

type LoadBalancer struct {
	backends []*Backend
	current  uint64
}

func (lb *LoadBalancer) AddBackend(backend *Backend) {
	lb.backends = append(lb.backends, backend)
}

// NextIndex returns the index of the next backend server in a round-robin fashion.
func (lb *LoadBalancer) NextIndex() int {
	return int(atomic.AddUint64(&lb.current, 1) % uint64(len(lb.backends)))
}

func (lb *LoadBalancer) GetNextPeer() *Backend {
	next := lb.NextIndex()
	l := len(lb.backends) + next

	for i := next; i < l; i++ {
		idx := i % len(lb.backends)
		if lb.backends[idx].IsAlive() {
			if i != next {
				atomic.StoreUint64(&lb.current, uint64(idx))
			}
			return lb.backends[idx]
		}
	}
	return nil
}

func (lb *LoadBalancer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	peer := lb.GetNextPeer()
	if peer == nil {
		peer.ReverseProxy.ServeHTTP(w, r)
		return
	}
	http.Error(w, "No available backend servers", http.StatusServiceUnavailable)
}

func isBackendAlive(url *url.URL) bool {
	conn, err := http.Get(url.String())
	if err != nil {
		return false
	}
	defer conn.Body.Close()
	return conn.StatusCode == 200
}

// healthCheck performs periodic health checks on all backends
func healthCheck(lb *LoadBalancer) {
	t := time.NewTicker(time.Second * 10)
	defer t.Stop()

	for {
		select {
		case <-t.C:
			log.Println("Starting health check...")
			for _, backend := range lb.backends {
				alive := isBackendAlive(backend.URL)
				backend.SetAlive(alive)
				status := "UP"
				if !alive {
					status = "DOWN"
				}
				log.Printf("Backend %s is %s", backend.URL.String(), status)
			}
		}
	}
}

func main() {
	serverList := []string{
		"http://localhost:3001",
	}

	lb := &LoadBalancer{}

	for _, server := range serverList {
		serverURL, err := url.Parse(server)
		if err != nil {
			log.Fatalf("Failed to parse server URL %s: %v", server, err)
		}

		proxy := httputil.NewSingleHostReverseProxy(serverURL)

		proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
			log.Printf("Error proxying request to %s: %v", serverURL.String(), err)
		}

		backend := &Backend{
			URL:          serverURL,
			Alive:        true,
			ReverseProxy: proxy,
		}

		lb.AddBackend(backend)
		log.Printf("Added backend server: %s", backend.URL.String())
	}

	go healthCheck(lb)

	server := http.Server{
		Addr:    ":8080",
		Handler: lb,
	}

	log.Println("Starting load balancer on :8080")
	if err := server.ListenAndServe(); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
