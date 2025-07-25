package metrics

import (
	"strconv"

	"github.com/czerwonk/bird_exporter/client"
	"github.com/czerwonk/bird_exporter/protocol"
	"github.com/prometheus/client_golang/prometheus"
	log "github.com/sirupsen/logrus"
)

// TablePrefixSizeExporter exports metrics for prefix length distribution across entire routing table
type TablePrefixSizeExporter struct {
	client client.Client
	prefix string
}

// NewTablePrefixSizeExporter creates a new instance of TablePrefixSizeExporter
func NewTablePrefixSizeExporter(prefix string, c client.Client) *TablePrefixSizeExporter {
	return &TablePrefixSizeExporter{
		client: c,
		prefix: prefix,
	}
}

func (m *TablePrefixSizeExporter) Describe(ch chan<- *prometheus.Desc) {
	// Descriptions are created dynamically based on the actual prefix lengths found
}

func (m *TablePrefixSizeExporter) Export(p *protocol.Protocol, ch chan<- prometheus.Metric, newFormat bool) {
	// This exporter works per IP version, not per protocol
	// We'll use the protocol's IP version to determine which table to query
	stats, err := m.client.GetAllPrefixStats(p.IPVersion)
	if err != nil {
		log.WithError(err).WithField("ip_version", p.IPVersion).Error("Failed to get table-wide prefix statistics")
		return
	}

	labelNames := []string{"ip_version", "prefix_length", "table"}
	
	var desc *prometheus.Desc
	if newFormat {
		desc = prometheus.NewDesc(
			m.prefix+"_table_prefix_length_count",
			"Number of unique prefixes by prefix length in routing table",
			labelNames,
			nil,
		)
	} else {
		desc = prometheus.NewDesc(
			m.prefix+"_table_prefix_count_by_length",
			"Number of unique prefixes by prefix length in routing table",
			labelNames,
			nil,
		)
	}

	tableName := "master4"
	if p.IPVersion == "6" {
		tableName = "master6"
	}

	// Export metrics for each prefix length that has routes
	for prefixLen, count := range stats.PrefixLengthCounts {
		labelValues := []string{
			p.IPVersion,
			strconv.Itoa(prefixLen),
			tableName,
		}

		ch <- prometheus.MustNewConstMetric(
			desc,
			prometheus.GaugeValue,
			float64(count),
			labelValues...,
		)
	}
}