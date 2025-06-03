package unit

import (
	"bytes"
	"testing"

	"dns-server/internal/dns"
)

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
			result := dns.EncodeDomainName(tt.input)
			if !bytes.Equal(result, tt.expected) {
				t.Errorf("EncodeDomainName() = %v, want %v", result, tt.expected)
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

	msg, err := dns.ParseDNSMessage(data)
	if err != nil {
		t.Fatalf("ParseDNSMessage() error = %v", err)
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

	if msg.Questions[0].Type != dns.TYPE_A {
		t.Errorf("Questions[0].Type = %v, want %v", msg.Questions[0].Type, dns.TYPE_A)
	}

	if msg.Questions[0].Class != dns.CLASS_IN {
		t.Errorf("Questions[0].Class = %v, want %v", msg.Questions[0].Class, dns.CLASS_IN)
	}
}

func TestParseDNSMessageTooShort(t *testing.T) {
	// Test with data shorter than minimum message size
	shortData := []byte{0x12, 0x34, 0x01, 0x00, 0x00} // Only 5 bytes, need at least 12

	_, err := dns.ParseDNSMessage(shortData)
	if err == nil {
		t.Errorf("ParseDNSMessage() expected error for short data but got none")
	}

	// Also test with exactly minimum size but no questions
	minData := make([]byte, dns.MIN_MESSAGE_SIZE)
	copy(minData, []byte{0x12, 0x34, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00})

	_, err = dns.ParseDNSMessage(minData)
	if err != nil {
		t.Errorf("ParseDNSMessage() with minimum valid size should not error: %v", err)
	}
}

func TestEncodeDNSMessage(t *testing.T) {
	msg := &dns.DNSMessage{
		Header: dns.DNSHeader{
			ID:      0x1234,
			Flags:   0x8180,
			QDCount: 1,
			ANCount: 1,
			NSCount: 0,
			ARCount: 0,
		},
		Questions: []dns.DNSQuestion{
			{
				Name:  "test.com",
				Type:  dns.TYPE_A,
				Class: dns.CLASS_IN,
			},
		},
		Answers: []dns.DNSResourceRecord{
			{
				Name:  "test.com",
				Type:  dns.TYPE_A,
				Class: dns.CLASS_IN,
				TTL:   300,
				Data:  []byte{10, 0, 0, 1},
			},
		},
	}

	encoded := dns.EncodeDNSMessage(msg)

	// Check header encoding
	if encoded[0] != 0x12 || encoded[1] != 0x34 {
		t.Errorf("ID encoding failed: got %02x%02x, want 1234", encoded[0], encoded[1])
	}

	if encoded[2] != 0x81 || encoded[3] != 0x80 {
		t.Errorf("Flags encoding failed: got %02x%02x, want 8180", encoded[2], encoded[3])
	}

	// Verify we can parse it back
	parsed, err := dns.ParseDNSMessage(encoded)
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

func TestRecordStore(t *testing.T) {
	store := dns.NewRecordStore()

	// Test existing record
	if data, found := store.LookupRecord("www.example.com", dns.TYPE_A); !found {
		t.Errorf("Expected to find record for www.example.com")
	} else {
		expectedIP := []byte{192, 168, 1, 1}
		if !bytes.Equal(data, expectedIP) {
			t.Errorf("Expected IP %v, got %v", expectedIP, data)
		}
	}

	// Test non-existent record
	if _, found := store.LookupRecord("nonexistent.com", dns.TYPE_A); found {
		t.Errorf("Expected not to find record for nonexistent.com")
	}

	// Test adding a record
	newIP := []byte{192, 168, 1, 100}
	store.AddRecord("new.example.com", dns.TYPE_A, newIP)

	if data, found := store.LookupRecord("new.example.com", dns.TYPE_A); !found {
		t.Errorf("Expected to find newly added record")
	} else if !bytes.Equal(data, newIP) {
		t.Errorf("Expected IP %v, got %v", newIP, data)
	}

	// Test removing a record
	store.RemoveRecord("new.example.com", dns.TYPE_A)
	if _, found := store.LookupRecord("new.example.com", dns.TYPE_A); found {
		t.Errorf("Expected record to be removed")
	}
}

func TestDNSMessageRoundTrip(t *testing.T) {
	// Test round-trip: encode a message and then parse it back
	originalMsg := &dns.DNSMessage{
		Header: dns.DNSHeader{
			ID:      0xABCD,
			Flags:   0x0100,
			QDCount: 1,
			ANCount: 0,
			NSCount: 0,
			ARCount: 0,
		},
		Questions: []dns.DNSQuestion{
			{
				Name:  "example.com",
				Type:  dns.TYPE_A,
				Class: dns.CLASS_IN,
			},
		},
	}

	// Encode the message
	encoded := dns.EncodeDNSMessage(originalMsg)

	// Parse it back
	parsed, err := dns.ParseDNSMessage(encoded)
	if err != nil {
		t.Fatalf("Failed to parse encoded message: %v", err)
	}

	// Compare the results
	if parsed.Header.ID != originalMsg.Header.ID {
		t.Errorf("ID mismatch: got %v, want %v", parsed.Header.ID, originalMsg.Header.ID)
	}

	if parsed.Header.Flags != originalMsg.Header.Flags {
		t.Errorf("Flags mismatch: got %v, want %v", parsed.Header.Flags, originalMsg.Header.Flags)
	}

	if len(parsed.Questions) != len(originalMsg.Questions) {
		t.Errorf("Questions length mismatch: got %v, want %v", len(parsed.Questions), len(originalMsg.Questions))
	}

	if len(parsed.Questions) > 0 && len(originalMsg.Questions) > 0 {
		if parsed.Questions[0].Name != originalMsg.Questions[0].Name {
			t.Errorf("Question name mismatch: got %v, want %v", parsed.Questions[0].Name, originalMsg.Questions[0].Name)
		}
	}
}

// Benchmark tests
func BenchmarkEncodeDomainName(b *testing.B) {
	domain := "www.example.com"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = dns.EncodeDomainName(domain)
	}
}

func BenchmarkParseDNSMessage(b *testing.B) {
	data := []byte{
		0x12, 0x34, // ID
		0x01, 0x00, // Flags
		0x00, 0x01, // QDCount
		0x00, 0x00, // ANCount
		0x00, 0x00, // NSCount
		0x00, 0x00, // ARCount
		3, 'w', 'w', 'w', 7, 'e', 'x', 'a', 'm', 'p', 'l', 'e', 3, 'c', 'o', 'm', 0,
		0x00, 0x01, // Type A
		0x00, 0x01, // Class IN
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = dns.ParseDNSMessage(data)
	}
}

func BenchmarkEncodeDNSMessage(b *testing.B) {
	msg := &dns.DNSMessage{
		Header: dns.DNSHeader{
			ID:      0x1234,
			Flags:   0x8180,
			QDCount: 1,
			ANCount: 1,
			NSCount: 0,
			ARCount: 0,
		},
		Questions: []dns.DNSQuestion{
			{
				Name:  "test.com",
				Type:  dns.TYPE_A,
				Class: dns.CLASS_IN,
			},
		},
		Answers: []dns.DNSResourceRecord{
			{
				Name:  "test.com",
				Type:  dns.TYPE_A,
				Class: dns.CLASS_IN,
				TTL:   300,
				Data:  []byte{10, 0, 0, 1},
			},
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = dns.EncodeDNSMessage(msg)
	}
}
