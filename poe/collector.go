package poe

import (
	"strconv"
	"strings"

	"github.com/R4scal/zyxel_gs1920_exporter/collector"
	"github.com/prometheus/client_golang/prometheus"
	"golang.org/x/net/html"
)

const (
	prefix string = "zyxel_poe_"
	uri    string = "/US/1/rppoestatus.html"
)

var (
	powerConsuming *prometheus.Desc
	powerMax       *prometheus.Desc
)

func init() {
	l := []string{"target", "ifIndex"}

	powerConsuming = prometheus.NewDesc(prefix+"power_consuming", "Consuming Power (mW)", l, nil)
	powerMax = prometheus.NewDesc(prefix+"power_max", "Max Power (mW)", l, nil)
}

type poeCollector struct {
}

// NewCollector creates a new collector
func NewCollector() collector.Collector {
	return &poeCollector{}
}

func (*poeCollector) GetURI() string {
	return uri
}

// Describe describes the metrics
func (*poeCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- powerConsuming
	ch <- powerMax
}

// Collect collects metrics
func (c *poeCollector) Collect(z *html.Tokenizer, ch chan<- prometheus.Metric, l []string) error {
	table := [][]string{}
	row := []string{}
	var curToken string
loop:
	for {
		tt := z.Next()
		switch tt {
		case html.ErrorToken:
			break loop
		case html.TextToken:
			if curToken == "td" || curToken == "div" {
				text := (string)(z.Text())
				t := strings.TrimSpace(text)
				row = append(row, t)
			}
		case html.StartTagToken:
			t := z.Token()
			if t.Data == "tr" {
				if len(row) > 0 {
					table = append(table, row)
					row = []string{}
				}
			}
			curToken = t.Data
		}
	}

	if len(row) > 0 {
		table = append(table, row)
	}

	for _, line := range table {
		//log.Info(strings.Join(line, "|"), " ", len(line))
		// port table
		if len(line) == 35 {
			// skip disabled ports
			if line[6] == "Disable" {
				continue
			}

			idx := line[2]
			// current
			if current, err := strconv.ParseFloat(line[22], 64); err == nil {
				ch <- prometheus.MustNewConstMetric(powerConsuming, prometheus.GaugeValue, current, append(l, idx)...)
			}
			// max
			if max, err := strconv.ParseFloat(line[26], 64); err == nil {
				ch <- prometheus.MustNewConstMetric(powerMax, prometheus.GaugeValue, max, append(l, idx)...)
			}
		}

	}
	return nil
}
