package parser

import (
	"testing"

	"github.com/czerwonk/testutils/assert"
)

func TestBirdV2RouteFormat(t *testing.T) {
	// Test with real BIRD v2 output format
	data := []byte(`BIRD 2.0.12 ready.
Table master6:
2800:200:ea00::/48   unicast [JC01_NYC01 13:09:51.102 from 2a0e:97c0:e61:ff80::101] * (100) [AS12252i]
        dev nyc01
                     unicast [JC01_ASH01 13:09:46.383 from 2a0e:97c0:e61:ff80::9] (100) [AS12252i]
        dev ash01
2600:380:180::/41    unicast [JC01_NYC01 02:52:09.242 from 2a0e:97c0:e61:ff80::101] * (100) [AS20057i]
        dev nyc01
                     unicast [JC01_ASH01 2025-07-24 from 2a0e:97c0:e61:ff80::9] (100) [AS20057i]
        dev ash01
2804:2404:8000::/34  unicast [JC01_NYC01 02:52:09.242 from 2a0e:97c0:e61:ff80::101] * (100) [AS264197i]
        dev nyc01
                     unicast [JC01_ASH01 2025-07-24 from 2a0e:97c0:e61:ff80::9] (100) [AS264197i]
        dev ash01
2400:3800:8800::/37  unicast [JC01_NYC01 02:52:09.242 from 2a0e:97c0:e61:ff80::101] * (100) [AS9617i]
        dev nyc01
                     unicast [JC01_ASH01 2025-07-24 from 2a0e:97c0:e61:ff80::9] (100) [AS9617i]
        dev ash01
2400:54c0:c0::/44    unicast [JC01_NYC01 02:52:09.242 from 2a0e:97c0:e61:ff80::101] * (100) [AS136352i]
        dev nyc01`)

	stats := ParsePrefixStats("all_routes", "6", data)

	// Should find 5 unique prefixes: /48, /41, /34, /37, /44
	expectedCounts := map[int]int{
		48: 1, // 2800:200:ea00::/48
		41: 1, // 2600:380:180::/41
		34: 1, // 2804:2404:8000::/34
		37: 1, // 2400:3800:8800::/37
		44: 1, // 2400:54c0:c0::/44
	}

	for prefixLen, expectedCount := range expectedCounts {
		actualCount := int(stats.PrefixLengthCounts[prefixLen])
		assert.IntEqual("prefix_length_"+string(rune(prefixLen)), expectedCount, actualCount, t)
	}

	// Calculate total routes
	totalActual := int64(0)
	for _, count := range stats.PrefixLengthCounts {
		totalActual += count
	}

	assert.IntEqual("total_routes", 5, int(totalActual), t)
}

func TestExtractPrefixLengthBirdV2(t *testing.T) {
	testCases := []struct {
		name string
		line string
		expected int
	}{
		{"bird_v2_route", "2800:200:ea00::/48   unicast [JC01_NYC01 13:09:51.102 from 2a0e:97c0:e61:ff80::101] * (100) [AS12252i]", 48},
		{"bird_v2_route_41", "2600:380:180::/41    unicast [JC01_NYC01 02:52:09.242 from 2a0e:97c0:e61:ff80::101] * (100) [AS20057i]", 41},
		{"bird_v2_route_34", "2804:2404:8000::/34  unicast [JC01_NYC01 02:52:09.242 from 2a0e:97c0:e61:ff80::101] * (100) [AS264197i]", 34},
		{"bird_v2_route_37", "2400:3800:8800::/37  unicast [JC01_NYC01 02:52:09.242 from 2a0e:97c0:e61:ff80::101] * (100) [AS9617i]", 37},
		{"bird_v2_route_44", "2400:54c0:c0::/44    unicast [JC01_NYC01 02:52:09.242 from 2a0e:97c0:e61:ff80::101] * (100) [AS136352i]", 44},
		{"continuation_line", "        dev nyc01", 0},
		{"indented_unicast", "                     unicast [JC01_ASH01 13:09:46.383 from 2a0e:97c0:e61:ff80::9] (100) [AS12252i]", 0},
		{"table_header", "Table master6:", 0},
		{"bird_ready", "BIRD 2.0.12 ready.", 0},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := extractPrefixLength(tc.line)
			assert.IntEqual("prefix_length", tc.expected, result, t)
		})
	}
}