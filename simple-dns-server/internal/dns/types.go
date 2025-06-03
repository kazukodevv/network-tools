package dns

// DNS Record Types
const (
	TYPE_A     = 1
	TYPE_NS    = 2
	TYPE_CNAME = 5
	TYPE_AAAA  = 28
	CLASS_IN   = 1
)

// Server constants
const (
	DNS_PORT         = 8053
	MESSAGE_SIZE     = 512
	MIN_MESSAGE_SIZE = 12
	DEFAULT_TTL      = 300 // Default TTL for DNS records in seconds
)

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
