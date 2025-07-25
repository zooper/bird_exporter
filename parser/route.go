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

	lineCount := 0
	routeCount := 0

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		lineCount++
		
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
			routeCount++
		}
	}

	// Log basic statistics for debugging
	totalRoutes := int64(0)
	for _, count := range stats.PrefixLengthCounts {
		totalRoutes += count
	}

	return stats
}

// extractPrefixLength extracts the prefix length from a route line
func extractPrefixLength(line string) int {
	// Skip obvious non-route lines first
	if line == "" || 
		strings.HasPrefix(line, "BIRD") ||
		strings.HasPrefix(line, "Access") ||
		strings.HasPrefix(line, "Table") ||
		strings.Contains(line, "BGP") && strings.Contains(line, "up") ||
		strings.Contains(line, "---") ||
		strings.HasPrefix(line, "        ") { // Skip indented continuation lines
		return 0
	}

	// BIRD v2 route format: "2800:200:ea00::/48   unicast [...]"
	// Try multiple patterns for BIRD v2
	birdV2Patterns := []*regexp.Regexp{
		regexp.MustCompile(`^([0-9a-fA-F:.]+)/(\d{1,3})\s+unicast`),   // Standard unicast
		regexp.MustCompile(`^([0-9a-fA-F:.]+)/(\d{1,3})\s+blackhole`), // Blackhole routes
		regexp.MustCompile(`^([0-9a-fA-F:.]+)/(\d{1,3})\s+unreachable`), // Unreachable routes
		regexp.MustCompile(`^([0-9a-fA-F:.]+)/(\d{1,3})\s+`),          // Any route at start of line
	}
	
	for _, pattern := range birdV2Patterns {
		if match := pattern.FindStringSubmatch(line); match != nil && len(match) > 2 {
			if prefixLen, err := strconv.Atoi(match[2]); err == nil {
				// Validate reasonable prefix lengths
				if (strings.Contains(match[1], ":") && prefixLen <= 128) || // IPv6
				   (!strings.Contains(match[1], ":") && prefixLen <= 32) {   // IPv4
					return prefixLen
				}
			}
		}
	}

	// Fallback patterns for other formats
	prefixPatterns := []*regexp.Regexp{
		// IPv4: 192.168.1.0/24 at start of line
		regexp.MustCompile(`^(\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3})/(\d{1,2})\s`),
		// IPv6: 2001:db8::/32 at start of line  
		regexp.MustCompile(`^([0-9a-fA-F:]+::[0-9a-fA-F:]*)/(\d{1,3})\s`),
		// IPv6 full format at start of line
		regexp.MustCompile(`^([0-9a-fA-F]{1,4}:[0-9a-fA-F:]+)/(\d{1,3})\s`),
		// Generic prefix/length pattern anywhere (fallback)
		regexp.MustCompile(`\b([0-9a-fA-F:.]+)/(\d{1,3})\b`),
	}

	for _, regex := range prefixPatterns {
		if match := regex.FindStringSubmatch(line); match != nil && len(match) > 2 {
			if prefixLen, err := strconv.Atoi(match[2]); err == nil {
				// Validate reasonable prefix lengths
				if (strings.Contains(match[1], ":") && prefixLen <= 128) || // IPv6
				   (!strings.Contains(match[1], ":") && prefixLen <= 32) {   // IPv4
					return prefixLen
				}
			}
		}
	}

	return 0
}