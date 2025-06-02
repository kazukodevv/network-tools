# Simple DNS Server

A lightweight DNS server implementation in Go that can parse DNS queries and respond to DNS requests. This server listens on UDP port 8053 and implements DNS message parsing with support for domain name compression.

## Features

- ✅ DNS message parsing (header, questions)
- ✅ Domain name parsing with compression pointer support
- ✅ UDP server implementation
- ✅ Concurrent request handling
- ⏳ DNS response generation (in development)
- ⏳ A record resolution (in development)

## Architecture

The server implements the following DNS data structures:

- **DNSHeader**: Contains message ID, flags, and record counts
- **DNSQuestion**: Represents DNS queries (domain name, type, class)
- **DNSResourceRecord**: For DNS answers (name, type, class, TTL, data)
- **DNSMessage**: Complete DNS message structure

## Getting Started

### Prerequisites

- Go 1.23.1 or later

### Installation

1. Clone or download this repository
2. Navigate to the project directory:
   ```sh
   cd simple-dns-server
   ```

### Running the Server

```sh
go run main.go
```

The server will start on port 8053 and display:
```
DNS Server started on port 8053
```

### Testing the Server

Use `dig` to test DNS queries:

```sh
# Test basic DNS queries
dig @localhost -p 8053 www.example.com
dig @localhost -p 8053 test.com
dig @localhost -p 8053 localhost

# Test with specific record types
dig @localhost -p 8053 example.com A
dig @localhost -p 8053 example.com AAAA

# Verbose output to see the full exchange
dig @localhost -p 8053 +short www.example.com
```

### Example Output

When a DNS query is received, the server will log:

```
Received DNS query from 127.0.0.1:12345: 1a2b0100000100000000000003777777076578616d706c6503636f6d0000010001
DNS Header: ID=6699, Flags=256, QDCount=1, ANCount=0, NSCount=0, ARCount=0
Parsed question name: www.example.com
Parsed DNS message: &{Header:{ID:6699 Flags=256 QDCount:1 ANCount:0 NSCount:0 ARCount:0} Questions:[{Name:www.example.com Type:1 Class:1}] Answers:[]}
```

## Testing

The project includes comprehensive unit tests and integration tests to ensure reliability.

### Running Tests

```sh
# Run all tests
go test -v

# Run tests with coverage
go test -cover

# Run only unit tests (excluding integration tests)
go test -v -run "^Test[^I]"

# Run only integration tests
go test -v -run "TestIntegration"

# Run benchmarks
go test -bench=.

# Generate coverage report
go test -coverprofile=coverage.out
go tool cover -html=coverage.out
```

### Test Coverage

The test suite includes:

- **Unit Tests**: Test individual functions like domain name parsing, DNS message encoding/decoding
- **Integration Tests**: End-to-end tests that start a real server and send DNS queries
- **Benchmark Tests**: Performance tests for critical functions
- **Error Handling Tests**: Validate proper error handling for malformed inputs

Current test coverage: ~67% of statements

## DNS Protocol Implementation

### Supported DNS Record Types

| Type | Value | Description |
|------|-------|-------------|
| A    | 1     | IPv4 address |
| NS   | 2     | Name server |
| CNAME| 5     | Canonical name |
| AAAA | 28    | IPv6 address |

### DNS Classes

| Class | Value | Description |
|-------|-------|-------------|
| IN    | 1     | Internet |

## Development

### Project Structure

```
.
├── go.mod          # Go module definition
├── main.go         # Main server implementation
└── README.md       # This file
```

### Key Functions

- `main()`: Sets up UDP server and handles incoming connections
- `handleDNSQuery()`: Processes individual DNS queries
- `parseDNSMessage()`: Parses DNS packet headers and questions
- `parseQuestions()`: Extracts DNS questions from packets
- `parseDomainName()`: Handles domain name parsing with compression

### Adding DNS Records

Currently, the server only parses DNS queries. To add response functionality, uncomment and modify the `dnsRecords` map in `main.go`:

```go
var dnsRecords = map[string]map[uint16][]byte{
    "example.com": {
        TYPE_A: []byte{192, 168, 1, 1}, // 192.168.1.1
    },
    "test.com": {
        TYPE_A: []byte{10, 0, 0, 1}, // 10.0.0.1
    },
    "localhost": {
        TYPE_A: []byte{127, 0, 0, 1}, // 127.0.0.1
    },
}
```

## Configuration

- **Port**: 8053 (configurable via `DNS_PORT` constant)
- **Message Size**: 512 bytes (configurable via `MESSAGE_SIZE` constant)
- **Minimum Message Size**: 12 bytes (DNS header size)

## Troubleshooting

### Common Issues

1. **Permission denied on port 53**: Use port 8053 instead (port 53 requires root privileges)
2. **No response from server**: Ensure the server is running and listening on the correct port
3. **Parse errors**: Check that DNS queries are properly formatted

### Debugging

Enable verbose logging by examining the console output when running the server. The server logs:
- Incoming DNS queries (hex format)
- Parsed DNS headers
- Domain names extracted from queries
- Any parsing errors
