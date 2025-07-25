package protocol

const (
	PROTO_UNKNOWN = Proto(0)
	BGP           = Proto(1)
	OSPF          = Proto(2)
	Kernel        = Proto(4)
	Static        = Proto(8)
	Direct        = Proto(16)
	Babel         = Proto(32)
	RPKI          = Proto(64)
	BFD           = Proto(128)
)

type Proto int

type Protocol struct {
	Name            string
	Description     string
	IPVersion       string
	ImportFilter    string
	ExportFilter    string
	Proto           Proto
	Up              int
	State           string
	Imported        int64
	Exported        int64
	Filtered        int64
	Preferred       int64
	Uptime          int
	ImportUpdates   RouteChangeCount
	ImportWithdraws RouteChangeCount
	ExportUpdates   RouteChangeCount
	ExportWithdraws RouteChangeCount
}

type RouteChangeCount struct {
	Received int64
	Rejected int64
	Filtered int64
	Ignored  int64
	Accepted int64
}

// Route represents a single route entry from BIRD
type Route struct {
	Network     string
	PrefixLen   int
	NextHop     string
	Protocol    string
	Metric      int
	Origin      string
}

// PrefixStats holds statistics about prefix counts by prefix length
type PrefixStats struct {
	PrefixLengthCounts map[int]int64 // map[prefix_length]count
	IPVersion          string
	Protocol           string
}

// NewPrefixStats creates a new PrefixStats instance
func NewPrefixStats(ipVersion, protocol string) *PrefixStats {
	return &PrefixStats{
		PrefixLengthCounts: make(map[int]int64),
		IPVersion:          ipVersion,
		Protocol:           protocol,
	}
}

// AddRoute increments the count for the given prefix length
func (ps *PrefixStats) AddRoute(prefixLen int) {
	ps.PrefixLengthCounts[prefixLen]++
}

func NewProtocol(name string, proto Proto, ipVersion string, uptime int) *Protocol {
	return &Protocol{Name: name, Proto: proto, IPVersion: ipVersion, Uptime: uptime}
}
