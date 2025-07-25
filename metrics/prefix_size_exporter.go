package metrics

import (
	"strconv"

	"github.com/czerwonk/bird_exporter/client"
	"github.com/czerwonk/bird_exporter/protocol"
	"github.com/prometheus/client_golang/prometheus"
	log "github.com/sirupsen/logrus"
)

// PrefixSizeExporter exports metrics for prefix length distribution
type PrefixSizeExporter struct {
	client client.Client
	prefix string
}

// NewPrefixSizeExporter creates a new instance of PrefixSizeExporter
func NewPrefixSizeExporter(prefix string, c client.Client) *PrefixSizeExporter {
	return &PrefixSizeExporter{
		client: c,
		prefix: prefix,
	}
}

func (m *PrefixSizeExporter) Describe(ch chan<- *prometheus.Desc) {
	// Descriptions are created dynamically based on the actual prefix lengths found
}

func (m *PrefixSizeExporter) Export(p *protocol.Protocol, ch chan<- prometheus.Metric, newFormat bool) {
	stats, err := m.client.GetPrefixStats(p)
	if err != nil {
		log.WithError(err).WithField("protocol", p.Name).Error("Failed to get prefix statistics")
		return
	}

	labelNames := []string{"name", "proto", "ip_version", "prefix_length"}
	
	var desc *prometheus.Desc
	if newFormat {
		desc = prometheus.NewDesc(
			m.prefix+"_prefix_length_count",
			"Number of prefixes by prefix length",
			labelNames,
			nil,
		)
	} else {
		desc = prometheus.NewDesc(
			m.prefix+"_prefix_count_by_length",
			"Number of prefixes by prefix length",
			labelNames,
			nil,
		)
	}

	// Export metrics for each prefix length that has routes
	for prefixLen, count := range stats.PrefixLengthCounts {
		labelValues := []string{
			p.Name,
			protocolTypeToString(p.Proto),
			p.IPVersion,
			intToString(prefixLen),
		}

		ch <- prometheus.MustNewConstMetric(
			desc,
			prometheus.GaugeValue,
			float64(count),
			labelValues...,
		)
	}
}

func protocolTypeToString(proto protocol.Proto) string {
	switch proto {
	case protocol.BGP:
		return "BGP"
	case protocol.OSPF:
		return "OSPF"
	case protocol.Direct:
		return "Direct"
	case protocol.Kernel:
		return "Kernel"
	case protocol.Static:
		return "Static"
	case protocol.Babel:
		return "Babel"
	case protocol.RPKI:
		return "RPKI"
	case protocol.BFD:
		return "BFD"
	default:
		return "Unknown"
	}
}

func intToString(i int) string {
	return strconv.Itoa(i)
}