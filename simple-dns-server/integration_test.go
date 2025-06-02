package main

import (
	"net"
	"testing"
	"time"
)

func TestDNSServerIntegration(t *testing.T) {
	// Start the server on a free port
	addr, err := net.ResolveUDPAddr("udp", ":0")
	if err != nil {
		t.Fatalf("Error resolving UDP address: %v", err)
	}

	conn, err := net.ListenUDP("udp", addr)
	if err != nil {
		t.Fatalf("Error listening on UDP: %v", err)
	}
	defer conn.Close()

	serverAddr := conn.LocalAddr().(*net.UDPAddr)
	
	// Start server in background
	go func() {
		for {
			buffer := make([]byte, MESSAGE_SIZE)
			n, clientAddr, err := conn.ReadFromUDP(buffer)
			if err != nil {
				return // Server stopped
			}
			go handleDNSQuery(conn, clientAddr, buffer[:n])
		}
	}()

	// Give server time to start
	time.Sleep(50 * time.Millisecond)

	// Create a DNS query for www.example.com
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

	// Send query to server
	clientConn, err := net.DialUDP("udp", nil, serverAddr)
	if err != nil {
		t.Fatalf("Error connecting to server: %v", err)
	}
	defer clientConn.Close()

	_, err = clientConn.Write(query)
	if err != nil {
		t.Fatalf("Error sending query: %v", err)
	}

	// Read response
	response := make([]byte, MESSAGE_SIZE)
	clientConn.SetReadDeadline(time.Now().Add(1 * time.Second))
	n, err := clientConn.Read(response)
	if err != nil {
		t.Fatalf("Error reading response: %v", err)
	}

	// Parse response
	responseMsg, err := parseDNSMessage(response[:n])
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
	if answer.Type != TYPE_A {
		t.Errorf("Answer.Type = %v, want %v", answer.Type, TYPE_A)
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
}

func TestDNSServerNXDOMAINIntegration(t *testing.T) {
	// Start the server on a free port
	addr, err := net.ResolveUDPAddr("udp", ":0")
	if err != nil {
		t.Fatalf("Error resolving UDP address: %v", err)
	}

	conn, err := net.ListenUDP("udp", addr)
	if err != nil {
		t.Fatalf("Error listening on UDP: %v", err)
	}
	defer conn.Close()

	serverAddr := conn.LocalAddr().(*net.UDPAddr)
	
	// Start server in background
	go func() {
		for {
			buffer := make([]byte, MESSAGE_SIZE)
			n, clientAddr, err := conn.ReadFromUDP(buffer)
			if err != nil {
				return // Server stopped
			}
			go handleDNSQuery(conn, clientAddr, buffer[:n])
		}
	}()

	// Give server time to start
	time.Sleep(50 * time.Millisecond)

	// Create a DNS query for nonexistent.example.com
	query := []byte{
		0x12, 0x34, // ID
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

	// Send query to server
	clientConn, err := net.DialUDP("udp", nil, serverAddr)
	if err != nil {
		t.Fatalf("Error connecting to server: %v", err)
	}
	defer clientConn.Close()

	_, err = clientConn.Write(query)
	if err != nil {
		t.Fatalf("Error sending query: %v", err)
	}

	// Read response
	response := make([]byte, MESSAGE_SIZE)
	clientConn.SetReadDeadline(time.Now().Add(1 * time.Second))
	n, err := clientConn.Read(response)
	if err != nil {
		t.Fatalf("Error reading response: %v", err)
	}

	// Parse response
	responseMsg, err := parseDNSMessage(response[:n])
	if err != nil {
		t.Fatalf("Error parsing response: %v", err)
	}

	// Verify NXDOMAIN response
	if responseMsg.Header.ID != 0x1234 {
		t.Errorf("Response ID = %v, want %v", responseMsg.Header.ID, 0x1234)
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
}
