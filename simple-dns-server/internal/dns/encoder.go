package dns

import (
	"log/slog"
	"strings"
)

// EncodeDNSMessage encodes a DNS message to bytes
func EncodeDNSMessage(msg *DNSMessage) []byte {
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
		nameBytes := EncodeDomainName(question.Name)
		buffer = append(buffer, nameBytes...)
		buffer = append(buffer, byte(question.Type>>8), byte(question.Type))
		buffer = append(buffer, byte(question.Class>>8), byte(question.Class))
	}

	// Encode the answers
	for _, answer := range msg.Answers {
		buffer = append(buffer, EncodeDomainName(answer.Name)...)
		buffer = append(buffer, byte(answer.Type>>8), byte(answer.Type))
		buffer = append(buffer, byte(answer.Class>>8), byte(answer.Class))
		buffer = append(buffer, byte(answer.TTL>>24), byte(answer.TTL>>16),
			byte(answer.TTL>>8), byte(answer.TTL))
		buffer = append(buffer, byte(len(answer.Data)>>8), byte(len(answer.Data)))
		buffer = append(buffer, answer.Data...)
	}

	return buffer
}

// EncodeDomainName encodes a domain name to DNS format
func EncodeDomainName(name string) []byte {
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
