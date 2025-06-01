package main

import (
	"fmt"
	"log"
	"net"
)

const DNS_PORT = 8053
const MESSAGE_SIZE = 512
const MIN_MESSAGE_SIZE = 12

type DNSHeader struct {
	ID uint16
}

type DNSMessage struct {
	Header DNSHeader
}

func main() {
	addr, err := net.ResolveUDPAddr("udp", fmt.Sprintf(":%d", DNS_PORT))
	if err != nil {
		log.Fatal("Error resolving UDP address:", err)
	}

	conn, err := net.ListenUDP("udp", addr)
	if err != nil {
		log.Fatal("Error listening on UDP:", err)
	}
	defer conn.Close()

	fmt.Printf("DNS Server started on port %d\n", DNS_PORT)

	for {
		buffer := make([]byte, MESSAGE_SIZE)
		n, clientAddr, err := conn.ReadFromUDP(buffer)
		if err != nil {
			log.Printf("Error reading from UDP: %v", err)
			continue
		}

		go handleDNSQuery(conn, clientAddr, buffer[:n])
	}
}

func handleDNSQuery(conn *net.UDPConn, clientAddr *net.UDPAddr, data []byte) {
	fmt.Printf("Received DNS query from %s: %x\n", clientAddr, data)

	if len(data) < MIN_MESSAGE_SIZE {
		log.Println()
		return
	}

	msg, err := parseDNSMessage(data)
	if err != nil {
		log.Printf("Error parsing DNS message: %v", err)
		return
	}

	fmt.Println("Parsed DNS message:", msg)
}

func parseDNSMessage(data []byte) (*DNSMessage, error) {
	if len(data) < MIN_MESSAGE_SIZE {
		return nil, fmt.Errorf("message too short")
	}

	msg := &DNSMessage{}

	return msg, nil
}
