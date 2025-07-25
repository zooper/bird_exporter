package client

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/czerwonk/bird_exporter/parser"
	"github.com/czerwonk/bird_exporter/protocol"
	birdsocket "github.com/czerwonk/bird_socket"
)

// BirdClient communicates with the bird socket to retrieve information
type BirdClient struct {
	Options *BirdClientOptions
}

// BirdClientOptions defines options to connect to bird
type BirdClientOptions struct {
	BirdV2       bool
	BirdEnabled  bool
	Bird6Enabled bool
	BirdSocket   string
	Bird6Socket  string
}

// GetProtocols retrieves protocol information and statistics from bird
func (c *BirdClient) GetProtocols() ([]*protocol.Protocol, error) {
	ipVersions := make([]string, 0)
	if c.Options.BirdV2 {
		ipVersions = append(ipVersions, "")
	} else {
		if c.Options.BirdEnabled {
			ipVersions = append(ipVersions, "4")
		}

		if c.Options.Bird6Enabled {
			ipVersions = append(ipVersions, "6")
		}
	}

	return c.protocolsFromBird(ipVersions)
}

// GetOSPFAreas retrieves OSPF specific information from bird
func (c *BirdClient) GetOSPFAreas(protocol *protocol.Protocol) ([]*protocol.OSPFArea, error) {
	sock := c.socketFor(protocol.IPVersion)
	b, err := birdsocket.Query(sock, fmt.Sprintf("show ospf %s", protocol.Name))
	if err != nil {
		return nil, err
	}

	return parser.ParseOSPF(b), nil
}

// GetBFDSessions retrieves BFD specific information from bird
func (c *BirdClient) GetBFDSessions(protocol *protocol.Protocol) ([]*protocol.BFDSession, error) {
	sock := c.socketFor(protocol.IPVersion)
	b, err := birdsocket.Query(sock, fmt.Sprintf("show bfd sessions %s", protocol.Name))
	if err != nil {
		return nil, err
	}

	return parser.ParseBFDSessions(protocol.Name, b), nil
}

// GetPrefixStats retrieves prefix length statistics from routing table
func (c *BirdClient) GetPrefixStats(proto *protocol.Protocol) (*protocol.PrefixStats, error) {
	sock := c.socketFor(proto.IPVersion)
	
	// Try multiple commands to get comprehensive route information
	tableName := "master4"
	if proto.IPVersion == "6" {
		tableName = "master6"
	}
	
	commands := []string{
		fmt.Sprintf("show route all protocol %s", proto.Name),  // ALL routes from protocol (including filtered)
		fmt.Sprintf("show route protocol %s all", proto.Name),  // More detailed output  
		fmt.Sprintf("show route protocol %s", proto.Name),      // Standard output (only selected routes)
		fmt.Sprintf("show route table %s protocol %s all", tableName, proto.Name), // Table-specific with all
		fmt.Sprintf("show route where source = RTS_%s", getRouteSource(proto.Proto)), // By route source
	}
	
	var stats *protocol.PrefixStats
	var lastErr error
	
	for _, cmd := range commands {
		b, err := birdsocket.Query(sock, cmd)
		if err != nil {
			lastErr = err
			continue
		}
		
		stats = parser.ParsePrefixStats(proto.Name, proto.IPVersion, b)
		
		// If we got a reasonable number of routes, use this result
		totalRoutes := int64(0)
		for _, count := range stats.PrefixLengthCounts {
			totalRoutes += count
		}
		
		// If we found routes, return this result
		if totalRoutes > 0 {
			return stats, nil
		}
	}
	
	// If no command worked, return the last attempt or error
	if stats != nil {
		return stats, nil
	}
	
	return nil, lastErr
}

// GetAllPrefixStats retrieves prefix length statistics for all routes in a table
func (c *BirdClient) GetAllPrefixStats(ipVersion string) (*protocol.PrefixStats, error) {
	sock := c.socketFor(ipVersion)
	
	tableName := "master4"
	if ipVersion == "6" {
		tableName = "master6"
	}
	
	// Use count-based approach for large datasets since each route generates ~4 lines
	countStats, err := c.getCountBasedPrefixStats(sock, tableName, ipVersion)
	if err == nil && countStats != nil {
		totalRoutes := int64(0)
		for _, count := range countStats.PrefixLengthCounts {
			totalRoutes += count
		}
		if totalRoutes > 0 {
			return countStats, nil
		}
	}
	
	// Fallback to sampling approach for very large datasets
	sampleStats, err := c.getSampledPrefixStats(sock, tableName, ipVersion)
	if err == nil && sampleStats != nil {
		return sampleStats, nil
	}
	
	return nil, fmt.Errorf("unable to get prefix stats: count failed (%v), sampling failed (%v)", err, err)
}

// getCountBasedPrefixStats uses BIRD's count functionality to efficiently get prefix statistics
func (c *BirdClient) getCountBasedPrefixStats(sock, tableName, ipVersion string) (*protocol.PrefixStats, error) {
	stats := protocol.NewPrefixStats(ipVersion, "all_routes")
	
	// Define prefix length ranges to query - use all possible lengths
	var prefixLengths []int
	if ipVersion == "6" {
		// All IPv6 prefix lengths from /1 to /128
		for i := 1; i <= 128; i++ {
			prefixLengths = append(prefixLengths, i)
		}
	} else {
		// All IPv4 prefix lengths from /1 to /32
		for i := 1; i <= 32; i++ {
			prefixLengths = append(prefixLengths, i)
		}
	}
	
	// Query each prefix length using count command
	for _, prefixLen := range prefixLengths {
		var cmd string
		if ipVersion == "6" {
			cmd = fmt.Sprintf("show route table %s where net ~ [::/0{%d,%d}] primary count", tableName, prefixLen, prefixLen)
		} else {
			cmd = fmt.Sprintf("show route table %s where net ~ [0.0.0.0/0{%d,%d}] primary count", tableName, prefixLen, prefixLen)
		}
		
		b, err := birdsocket.Query(sock, cmd)
		if err != nil {
			continue // Skip failed queries
		}
		
		count := parseRouteCount(b)
		if count > 0 {
			stats.PrefixLengthCounts[prefixLen] = count
		}
	}
	
	return stats, nil
}

// getSampledPrefixStats uses sampling to estimate prefix distribution for very large datasets
func (c *BirdClient) getSampledPrefixStats(sock, tableName, ipVersion string) (*protocol.PrefixStats, error) {
	stats := protocol.NewPrefixStats(ipVersion, "all_routes")
	
	// Try to get a sample of routes - BIRD doesn't support limit keyword
	var cmd string
	if ipVersion == "6" {
		cmd = fmt.Sprintf("show route table %s", tableName)
	} else {
		cmd = fmt.Sprintf("show route table %s", tableName)
	}
	
	b, err := birdsocket.Query(sock, cmd)
	if err != nil {
		// Try simpler command
		simpleCmd := "show route"
		b, err = birdsocket.Query(sock, simpleCmd)
		if err != nil {
			return nil, err
		}
	}
	
	sampleStats := parser.ParsePrefixStats("sample", ipVersion, b)
	
	// Get total route count for scaling
	totalCmd := fmt.Sprintf("show route table %s count", tableName)
	totalBytes, err := birdsocket.Query(sock, totalCmd)
	if err != nil {
		return sampleStats, nil // Return sample without scaling
	}
	
	totalCount := parseRouteCount(totalBytes)
	if totalCount > 0 {
		// Scale sample to full population
		sampleTotal := int64(0)
		for _, count := range sampleStats.PrefixLengthCounts {
			sampleTotal += count
		}
		
		if sampleTotal > 0 {
			scaleFactor := float64(totalCount) / float64(sampleTotal)
			for prefixLen, count := range sampleStats.PrefixLengthCounts {
				stats.PrefixLengthCounts[prefixLen] = int64(float64(count) * scaleFactor)
			}
		}
	}
	
	return stats, nil
}

// parseRouteCount extracts the route count from BIRD's count output
func parseRouteCount(data []byte) int64 {
	output := string(data)
	lines := strings.Split(output, "\n")
	
	// Look for BIRD v2 format: "1007-197991 of 440662 routes for 220785 networks in table master6"
	// The format is: error_code-count of total_routes routes for networks in table
	// We want the number after the dash (filtered routes matching our criteria)
	countRegexV2 := regexp.MustCompile(`^(\d+)-(\d+)\s+of\s+\d+\s+routes\s+for\s+\d+\s+networks\s+in\s+table`)
	
	// Also support simple format: "42 routes" or "0 routes"
	countRegex := regexp.MustCompile(`^(\d+)\s+routes?`)
	
	for _, line := range lines {
		line = strings.TrimSpace(line)
		
		// Try BIRD v2 format first
		if match := countRegexV2.FindStringSubmatch(line); match != nil {
			if count, err := strconv.ParseInt(match[2], 10, 64); err == nil {
				return count
			}
		}
		
		// Fallback to simple format
		if match := countRegex.FindStringSubmatch(line); match != nil {
			if count, err := strconv.ParseInt(match[1], 10, 64); err == nil {
				return count
			}
		}
	}
	
	return 0
}

// getRouteSource converts protocol type to BIRD route source
func getRouteSource(proto protocol.Proto) string {
	switch proto {
	case protocol.BGP:
		return "BGP"
	case protocol.OSPF:
		return "OSPF"
	case protocol.Kernel:
		return "KERNEL"
	case protocol.Static:
		return "STATIC"
	case protocol.Direct:
		return "DIRECT"
	case protocol.Babel:
		return "BABEL"
	default:
		return "BGP" // Default fallback
	}
}

func (c *BirdClient) protocolsFromBird(ipVersions []string) ([]*protocol.Protocol, error) {
	protocols := make([]*protocol.Protocol, 0)

	for _, ipVersion := range ipVersions {
		sock := c.socketFor(ipVersion)
		s, err := c.protocolsFromSocket(sock, ipVersion)
		if err != nil {
			return nil, err
		}

		protocols = append(protocols, s...)
	}

	return protocols, nil
}

func (c *BirdClient) protocolsFromSocket(socketPath string, ipVersion string) ([]*protocol.Protocol, error) {
	b, err := birdsocket.Query(socketPath, "show protocols all")
	if err != nil {
		return nil, err
	}

	return parser.ParseProtocols(b, ipVersion), nil
}

func (c *BirdClient) socketFor(ipVersion string) string {
	if !c.Options.BirdV2 && ipVersion == "6" {
		return c.Options.Bird6Socket
	}

	return c.Options.BirdSocket
}
