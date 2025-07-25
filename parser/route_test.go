package parser

import (
	"testing"

	"github.com/czerwonk/testutils/assert"
)

func TestParsePrefixStats(t *testing.T) {
	data := []byte(`BIRD 1.6.8 ready.
192.168.1.0/24      via 10.0.0.1 on eth0 [bgp1 12:34:56] * (100) [AS65001i]
10.0.0.0/8          via 192.168.1.1 on eth1 [bgp1 12:34:56] * (100) [AS65001i]
172.16.0.0/16       via 10.0.0.2 on eth0 [bgp1 12:34:56] * (100) [AS65001i]
192.168.2.0/24      via 10.0.0.1 on eth0 [bgp1 12:34:56] * (100) [AS65001i]
203.0.113.0/24      via 10.0.0.3 on eth0 [static1 12:34:56] * (200)
2001:db8::/32       via 2001:db8::1 on eth0 [bgp1 12:34:56] * (100) [AS65001i]
`)

	stats := ParsePrefixStats("bgp1", "4", data)

	assert.StringEqual("protocol", "bgp1", stats.Protocol, t)
	assert.StringEqual("ip_version", "4", stats.IPVersion, t)

	// Should have 3 different prefix lengths: /8, /16, /24, /32
	assert.IntEqual("prefix_length_8_count", 1, int(stats.PrefixLengthCounts[8]), t)
	assert.IntEqual("prefix_length_16_count", 1, int(stats.PrefixLengthCounts[16]), t)
	assert.IntEqual("prefix_length_24_count", 3, int(stats.PrefixLengthCounts[24]), t)
	assert.IntEqual("prefix_length_32_count", 1, int(stats.PrefixLengthCounts[32]), t)
}

func TestParsePrefixStatsIPv6(t *testing.T) {
	data := []byte(`BIRD 2.0.8 ready.
2001:db8::/32       via 2001:db8::1 on eth0 [bgp1 12:34:56] * (100) [AS65001i]
2001:db8:1::/48     via 2001:db8::1 on eth0 [bgp1 12:34:56] * (100) [AS65001i]
2001:db8:2::/48     via 2001:db8::1 on eth0 [bgp1 12:34:56] * (100) [AS65001i]
2001:db8:3:1::/64   via 2001:db8::1 on eth0 [bgp1 12:34:56] * (100) [AS65001i]
`)

	stats := ParsePrefixStats("bgp1", "6", data)

	assert.StringEqual("protocol", "bgp1", stats.Protocol, t)
	assert.StringEqual("ip_version", "6", stats.IPVersion, t)

	assert.IntEqual("prefix_length_32_count", 1, int(stats.PrefixLengthCounts[32]), t)
	assert.IntEqual("prefix_length_48_count", 2, int(stats.PrefixLengthCounts[48]), t)
	assert.IntEqual("prefix_length_64_count", 1, int(stats.PrefixLengthCounts[64]), t)
}

func TestExtractPrefixLength(t *testing.T) {
	testCases := []struct {
		line     string
		expected int
	}{
		{"192.168.1.0/24      via 10.0.0.1 on eth0 [bgp1 12:34:56] * (100)", 24},
		{"10.0.0.0/8          via 192.168.1.1 on eth1 [bgp1 12:34:56] * (100)", 8},
		{"172.16.0.0/16       via 10.0.0.2 on eth0 [bgp1 12:34:56] * (100)", 16},
		{"2001:db8::/32       via 2001:db8::1 on eth0 [bgp1 12:34:56] * (100)", 32},
		{"2001:db8:1::/48     via 2001:db8::1 on eth0 [bgp1 12:34:56] * (100)", 48},
		{"192.168.1.1/32      blackhole [static1 12:34:56] * (200)", 32},
		{"BIRD 1.6.8 ready.", 0},
		{"Access restricted", 0},
		{"", 0},
		{"invalid line", 0},
	}

	for _, tc := range testCases {
		result := extractPrefixLength(tc.line)
		assert.IntEqual("prefix_length for: "+tc.line, tc.expected, result, t)
	}
}

func TestParsePrefixStatsEmpty(t *testing.T) {
	data := []byte(`BIRD 1.6.8 ready.
Access restricted
`)

	stats := ParsePrefixStats("bgp1", "4", data)

	assert.StringEqual("protocol", "bgp1", stats.Protocol, t)
	assert.StringEqual("ip_version", "4", stats.IPVersion, t)
	assert.IntEqual("prefix_counts_length", 0, len(stats.PrefixLengthCounts), t)
}