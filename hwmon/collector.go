package hwmon

import (
	"regexp"
	"strconv"
	"strings"

	"github.com/R4scal/zyxel_gs1920_exporter/collector"
	"github.com/prometheus/client_golang/prometheus"
	"golang.org/x/net/html"
)

const (
	prefix string = "zyxel_hwmon_"
	uri    string = "/US/1/rpsysinfo.html"
)

var (
	sensorValue  *prometheus.Desc
	sensorStatus *prometheus.Desc

	rFan         = regexp.MustCompile(`^FAN\d$`)
	rTemperature = regexp.MustCompile(`^(BOARD|MAC|PHY)$`)
	rVoltage     = regexp.MustCompile(`^\d+\.?\d+?V$`)
)

func init() {
	l := []string{"target", "sensor"}

	sensorValue = prometheus.NewDesc(prefix+"sensor_value", "Current sensor value", l, nil)
	sensorStatus = prometheus.NewDesc(prefix+"sensor_status", "Current sensor status", l, nil)
}

type hwmonCollector struct {
}

// NewCollector creates a new collector
func NewCollector() collector.Collector {
	return &hwmonCollector{}
}

func (*hwmonCollector) GetURI() string {
	return uri
}

// Describe describes the metrics
func (*hwmonCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- sensorValue
	ch <- sensorStatus
}

// Collect collects metrics
func (c *hwmonCollector) Collect(z *html.Tokenizer, ch chan<- prometheus.Metric, l []string) error {
	table := [][]string{}
	row := []string{}
	for z.Token().Data != "html" {
		tt := z.Next()
		if tt == html.StartTagToken {
			t := z.Token()

			if t.Data == "tr" {
				if len(row) > 0 {
					table = append(table, row)
					row = []string{}
				}
			}

			if t.Data == "td" {
				inner := z.Next()

				if inner == html.TextToken {
					text := (string)(z.Text())
					t := strings.TrimSpace(text)
					row = append(row, t)
				}
			}

		}
	}
	if len(row) > 0 {
		table = append(table, row)
	}

	for _, line := range table {
		//log.Info(strings.Join(line, "|"), " ", len(line))
		// sensors table
		if len(line) == 6 {
			// sensor
			if rVoltage.MatchString(line[0]) || rFan.MatchString(line[0]) || rTemperature.MatchString(line[0]) {
				ls := append(l, line[0])
				// value
				if val, err := strconv.ParseFloat(line[1], 64); err == nil {
					ch <- prometheus.MustNewConstMetric(sensorValue, prometheus.GaugeValue, val, ls...)
				}
				// status
				var status float64 = 1
				if line[5] == "Normal" {
					status = 0
				}
				ch <- prometheus.MustNewConstMetric(sensorStatus, prometheus.GaugeValue, status, ls...)
			}
		}

	}
	return nil
}
