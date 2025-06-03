package main

import (
	"bytes"
	"net"
	"testing"
	"time"
)

func TestParseDomainName(t *testing.T) {
	tests := []struct {
		name     string
		data     []byte
		offset   int
		expected string
		wantErr  bool
	}{
		{
			name:     "simple domain",
			data:     []byte{3, 'w', 'w', 'w', 7, 'e', 'x', 'a', 'm', 'p', 'l', 'e', 3, 'c', 'o', 'm', 0},
			offset:   0,
			expected: "www.example.com",
			wantErr:  false,
		},
		{
			name:     "root domain",
			data:     []byte{0},
			offset:   0,
			expected: ".",
			wantErr:  false,
		},
		{
			name:     "single label",
			data:     []byte{9, 'l', 'o', 'c', 'a', 'l', 'h', 'o', 's', 't', 0},
			offset:   0,
			expected: "localhost",
			wantErr:  false,
		},
		{
			name:    "truncated data",
			data:    []byte{3, 'w', 'w'},
			offset:  0,
			wantErr: true,
		},
		{
			name:    "empty data",
			data:    []byte{},
			offset:  0,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, _, err := parseDomainName(tt.data, tt.offset)
			if tt.wantErr {
				if err == nil {
					t.Errorf("parseDomainName() expected error but got none")
				}
				return
			}
			if err != nil {
				t.Errorf("parseDomainName() error = %v", err)
				return
			}
			if result != tt.expected {
				t.Errorf("parseDomainName() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestEncodeDomainName(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []byte
	}{
		{
			name:     "simple domain",
			input:    "www.example.com",
			expected: []byte{3, 'w', 'w', 'w', 7, 'e', 'x', 'a', 'm', 'p', 'l', 'e', 3, 'c', 'o', 'm', 0},
		},
		{
			name:     "empty domain",
			input:    "",
			expected: []byte{0},
		},
		{
			name:     "single label",
			input:    "localhost",
			expected: []byte{9, 'l', 'o', 'c', 'a', 'l', 'h', 'o', 's', 't', 0},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := encodeDomainName(tt.input)
			if !bytes.Equal(result, tt.expected) {
				t.Errorf("encodeDomainName() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestParseDNSMessage(t *testing.T) {
	// Create a simple DNS query for www.example.com A record
	data := []byte{
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

	msg, err := parseDNSMessage(data)
	if err != nil {
		t.Fatalf("parseDNSMessage() error = %v", err)
	}

	if msg.Header.ID != 0x1234 {
		t.Errorf("Header.ID = %v, want %v", msg.Header.ID, 0x1234)
	}

	if msg.Header.QDCount != 1 {
		t.Errorf("Header.QDCount = %v, want %v", msg.Header.QDCount, 1)
	}

	if len(msg.Questions) != 1 {
		t.Errorf("len(Questions) = %v, want %v", len(msg.Questions), 1)
	}

	if msg.Questions[0].Name != "www.example.com" {
		t.Errorf("Questions[0].Name = %v, want %v", msg.Questions[0].Name, "www.example.com")
	}

	if msg.Questions[0].Type != TYPE_A {
		t.Errorf("Questions[0].Type = %v, want %v", msg.Questions[0].Type, TYPE_A)
	}

	if msg.Questions[0].Class != CLASS_IN {
		t.Errorf("Questions[0].Class = %v, want %v", msg.Questions[0].Class, CLASS_IN)
	}
}

func TestCreateDNSResponse(t *testing.T) {
	// Create a query for www.example.com
	query := &DNSMessage{
		Header: DNSHeader{
			ID:      0x1234,
			Flags:   0x0100,
			QDCount: 1,
			ANCount: 0,
			NSCount: 0,
			ARCount: 0,
		},
		Questions: []DNSQuestion{
			{
				Name:  "www.example.com",
				Type:  TYPE_A,
				Class: CLASS_IN,
			},
		},
	}

	response := createDNSResponse(query)

	// Check response header
	if response.Header.ID != query.Header.ID {
		t.Errorf("Response ID = %v, want %v", response.Header.ID, query.Header.ID)
	}

	if response.Header.Flags != 0x8180 {
		t.Errorf("Response Flags = %v, want %v", response.Header.Flags, 0x8180)
	}

	if response.Header.QDCount != 1 {
		t.Errorf("Response QDCount = %v, want %v", response.Header.QDCount, 1)
	}

	if response.Header.ANCount != 1 {
		t.Errorf("Response ANCount = %v, want %v", response.Header.ANCount, 1)
	}

	// Check that the question is echoed back
	if len(response.Questions) != 1 {
		t.Errorf("len(Response.Questions) = %v, want %v", len(response.Questions), 1)
	}

	// Check the answer
	if len(response.Answers) != 1 {
		t.Errorf("len(Response.Answers) = %v, want %v", len(response.Answers), 1)
	}

	answer := response.Answers[0]
	if answer.Name != "www.example.com" {
		t.Errorf("Answer.Name = %v, want %v", answer.Name, "www.example.com")
	}

	if answer.Type != TYPE_A {
		t.Errorf("Answer.Type = %v, want %v", answer.Type, TYPE_A)
	}

	if answer.Class != CLASS_IN {
		t.Errorf("Answer.Class = %v, want %v", answer.Class, CLASS_IN)
	}

	expectedIP := []byte{192, 168, 1, 1}
	if !bytes.Equal(answer.Data, expectedIP) {
		t.Errorf("Answer.Data = %v, want %v", answer.Data, expectedIP)
	}
}

func TestCreateDNSResponseNXDOMAIN(t *testing.T) {
	// Create a query for a non-existent domain
	query := &DNSMessage{
		Header: DNSHeader{
			ID:      0x1234,
			Flags:   0x0100,
			QDCount: 1,
			ANCount: 0,
			NSCount: 0,
			ARCount: 0,
		},
		Questions: []DNSQuestion{
			{
				Name:  "nonexistent.example.com",
				Type:  TYPE_A,
				Class: CLASS_IN,
			},
		},
	}

	response := createDNSResponse(query)

	// Check that NXDOMAIN flag is set
	if response.Header.Flags&0x0003 == 0 {
		t.Errorf("Expected NXDOMAIN flag to be set, got flags = %v", response.Header.Flags)
	}

	if response.Header.ANCount != 0 {
		t.Errorf("Response ANCount = %v, want %v for NXDOMAIN", response.Header.ANCount, 0)
	}

	if len(response.Answers) != 0 {
		t.Errorf("len(Response.Answers) = %v, want %v for NXDOMAIN", len(response.Answers), 0)
	}
}

func TestEncodeDNSMessage(t *testing.T) {
	msg := &DNSMessage{
		Header: DNSHeader{
			ID:      0x1234,
			Flags:   0x8180,
			QDCount: 1,
			ANCount: 1,
			NSCount: 0,
			ARCount: 0,
		},
		Questions: []DNSQuestion{
			{
				Name:  "test.com",
				Type:  TYPE_A,
				Class: CLASS_IN,
			},
		},
		Answers: []DNSResourceRecord{
			{
				Name:  "test.com",
				Type:  TYPE_A,
				Class: CLASS_IN,
				TTL:   300,
				Data:  []byte{10, 0, 0, 1},
			},
		},
	}

	encoded := encodeDNSMessage(msg)

	// Check header encoding
	if encoded[0] != 0x12 || encoded[1] != 0x34 {
		t.Errorf("ID encoding failed: got %02x%02x, want 1234", encoded[0], encoded[1])
	}

	if encoded[2] != 0x81 || encoded[3] != 0x80 {
		t.Errorf("Flags encoding failed: got %02x%02x, want 8180", encoded[2], encoded[3])
	}

	// Verify we can parse it back
	parsed, err := parseDNSMessage(encoded)
	if err != nil {
		t.Errorf("Failed to parse encoded message: %v", err)
	}

	if parsed.Header.ID != msg.Header.ID {
		t.Errorf("Round-trip ID failed: got %v, want %v", parsed.Header.ID, msg.Header.ID)
	}

	if len(parsed.Questions) != 1 {
		t.Errorf("Round-trip questions failed: got %v questions, want 1", len(parsed.Questions))
	}

	if parsed.Questions[0].Name != "test.com" {
		t.Errorf("Round-trip question name failed: got %v, want test.com", parsed.Questions[0].Name)
	}
}

func TestDNSServer(t *testing.T) {
	// Start the server in a goroutine
	go func() {
		addr, err := net.ResolveUDPAddr("udp", ":0") // Use port 0 to get a free port
		if err != nil {
			t.Errorf("Error resolving UDP address: %v", err)
			return
		}

		conn, err := net.ListenUDP("udp", addr)
		if err != nil {
			t.Errorf("Error listening on UDP: %v", err)
			return
		}
		defer conn.Close()

		// Handle one request for testing
		buffer := make([]byte, MESSAGE_SIZE)
		n, clientAddr, err := conn.ReadFromUDP(buffer)
		if err != nil {
			t.Errorf("Error reading from UDP: %v", err)
			return
		}

		handleDNSQuery(conn, clientAddr, buffer[:n])
	}()

	// Give the server time to start
	time.Sleep(100 * time.Millisecond)
}

func TestParseQuestionsInvalidData(t *testing.T) {
	tests := []struct {
		name   string
		data   []byte
		offset int
	}{
		{
			name:   "insufficient data for type and class",
			data:   []byte{3, 'w', 'w', 'w', 0, 0x00}, // Missing class bytes
			offset: 0,
		},
		{
			name:   "empty data",
			data:   []byte{},
			offset: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, err := parseQuestions(tt.data, tt.offset)
			if err == nil {
				t.Errorf("parseQuestions() expected error but got none")
			}
		})
	}
}

func TestParseDNSMessageTooShort(t *testing.T) {
	// Test with data shorter than minimum message size
	shortData := []byte{0x12, 0x34, 0x01, 0x00, 0x00} // Only 5 bytes, need at least 12

	_, err := parseDNSMessage(shortData)
	if err == nil {
		t.Errorf("parseDNSMessage() expected error for short data but got none")
	}

	// Also test with exactly minimum size but no questions
	minData := make([]byte, MIN_MESSAGE_SIZE)
	copy(minData, []byte{0x12, 0x34, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00})

	_, err = parseDNSMessage(minData)
	if err != nil {
		t.Errorf("parseDNSMessage() with minimum valid size should not error: %v", err)
	}
}

func BenchmarkParseDomainName(b *testing.B) {
	data := []byte{3, 'w', 'w', 'w', 7, 'e', 'x', 'a', 'm', 'p', 'l', 'e', 3, 'c', 'o', 'm', 0}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _, _ = parseDomainName(data, 0)
	}
}

func BenchmarkEncodeDomainName(b *testing.B) {
	domain := "www.example.com"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = encodeDomainName(domain)
	}
}

func BenchmarkCreateDNSResponse(b *testing.B) {
	query := &DNSMessage{
		Header: DNSHeader{
			ID:      0x1234,
			Flags:   0x0100,
			QDCount: 1,
			ANCount: 0,
			NSCount: 0,
			ARCount: 0,
		},
		Questions: []DNSQuestion{
			{
				Name:  "www.example.com",
				Type:  TYPE_A,
				Class: CLASS_IN,
			},
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = createDNSResponse(query)
	}
}
