package main

import (
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"dns-server/internal/dns"
)

func main() {
	// Initialize structured logger
	handler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level:     slog.LevelDebug,
		AddSource: true,
	})
	logger := slog.New(handler)
	slog.SetDefault(logger)

	// Create and start DNS server
	server := dns.NewServer(dns.DNS_PORT, logger)

	// Set up signal handling for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Start server in a goroutine
	go func() {
		if err := server.Start(); err != nil {
			logger.Error("Failed to start DNS server", "error", err)
			os.Exit(1)
		}
	}()

	// Wait for shutdown signal
	<-sigChan
	logger.Info("Shutting down DNS server...")

	if err := server.Stop(); err != nil {
		logger.Error("Error stopping server", "error", err)
		os.Exit(1)
	}

	logger.Info("DNS server stopped")
}
