package main

import (
	"log"
	"net/http/httputil"
	"net/url"
	"sync"
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
	backend []*Backend
	current uint64
}

func (lb *LoadBalancer) AddBackend(backend *Backend) {
	lb.backend = append(lb.backend, backend)
}

func main() {
	serverList := []string{
		"http://localhost:3001",
	}

	lb := &LoadBalancer{}

	for _, server := range serverList {
		serverURL, err := url.Parse(server)
		if err != nil {
			panic(err)
		}

		proxy := httputil.NewSingleHostReverseProxy(serverURL)

		backend := &Backend{
			URL:          serverURL,
			Alive:        true,
			ReverseProxy: proxy,
		}

		lb.AddBackend(backend)
		log.Printf("Added backend server: %s", backend.URL.String())
	}
}
