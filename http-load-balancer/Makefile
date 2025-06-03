.PHONY: build test clean docker

build:
	go build -o bin/http-load-balancer ./cmd/http-load-balancer

run:
	./bin/http-load-balancer

dev:
	go run ./cmd/http-load-balancer