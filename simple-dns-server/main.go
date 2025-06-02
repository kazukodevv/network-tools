package main

import (
	"fmt"
	"log/slog"
	"net"
	"os"
	"strings"
)

const DNS_PORT = 8053
const MESSAGE_SIZE = 512
const MIN_MESSAGE_SIZE = 12

// DNSHeader represents the header of a DNS message
type DNSHeader struct {
	ID      uint16 // Identifier for the DNS message
	Flags   uint16 // Flags for the DNS message
	QDCount uint16 // Number of questions
	ANCount uint16 // Number of answers
	NSCount uint16 // Number of authority records
	ARCount uint16 // Number of additional records
}

// DNSQuestion represents a single DNS question
type DNSQuestion struct {
	Name  string // Domain name in the question
	Type  uint16 // Type of the query (A, AAAA, etc.)
	Class uint16 // Class of the query (IN, CH, HS, etc.)
}

// DNSResourceRecord represents a single DNS resource record
type DNSResourceRecord struct {
	Name  string // Domain name of the resource record
	Type  uint16 // Type of the resource record (A, AAAA, etc.)
	Class uint16 // Class of the resource record (IN, CH, HS, etc.)
	TTL   uint32 // Time to live for the resource record
	Data  []byte // Data of the resource record (IP address, etc.)
}

// DNSMessage represents a complete DNS message
type DNSMessage struct {
	Header    DNSHeader           // Header of the DNS message
	Questions []DNSQuestion       // List of questions in the DNS message
	Answers   []DNSResourceRecord // List of answers in the DNS message
}

// DNS Record Types
const (
	TYPE_A     = 1
	TYPE_NS    = 2
	TYPE_CNAME = 5
	TYPE_AAAA  = 28
	CLASS_IN   = 1
)

// Simple in memory DNS server that listens for DNS queries on UDP port 8053
var dnsRecords = map[string]map[uint16][]byte{
	"www.example.com": {
		TYPE_A: []byte{192, 168, 1, 1}, // 192.168.1.1
	},
	"example.com": {
		TYPE_A: []byte{192, 168, 1, 1}, // 192.168.1.1
	},
	"test.com": {
		TYPE_A: []byte{10, 0, 0, 1}, // 10.0.0.1
	},
	"localhost": {
		TYPE_A: []byte{127, 0, 0, 1}, // 127.0.0.1
	},
	"google.com": {
		TYPE_A: []byte{8, 8, 8, 8}, // 8.8.8.8 (example)
	},
}

var logger *slog.Logger

func init() {
	handler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level:     slog.LevelDebug,
		AddSource: true,
	})
	logger = slog.New(handler)
	slog.SetDefault(logger)
}

func main() {
	addr, err := net.ResolveUDPAddr("udp", fmt.Sprintf(":%d", DNS_PORT))
	if err != nil {
		logger.Error("Failed to resolve UDP address",
			"error", err,
			"port", DNS_PORT)
		os.Exit(1)
	}

	conn, err := net.ListenUDP("udp", addr)
	if err != nil {
		logger.Error("Failed to listen on UDP",
			"error", err,
			"address", addr.String())
		os.Exit(1)
	}
	defer conn.Close()

	logger.Info("DNS Server started",
		"port", DNS_PORT,
		"message_size", MESSAGE_SIZE)

	for {
		buffer := make([]byte, MESSAGE_SIZE)
		n, clientAddr, err := conn.ReadFromUDP(buffer)
		if err != nil {
			logger.Error("Error reading from UDP",
				"error", err,
				"client_addr", clientAddr)
			continue
		}

		go handleDNSQuery(conn, clientAddr, buffer[:n])
	}
}

func handleDNSQuery(conn *net.UDPConn, clientAddr *net.UDPAddr, data []byte) {
	queryLogger := logger.With(
		"client_addr", clientAddr.String(),
		"query_size", len(data))

	queryLogger.Debug("Received DNS query",
		"data_hex", fmt.Sprintf("%x", data))

	if len(data) < MIN_MESSAGE_SIZE {
		queryLogger.Warn("DNS message too short",
			"min_size", MIN_MESSAGE_SIZE)
		return
	}

	msg, err := parseDNSMessage(data)
	if err != nil {
		queryLogger.Error("Failed to parse DNS message", "error", err)
		return
	}

	queryLogger.Info("Parsed DNS message",
		"message_id", msg.Header.ID,
		"flags", msg.Header.Flags,
		"question_count", msg.Header.QDCount,
		"question_name", func() string {
			if len(msg.Questions) > 0 {
				return msg.Questions[0].Name
			}
			return ""
		}())

	response := createDNSResponse(msg)

	responseBytes := encodeDNSMessage(response)
	_, err = conn.WriteToUDP(responseBytes, clientAddr)
	if err != nil {
		queryLogger.Error("Failed to send DNS response", "error", err)
		return
	}

	queryLogger.Info("Query handled successfully",
		"domain", func() string {
			if len(msg.Questions) > 0 {
				return msg.Questions[0].Name
			}
			return ""
		}(),
		"response_size", len(responseBytes),
		"answer_count", response.Header.ANCount)
}

func parseDNSMessage(data []byte) (*DNSMessage, error) {
	if len(data) < MIN_MESSAGE_SIZE {
		return nil, fmt.Errorf("message too short: %d bytes, minimum %d required", len(data), MIN_MESSAGE_SIZE)
	}

	msg := &DNSMessage{}

	msg.Header.ID = uint16(data[0])<<8 | uint16(data[1])
	msg.Header.Flags = uint16(data[2])<<8 | uint16(data[3])
	msg.Header.QDCount = uint16(data[4])<<8 | uint16(data[5])
	msg.Header.ANCount = uint16(data[6])<<8 | uint16(data[7])
	msg.Header.NSCount = uint16(data[8])<<8 | uint16(data[9])
	msg.Header.ARCount = uint16(data[10])<<8 | uint16(data[11])

	fmt.Printf("DNS Header: ID=%d, Flags=%d, QDCount=%d, ANCount=%d, NSCount=%d, ARCount=%d\n",
		msg.Header.ID, msg.Header.Flags, msg.Header.QDCount,
		msg.Header.ANCount, msg.Header.NSCount, msg.Header.ARCount)

	offset := 12
	for range int(msg.Header.QDCount) {
		question, newOffset, err := parseQuestions(data, offset)
		if err != nil {
			return nil, err
		}
		msg.Questions = append(msg.Questions, question)
		offset = newOffset
	}

	return msg, nil
}

func parseQuestions(data []byte, offset int) (DNSQuestion, int, error) {
	question := DNSQuestion{}

	name, newOffset, err := parseDomainName(data, offset)
	if err != nil {
		return question, 0, err
	}
	question.Name = name

	logger.Debug("Parsed question name", "name", question.Name)

	if newOffset+4 > len(data) {
		return question, 0, fmt.Errorf("not enough data for question type and class")
	}

	question.Type = uint16(data[newOffset])<<8 | uint16(data[newOffset+1])
	question.Class = uint16(data[newOffset+2])<<8 | uint16(data[newOffset+3])

	logger.Debug("Parsed question details",
		"name", question.Name,
		"type", question.Type,
		"class", question.Class)

	return question, newOffset + 4, nil
}

func parseDomainName(data []byte, offset int) (string, int, error) {
	var labels []string

	for {
		if offset >= len(data) {
			return "", 0, fmt.Errorf("unexpected end of data")
		}

		length := data[offset]

		// check for end of labels
		if length == 0 {
			offset++
			break
		}

		// check for compression pointer
		// 0xC0 = 11000000
		if length&0xC0 == 0xC0 {
			if offset+1 >= len(data) {
				return "", 0, fmt.Errorf("invalid compression pointer")
			}
			// 0x3F = 00111111
			pointer := int(uint16(length&0x3F)<<8 | uint16(data[offset+1]))
			name, _, err := parseDomainName(data, pointer)
			if err != nil {
				return "", 0, err
			}
			labels = append(labels, strings.Split(name, ".")...)
			offset += 2
			break
		}

		if offset+int(length)+1 > len(data) {
			return "", 0, fmt.Errorf("label extends beyond data")
		}

		label := string(data[offset+1 : offset+1+int(length)])
		labels = append(labels, label)
		offset += int(length) + 1
	}

	if len(labels) == 0 {
		return ".", offset, nil
	}

	return strings.Join(labels, "."), offset, nil
}

func createDNSResponse(query *DNSMessage) *DNSMessage {
	response := &DNSMessage{
		Header: DNSHeader{
			ID:      query.Header.ID,
			Flags:   0x8180, // Standard query response with no error 1000 0001 1000 0000
			QDCount: query.Header.QDCount,
			ANCount: 0,
			NSCount: 0,
			ARCount: 0,
		},
		Questions: query.Questions,
	}

	responseLogger := logger.With("query_id", query.Header.ID)

	for _, question := range query.Questions {
		questionLogger := responseLogger.With(
			"domain", question.Name,
			"type", question.Type,
			"class", question.Class)

		if question.Type == TYPE_A && question.Class == CLASS_IN {
			domainName := strings.ToLower(question.Name)
			if records, exists := dnsRecords[domainName]; exists {
				if ipData, hasA := records[TYPE_A]; hasA {
					answer := DNSResourceRecord{
						Name:  question.Name,
						Type:  TYPE_A,
						Class: CLASS_IN,
						TTL:   300, // Example TTL of 300 seconds
						Data:  ipData,
					}
					response.Answers = append(response.Answers, answer)
					response.Header.ANCount++

					questionLogger.Info("DNS record found",
						"ip", fmt.Sprintf("%d.%d.%d.%d", ipData[0], ipData[1], ipData[2], ipData[3]),
						"ttl", answer.TTL)
				}
			}
		}
	}

	if response.Header.ANCount == 0 {
		response.Header.Flags |= 0x0003 // Set the "NXDOMAIN" flag // NXDOMAIN（Non-Existent Domain）0000 0000 0000 0011
	}

	return response
}

func encodeDNSMessage(msg *DNSMessage) []byte {
	var buffer []byte

	// Encode the header
	buffer = append(buffer, byte(msg.Header.ID>>8), byte(msg.Header.ID))
	buffer = append(buffer, byte(msg.Header.Flags>>8), byte(msg.Header.Flags))
	buffer = append(buffer, byte(msg.Header.QDCount>>8), byte(msg.Header.QDCount))
	buffer = append(buffer, byte(msg.Header.ANCount>>8), byte(msg.Header.ANCount))
	buffer = append(buffer, byte(msg.Header.NSCount>>8), byte(msg.Header.NSCount))
	buffer = append(buffer, byte(msg.Header.ARCount>>8), byte(msg.Header.ARCount))

	// Encode the questions
	for _, question := range msg.Questions {
		nameBytes := encodeDomainName(question.Name)
		buffer = append(buffer, nameBytes...)
		buffer = append(buffer, byte(question.Type>>8), byte(question.Type))
		buffer = append(buffer, byte(question.Class>>8), byte(question.Class))
	}

	// Encode the answers
	for _, answer := range msg.Answers {
		buffer = append(buffer, encodeDomainName(answer.Name)...)
		buffer = append(buffer, byte(answer.Type>>8), byte(answer.Type))
		buffer = append(buffer, byte(answer.Class>>8), byte(answer.Class))
		buffer = append(buffer, byte(answer.TTL>>24), byte(answer.TTL>>16),
			byte(answer.TTL>>8), byte(answer.TTL))
		buffer = append(buffer, byte(len(answer.Data)>>8), byte(len(answer.Data)))
		buffer = append(buffer, answer.Data...)
	}

	return buffer
}

func encodeDomainName(name string) []byte {
	if name == "" {
		return []byte{0} // Empty domain name
	}

	var buffer []byte
	labels := strings.Split(name, ".")
	for _, label := range labels {
		if len(label) > 63 {
			slog.Warn("Label too long", "label", label)
			continue
		}
		buffer = append(buffer, byte(len(label)))
		buffer = append(buffer, []byte(label)...)
	}
	buffer = append(buffer, 0) // Null byte to end the domain name

	return buffer
}
