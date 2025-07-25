package parser

import (
	"bufio"
	"bytes"
	"regexp"
	"strconv"
	"strings"

	"github.com/czerwonk/bird_exporter/protocol"
)

var (
	routeLineRegex *regexp.Regexp
)

func init() {
	// Matches route lines like:
	// 192.168.1.0/24      via 10.0.0.1 on eth0 [bgp1 12:34:56] * (100) [AS65001i]
	// 2001:db8::/32       via 2001:db8::1 on eth0 [bgp1 12:34:56] * (100) [AS65001i]
	routeLineRegex = regexp.MustCompile(`^([0-9a-fA-F:./]+)(?:/(\d+))?\s+via\s+([0-9a-fA-F:.]+)\s+on\s+\S+\s+\[(\S+)\s+[^\]]+\]\s*[*]?\s*\((\d+)\)`)
}

// ParsePrefixStats parses BIRD route output and returns prefix length statistics
func ParsePrefixStats(protocolName, ipVersion string, data []byte) *protocol.PrefixStats {
	stats := protocol.NewPrefixStats(ipVersion, protocolName)
	reader := bytes.NewReader(data)
	scanner := bufio.NewScanner(reader)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		// Skip header and status lines
		if strings.HasPrefix(line, "BIRD") || 
		   strings.HasPrefix(line, "Access restricted") ||
		   strings.Contains(line, "Table") ||
		   strings.Contains(line, "Preference") {
			continue
		}

		prefixLen := extractPrefixLength(line)
		if prefixLen > 0 {
			stats.AddRoute(prefixLen)
		}
	}

	return stats
}

// extractPrefixLength extracts the prefix length from a route line
func extractPrefixLength(line string) int {
	// Try to match full route line format first
	match := routeLineRegex.FindStringSubmatch(line)
	if match != nil && len(match) > 2 && match[2] != "" {
		if prefixLen, err := strconv.Atoi(match[2]); err == nil {
			return prefixLen
		}
	}

	// Fallback: look for prefix/length pattern anywhere in the line
	prefixRegex := regexp.MustCompile(`(?:^|\s)([0-9a-fA-F:.]+)/(\d+)(?:\s|$)`)
	match = prefixRegex.FindStringSubmatch(line)
	if match != nil && len(match) > 2 {
		if prefixLen, err := strconv.Atoi(match[2]); err == nil {
			return prefixLen
		}
	}

	return 0
}