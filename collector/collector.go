package collector

import (
	"github.com/prometheus/client_golang/prometheus"
	"golang.org/x/net/html"
)

// Collector collects metrics from JunOS using rpc.Client
type Collector interface {

	// Describe describes the metrics
	Describe(ch chan<- *prometheus.Desc)

	// Collect collects metrics from JunOS
	Collect(z *html.Tokenizer, ch chan<- prometheus.Metric, l []string) error

	// GetURI get uri for http request
	GetURI() string
}
