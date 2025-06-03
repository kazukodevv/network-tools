.PHONY: build test clean docker

build:
	go build -o bin/dns-server ./cmd/dns-server

run:
	./bin/dns-server

dev:
	go run ./cmd/dns-server