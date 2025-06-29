package integration

import (
	"log/slog"
	"net"
	"os"
	"testing"
	"time"

	"dns-server/internal/dns"
)

func TestDNSServerBasicQuery(t *testing.T) {
	// Create a logger for the server
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelWarn, // Reduce noise during tests
	}))

	// Use a test port
	testPort := 8054
	server := dns.NewServer(testPort, logger)

	// Start server in background
	serverDone := make(chan bool)
	go func() {
		defer close(serverDone)
		if err := server.Start(); err != nil {
			t.Logf("Server error: %v", err)
		}
	}()

	// Give server time to start
	time.Sleep(300 * time.Millisecond)

	// Cleanup
	defer func() {
		server.Stop()
		// Wait a bit for cleanup
		time.Sleep(100 * time.Millisecond)
	}()

	// Test a simple query for www.example.com
	t.Run("valid_domain_query", func(t *testing.T) {
		query := []byte{
			0x12, 0x34, // ID
			0x01, 0x00, // Flags (standard query)
			0x00, 0x01, // QDCount (1 question)
			0x00, 0x00, // ANCount (0 answers)
			0x00, 0x00, // NSCount (0 authority)
			0x00, 0x00, // ARCount (0 additional)
			// Question section
			3, 'w', 'w', 'w', 7, 'e', 'x', 'a', 'm', 'p', 'l', 'e', 3, 'c', 'o', 'm', 0, // www.example.com
			0x00, 0x01, // Type A
			0x00, 0x01, // Class IN
		}

		// Connect to server
		serverAddr, err := net.ResolveUDPAddr("udp", ":8054")
		if err != nil {
			t.Fatalf("Error resolving server address: %v", err)
		}

		clientConn, err := net.DialUDP("udp", nil, serverAddr)
		if err != nil {
			t.Fatalf("Error connecting to server: %v", err)
		}
		defer clientConn.Close()

		// Send query
		_, err = clientConn.Write(query)
		if err != nil {
			t.Fatalf("Error sending query: %v", err)
		}

		// Read response
		response := make([]byte, dns.MESSAGE_SIZE)
		clientConn.SetReadDeadline(time.Now().Add(2 * time.Second))
		n, err := clientConn.Read(response)
		if err != nil {
			t.Fatalf("Error reading response: %v", err)
		}

		// Parse response
		responseMsg, err := dns.ParseDNSMessage(response[:n])
		if err != nil {
			t.Fatalf("Error parsing response: %v", err)
		}

		// Verify response
		if responseMsg.Header.ID != 0x1234 {
			t.Errorf("Response ID = %v, want %v", responseMsg.Header.ID, 0x1234)
		}

		if responseMsg.Header.Flags&0x8000 == 0 {
			t.Errorf("Response should have QR flag set (indicating response)")
		}

		if responseMsg.Header.ANCount != 1 {
			t.Errorf("Response ANCount = %v, want %v", responseMsg.Header.ANCount, 1)
		}

		if len(responseMsg.Answers) != 1 {
			t.Errorf("len(Response.Answers) = %v, want %v", len(responseMsg.Answers), 1)
		}

		// Check the answer
		answer := responseMsg.Answers[0]
		if answer.Type != dns.TYPE_A {
			t.Errorf("Answer.Type = %v, want %v", answer.Type, dns.TYPE_A)
		}

		expectedIP := []byte{192, 168, 1, 1}
		if len(answer.Data) != 4 {
			t.Errorf("Answer.Data length = %v, want %v", len(answer.Data), 4)
		} else {
			for i, b := range expectedIP {
				if answer.Data[i] != b {
					t.Errorf("Answer.Data[%d] = %v, want %v", i, answer.Data[i], b)
				}
			}
		}
	})

	// Test NXDOMAIN response
	t.Run("nxdomain_query", func(t *testing.T) {
		query := []byte{
			0x56, 0x78, // ID
			0x01, 0x00, // Flags (standard query)
			0x00, 0x01, // QDCount (1 question)
			0x00, 0x00, // ANCount (0 answers)
			0x00, 0x00, // NSCount (0 authority)
			0x00, 0x00, // ARCount (0 additional)
			// Question section
			11, 'n', 'o', 'n', 'e', 'x', 'i', 's', 't', 'e', 'n', 't', 7, 'e', 'x', 'a', 'm', 'p', 'l', 'e', 3, 'c', 'o', 'm', 0, // nonexistent.example.com
			0x00, 0x01, // Type A
			0x00, 0x01, // Class IN
		}

		// Connect to server
		serverAddr, err := net.ResolveUDPAddr("udp", ":8054")
		if err != nil {
			t.Fatalf("Error resolving server address: %v", err)
		}

		clientConn, err := net.DialUDP("udp", nil, serverAddr)
		if err != nil {
			t.Fatalf("Error connecting to server: %v", err)
		}
		defer clientConn.Close()

		// Send query
		_, err = clientConn.Write(query)
		if err != nil {
			t.Fatalf("Error sending query: %v", err)
		}

		// Read response
		response := make([]byte, dns.MESSAGE_SIZE)
		clientConn.SetReadDeadline(time.Now().Add(2 * time.Second))
		n, err := clientConn.Read(response)
		if err != nil {
			t.Fatalf("Error reading response: %v", err)
		}

		// Parse response
		responseMsg, err := dns.ParseDNSMessage(response[:n])
		if err != nil {
			t.Fatalf("Error parsing response: %v", err)
		}

		// Verify NXDOMAIN response
		if responseMsg.Header.ID != 0x5678 {
			t.Errorf("Response ID = %v, want %v", responseMsg.Header.ID, 0x5678)
		}

		if responseMsg.Header.Flags&0x0003 == 0 {
			t.Errorf("Response should have NXDOMAIN flag set")
		}

		if responseMsg.Header.ANCount != 0 {
			t.Errorf("NXDOMAIN response ANCount = %v, want %v", responseMsg.Header.ANCount, 0)
		}

		if len(responseMsg.Answers) != 0 {
			t.Errorf("NXDOMAIN response should have no answers, got %v", len(responseMsg.Answers))
		}
	})
}
