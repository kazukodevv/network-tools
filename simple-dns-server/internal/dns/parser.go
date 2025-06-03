package dns

import (
	"fmt"
	"log/slog"
	"strings"
)

// ParseDNSMessage parses a DNS message from raw bytes
func ParseDNSMessage(data []byte) (*DNSMessage, error) {
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

	slog.Debug("Parsed question name", "name", question.Name)

	if newOffset+4 > len(data) {
		return question, 0, fmt.Errorf("not enough data for question type and class")
	}

	question.Type = uint16(data[newOffset])<<8 | uint16(data[newOffset+1])
	question.Class = uint16(data[newOffset+2])<<8 | uint16(data[newOffset+3])

	slog.Debug("Parsed question details",
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
