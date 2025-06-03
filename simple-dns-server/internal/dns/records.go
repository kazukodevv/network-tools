package dns

// RecordStore manages DNS records in memory
type RecordStore struct {
	records map[string]map[uint16][]byte
}

// NewRecordStore creates a new DNS record store with default records
func NewRecordStore() *RecordStore {
	return &RecordStore{
		records: map[string]map[uint16][]byte{
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
		},
	}
}

// LookupRecord looks up a DNS record by domain name and type
func (rs *RecordStore) LookupRecord(domain string, recordType uint16) ([]byte, bool) {
	if domainRecords, exists := rs.records[domain]; exists {
		if data, hasType := domainRecords[recordType]; hasType {
			return data, true
		}
	}
	return nil, false
}

// AddRecord adds a DNS record to the store
func (rs *RecordStore) AddRecord(domain string, recordType uint16, data []byte) {
	if rs.records[domain] == nil {
		rs.records[domain] = make(map[uint16][]byte)
	}
	rs.records[domain][recordType] = data
}

// RemoveRecord removes a DNS record from the store
func (rs *RecordStore) RemoveRecord(domain string, recordType uint16) {
	if domainRecords, exists := rs.records[domain]; exists {
		delete(domainRecords, recordType)
		if len(domainRecords) == 0 {
			delete(rs.records, domain)
		}
	}
}
