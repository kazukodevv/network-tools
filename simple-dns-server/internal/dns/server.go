package dns

import (
	"fmt"
	"log/slog"
	"net"
	"strings"
)

// Server represents a DNS server
type Server struct {
	port        int
	conn        *net.UDPConn
	recordStore *RecordStore
	logger      *slog.Logger
}

// NewServer creates a new DNS server
func NewServer(port int, logger *slog.Logger) *Server {
	return &Server{
		port:        port,
		recordStore: NewRecordStore(),
		logger:      logger,
	}
}

// Start starts the DNS server
func (s *Server) Start() error {
	addr, err := net.ResolveUDPAddr("udp", fmt.Sprintf(":%d", s.port))
	if err != nil {
		return fmt.Errorf("failed to resolve UDP address: %w", err)
	}

	s.conn, err = net.ListenUDP("udp", addr)
	if err != nil {
		return fmt.Errorf("failed to listen on UDP: %w", err)
	}

	s.logger.Info("DNS Server started",
		"port", s.port,
		"message_size", MESSAGE_SIZE)

	for {
		buffer := make([]byte, MESSAGE_SIZE)
		n, clientAddr, err := s.conn.ReadFromUDP(buffer)
		if err != nil {
			s.logger.Error("Error reading from UDP",
				"error", err,
				"client_addr", clientAddr)
			continue
		}

		go s.handleDNSQuery(clientAddr, buffer[:n])
	}
}

// Stop stops the DNS server
func (s *Server) Stop() error {
	if s.conn != nil {
		return s.conn.Close()
	}
	return nil
}

// handleDNSQuery handles a single DNS query
func (s *Server) handleDNSQuery(clientAddr *net.UDPAddr, data []byte) {
	queryLogger := s.logger.With(
		"client_addr", clientAddr.String(),
		"query_size", len(data))

	queryLogger.Debug("Received DNS query",
		"data_hex", fmt.Sprintf("%x", data))

	if len(data) < MIN_MESSAGE_SIZE {
		queryLogger.Warn("DNS message too short",
			"min_size", MIN_MESSAGE_SIZE)
		return
	}

	msg, err := ParseDNSMessage(data)
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

	response := s.createDNSResponse(msg)

	responseBytes := EncodeDNSMessage(response)
	_, err = s.conn.WriteToUDP(responseBytes, clientAddr)
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

// createDNSResponse creates a DNS response for the given query
func (s *Server) createDNSResponse(query *DNSMessage) *DNSMessage {
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

	responseLogger := s.logger.With("query_id", query.Header.ID)

	for _, question := range query.Questions {
		questionLogger := responseLogger.With(
			"domain", question.Name,
			"type", question.Type,
			"class", question.Class)

		if question.Type == TYPE_A && question.Class == CLASS_IN {
			domainName := strings.ToLower(question.Name)
			if ipData, found := s.recordStore.LookupRecord(domainName, TYPE_A); found {
				answer := DNSResourceRecord{
					Name:  question.Name,
					Type:  TYPE_A,
					Class: CLASS_IN,
					TTL:   DEFAULT_TTL,
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

	if response.Header.ANCount == 0 {
		response.Header.Flags |= 0x0003 // Set the "NXDOMAIN" flag // NXDOMAIN（Non-Existent Domain）0000 0000 0000 0011
	}

	return response
}
