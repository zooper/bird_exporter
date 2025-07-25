package parser

import (
	"fmt"
	"strings"
	"testing"

	"github.com/czerwonk/testutils/assert"
)

func TestRealisticIPv6BGPTable(t *testing.T) {
	// Simulate realistic IPv6 BGP table output with various route formats
	routeLines := []string{
		"BIRD 2.0.8 ready.",
		"Table master6:",
		"",
		// Standard routes with different prefix lengths
		"2001:db8::/32       via 2001:db8::1 on eth0 [bgp1 12:34:56] * (100) [AS65001i]",
		"2001:470::/32       via 2001:db8::1 on eth0 [bgp1 12:34:56] * (100) [AS6939i]",
		"2001:4860::/32      via 2001:db8::1 on eth0 [bgp1 12:34:56] * (100) [AS15169i]",
		
		// More /32 routes (common for IPv6)
		"2400:cb00::/32      via 2001:db8::1 on eth0 [bgp1 12:34:56] * (100) [AS13335i]",
		"2606:4700::/32      via 2001:db8::1 on eth0 [bgp1 12:34:56] * (100) [AS13335i]",
		"2a00:1450::/32      via 2001:db8::1 on eth0 [bgp1 12:34:56] * (100) [AS15169i]",
		
		// /48 routes (common for smaller allocations)
		"2001:db8:1::/48     via 2001:db8::1 on eth0 [bgp1 12:34:56] * (100) [AS65001i]",
		"2001:db8:2::/48     via 2001:db8::1 on eth0 [bgp1 12:34:56] * (100) [AS65001i]",
		"2001:db8:3::/48     via 2001:db8::1 on eth0 [bgp1 12:34:56] * (100) [AS65001i]",
		"2001:db8:4::/48     via 2001:db8::1 on eth0 [bgp1 12:34:56] * (100) [AS65001i]",
		
		// /64 routes (subnet level)
		"2001:db8:100::/64   via 2001:db8::1 on eth0 [bgp1 12:34:56] * (100) [AS65001i]",
		"2001:db8:101::/64   via 2001:db8::1 on eth0 [bgp1 12:34:56] * (100) [AS65001i]",
		"2001:db8:102::/64   via 2001:db8::1 on eth0 [bgp1 12:34:56] * (100) [AS65001i]",
		
		// Different route types
		"fc00::/7            unreachable [kernel1 12:34:56] * (200) [i]",
		"fe80::/10           dev eth0 [kernel1 12:34:56] * (240) [i]",
		"::1/128             dev lo [kernel1 12:34:56] * (240) [i]",
		
		// Routes without 'via' (blackhole, unreachable, etc.)
		"2001:db8:dead::/48  blackhole [static1 12:34:56] * (200)",
		"2001:db8:beef::/48  unreachable [static1 12:34:56] * (200)",
		
		// Multi-line routes (some BIRD versions split long lines)
		"2001:0db8:85a3:0000:0000:8a2e:0370:7334/128",
		"                    via 2001:db8::1 on eth0 [bgp1 12:34:56] * (100) [AS65001i]",
		
		// Compressed IPv6 notation
		"2001:db8::1/128     via 2001:db8::1 on eth0 [bgp1 12:34:56] * (100) [AS65001i]",
		"::1/128             dev lo [direct1 12:34:56] * (240)",
		
		// Different prefix lengths to simulate diversity
		"2001:db8:8000::/33  via 2001:db8::1 on eth0 [bgp1 12:34:56] * (100) [AS65001i]",
		"2001:db8:c000::/34  via 2001:db8::1 on eth0 [bgp1 12:34:56] * (100) [AS65001i]",
		"2001:db8:e000::/35  via 2001:db8::1 on eth0 [bgp1 12:34:56] * (100) [AS65001i]",
		"2001:db8:f000::/36  via 2001:db8::1 on eth0 [bgp1 12:34:56] * (100) [AS65001i]",
	}
	
	data := []byte(strings.Join(routeLines, "\n"))
	stats := ParsePrefixStats("bgp1", "6", data)
	
	// Count expected routes
	expectedCounts := map[int]int{
		7:   1, // fc00::/7
		10:  1, // fe80::/10
		32:  6, // Various /32 routes
		33:  1, // /33 route
		34:  1, // /34 route  
		35:  1, // /35 route
		36:  1, // /36 route
		48:  6, // Various /48 routes (4 regular + 2 blackhole/unreachable)
		64:  3, // /64 routes
		128: 4, // /128 routes (::1 twice, 2001:db8::1, and full IPv6)
	}
	
	// Verify we parsed the expected number of routes for each prefix length
	for prefixLen, expectedCount := range expectedCounts {
		actualCount := int(stats.PrefixLengthCounts[prefixLen])
		assert.IntEqual(fmt.Sprintf("prefix_length_%d", prefixLen), expectedCount, actualCount, t)
	}
	
	// Calculate total routes
	totalExpected := 0
	for _, count := range expectedCounts {
		totalExpected += count
	}
	
	totalActual := int64(0)
	for _, count := range stats.PrefixLengthCounts {
		totalActual += count
	}
	
	assert.IntEqual("total_routes", totalExpected, int(totalActual), t)
	assert.StringEqual("protocol", "bgp1", stats.Protocol, t)
	assert.StringEqual("ip_version", "6", stats.IPVersion, t)
}

func TestLargeScaleSimulation(t *testing.T) {
	// Simulate a scenario closer to 200k routes
	var routeLines []string
	routeLines = append(routeLines, "BIRD 2.0.8 ready.")
	routeLines = append(routeLines, "Table master6:")
	routeLines = append(routeLines, "")
	
	expectedCounts := map[int]int{}
	
	// Generate many /48 routes (typical for IPv6)
	for i := 0; i < 1000; i++ {
		route := fmt.Sprintf("2001:db8:%x::/48     via 2001:db8::1 on eth0 [bgp1 12:34:56] * (100) [AS65001i]", i)
		routeLines = append(routeLines, route)
		expectedCounts[48]++
	}
	
	// Generate many /32 routes (ISP allocations)
	for i := 0; i < 500; i++ {
		route := fmt.Sprintf("240%x::/32          via 2001:db8::1 on eth0 [bgp1 12:34:56] * (100) [AS%di]", i, 65000+i)
		routeLines = append(routeLines, route)
		expectedCounts[32]++
	}
	
	// Generate /64 routes (subnet level)
	for i := 0; i < 300; i++ {
		route := fmt.Sprintf("2001:db8:1000:%x::/64 via 2001:db8::1 on eth0 [bgp1 12:34:56] * (100) [AS65001i]", i)
		routeLines = append(routeLines, route)
		expectedCounts[64]++
	}
	
	data := []byte(strings.Join(routeLines, "\n"))
	stats := ParsePrefixStats("bgp1", "6", data)
	
	// Verify we got the expected counts
	for prefixLen, expectedCount := range expectedCounts {
		actualCount := int(stats.PrefixLengthCounts[prefixLen])
		assert.IntEqual(fmt.Sprintf("prefix_length_%d", prefixLen), expectedCount, actualCount, t)
	}
	
	// Total should be 1800 routes
	totalActual := int64(0)
	for _, count := range stats.PrefixLengthCounts {
		totalActual += count
	}
	
	assert.IntEqual("total_routes", 1800, int(totalActual), t)
}