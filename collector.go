package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/R4scal/zyxel_gs1920_exporter/collector"
	"github.com/R4scal/zyxel_gs1920_exporter/config"
	"github.com/R4scal/zyxel_gs1920_exporter/hwmon"
	"github.com/R4scal/zyxel_gs1920_exporter/poe"
	"golang.org/x/net/html"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/common/log"
)

const (
	loginURI = "/FirstPage.html"
	prefix   = "zyxel_"
)

var (
	scrapeCollectorDurationDesc *prometheus.Desc
	scrapeDurationDesc          *prometheus.Desc
	upDesc                      *prometheus.Desc

	cookies map[string]*http.Cookie
)

func init() {
	upDesc = prometheus.NewDesc(prefix+"up", "Scrape of target was successful", []string{"target"}, nil)
	scrapeDurationDesc = prometheus.NewDesc(prefix+"collector_duration_seconds", "Duration of a collector scrape for one target", []string{"target"}, nil)
	scrapeCollectorDurationDesc = prometheus.NewDesc(prefix+"collect_duration_seconds", "Duration of a scrape by collector and target", []string{"target", "collector"}, nil)

	cookies = make(map[string]*http.Cookie)
}

type zyxelCollector struct {
	device     string
	collectors map[string]collector.Collector
}

func newZyxelCollector(device string, client *http.Client) *zyxelCollector {

	return &zyxelCollector{
		device:     device,
		collectors: collectors(),
	}
}

func collectors() map[string]collector.Collector {
	m := make(map[string]collector.Collector)

	f := conf.Features

	if f.HWMon {
		m["hwmon"] = hwmon.NewCollector()
	}
	if f.POE {
		m["poe"] = poe.NewCollector()
	}

	return m
}

// Describe implements prometheus.Collector interface
func (c *zyxelCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- upDesc
	ch <- scrapeDurationDesc
	ch <- scrapeCollectorDurationDesc

	for _, col := range c.collectors {
		col.Describe(ch)
	}
}

// Collect implements prometheus.Collector interface
func (c *zyxelCollector) Collect(ch chan<- prometheus.Metric) {
	var success = true
	l := []string{c.device}
	t := time.Now()
	defer func() {
		ch <- prometheus.MustNewConstMetric(scrapeDurationDesc, prometheus.GaugeValue, time.Since(t).Seconds(), l...)
	}()

	d := conf.Devices[c.device]

	for k, col := range c.collectors {
		ct := time.Now()

		z, err := c.makeHTTPQuery(d, col.GetURI())
		if err == nil {
			err = col.Collect(z, ch, l)
		}

		if err != nil {
			success = false
			log.Errorln(k + ": " + err.Error())
		}
		ch <- prometheus.MustNewConstMetric(scrapeCollectorDurationDesc, prometheus.GaugeValue, time.Since(ct).Seconds(), append(l, k)...)
	}

	switch success {
	case true:
		ch <- prometheus.MustNewConstMetric(upDesc, prometheus.GaugeValue, 1, l...)
	case false:
		ch <- prometheus.MustNewConstMetric(upDesc, prometheus.GaugeValue, 0, l...)
	}
}

func (c *zyxelCollector) makeHTTPQuery(d config.Device, uri string) (*html.Tokenizer, error) {
	// query
	req, err := http.NewRequest("GET", d.Schema+"://"+d.Address+uri, nil)
	// check for cookie
	_, ok := cookies[c.device]
	if ok {
		req.AddCookie(cookies[c.device])
	} else {
		// fallback to bacic auth
		req.SetBasicAuth(d.User, d.Password)
		log.Info("Use basic auth")
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	for _, cookie := range resp.Cookies() {
		if cookie.Name == "C1" {
			if cookie.Value == "youshallnotpass" && ok {
				delete(cookies, c.device)
			} else {
				cookies[c.device] = cookie
				log.Info("Save cookie")
			}

		}
	}

	if resp.StatusCode/100 != 2 {
		err = fmt.Errorf("Wrong status code %d", resp.StatusCode)
		return nil, err
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	//log.Info(string(body))
	//log.Info(resp.StatusCode)

	// parse html
	z := html.NewTokenizer(bytes.NewReader(body))

	return z, nil
}
