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

type LoadBalancer struct {
	backend []*Backend
	current uint64
}

func main() {
	serverList := []string{
		"http://localhost:3001",
		"http://localhost:3002",
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

		lb.backend = append(lb.backend, backend)
		log.Printf("Added backend server: %s", backend.URL.String())
	}

}
