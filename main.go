package main

import (
	"bytes"
	"context"
	"io/ioutil"
	"net"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"fmt"
	"net/http"

	"gopkg.in/alecthomas/kingpin.v2"

	"github.com/R4scal/zyxel_gs1920_exporter/config"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/prometheus/common/log"
	"github.com/prometheus/common/version"
)

// single device can be defined via CLI flags, multiple via config file.
var (
	listenAddress = kingpin.Flag("web.listen-address", "Address on which to expose metrics and web interface.").Default(":9653").String()
	metricsPath   = kingpin.Flag("web.telemetry-path", "Path under which to expose Prometheus metrics.").Default("/metrics").String()
	configFile    = kingpin.Flag("config-file", "config file to load").Default("").String()
	timeoutOffset = kingpin.Flag("timeout-offset", "Offset to subtract from timeout in seconds.").Default("0.5").Float64()
	insecure      = kingpin.Flag("insecure", "skips verification of server certificate when using TLS (not recommended)").Default("false").Bool()

	conf       *config.Config
	httpClient *http.Client

	appVersion = "DEVELOPMENT"
	shortSha   = "0xDEADBEEF"
)

func loadConfig() (*config.Config, error) {
	b, err := ioutil.ReadFile(*configFile)
	if err != nil {
		return nil, err
	}

	return config.Load(bytes.NewReader(b))
}

func init() {
	prometheus.MustRegister(version.NewCollector("gs1920_exporter"))
}

func main() {
	os.Exit(run())
}

func run() int {
	var err error
	kingpin.Version(version.Print("gs1920_exporter"))
	kingpin.HelpFlag.Short('h')
	kingpin.Parse()

	log.Info("msg", "Starting gs1920_exporter", "version", version.Info())
	log.Info("msg", "Build context", version.BuildContext())

	conf, err = loadConfig()
	if err != nil {
		log.Fatal("msg", "Could not load config", "err", err)
	}
	log.Info("msg", "Loaded config file")

	// HTTP Client
	netTransport := &http.Transport{
		Dial: (&net.Dialer{
			Timeout: 30 * time.Second,
		}).Dial,
		MaxIdleConnsPerHost: 1,
		TLSHandshakeTimeout: 10 * time.Second,
	}
	httpClient = &http.Client{
		Timeout:   time.Second * 30,
		Transport: netTransport,
	}

	http.Handle(*metricsPath, promhttp.Handler())
	http.HandleFunc("/probe", func(w http.ResponseWriter, r *http.Request) {
		probeHandler(w, r)
	})
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(`<html>
    <head><title>GS1920 Exporter</title></head>
    <body>
    <h1>GS1920 Exporter</h1>
    <p><a href="metrics">Metrics</a></p>
	</body>
    </html>`))
	})

	srv := http.Server{Addr: *listenAddress}
	srvc := make(chan struct{})
	term := make(chan os.Signal, 1)
	signal.Notify(term, os.Interrupt, syscall.SIGTERM)

	go func() {
		log.Info("msg", "Listening on address", "address", *listenAddress)
		if err := srv.ListenAndServe(); err != http.ErrServerClosed {
			log.Error("msg", "Error starting HTTP server", "err", err)
			close(srvc)
		}
	}()

	for {
		select {
		case <-term:
			log.Info("msg", "Received SIGTERM, exiting gracefully...")
			return 0
		case <-srvc:
			return 1
		}
	}

}

func probeHandler(w http.ResponseWriter, r *http.Request) {
	timeoutSeconds, err := getTimeout(r, *timeoutOffset)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to parse timeout from Prometheus header: %s", err), http.StatusInternalServerError)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), time.Duration(timeoutSeconds*float64(time.Second)))
	defer cancel()
	r = r.WithContext(ctx)

	params := r.URL.Query()
	target := params.Get("target")
	if target == "" {
		http.Error(w, "Target parameter is missing", http.StatusBadRequest)
		return
	}
	_, ok := conf.Devices[target]
	if !ok {
		http.Error(w, "Target device not found in config", http.StatusBadRequest)
		return
	}

	registry := prometheus.NewRegistry()
	collector := newZyxelCollector(target, httpClient)
	registry.MustRegister(collector)

	promhttp.HandlerFor(registry, promhttp.HandlerOpts{
		ErrorLog:      log.NewErrorLogger(),
		ErrorHandling: promhttp.ContinueOnError}).ServeHTTP(w, r)
}

func getTimeout(r *http.Request, offset float64) (timeoutSeconds float64, err error) {
	// If a timeout is configured via the Prometheus header, add it to the request.
	if v := r.Header.Get("X-Prometheus-Scrape-Timeout-Seconds"); v != "" {
		var err error
		timeoutSeconds, err = strconv.ParseFloat(v, 64)
		if err != nil {
			return 0, err
		}
	}
	if timeoutSeconds == 0 {
		timeoutSeconds = 120
	}

	var maxTimeoutSeconds = timeoutSeconds - offset
	timeoutSeconds = maxTimeoutSeconds

	return timeoutSeconds, nil
}
