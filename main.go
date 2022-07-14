package main

import (
	"net/http"
	"os"
	"context"
	"strings"
	"sync"
	"time"

	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/prometheus/exporter-toolkit/web"
	kingpin "gopkg.in/alecthomas/kingpin.v2"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/v2fly/v2ray-core/v4/app/stats/command"
	"google.golang.org/grpc"
)

var (
	V2RayEndpoint = kingpin.Flag("v2ray.endpoint","V2Ray API endpoint").Default("127.0.0.1:10085").String()
	listenAddress = kingpin.Flag("web.listen-address","Address on which to expose metrics and web interface.").Default(":9110").String()
	metricsPath = kingpin.Flag("web.telemetry-path","Path under which to expose metrics.").Default("/metrics").String()
	configFile = kingpin.Flag("web.config","Path to config yaml file that can enable TLS or authentication.").Default("").String()
)

type Exporter struct {
	sync.Mutex
	conn *grpc.ClientConn
	scrapeCounter prometheus.Counter
	metricDescriptions map[string]*prometheus.Desc
}

func main() {
	kingpin.Parse()
	logger := log.NewLogfmtLogger(os.Stdout)
	registry := prometheus.NewRegistry()

	// prometheus
	newexporter := Exporter{}
	newexporter.scrapeCounter = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "v2ray_scrape_counter",
		Help: "Number of scrapes performed",
	})

	newexporter.metricDescriptions = map[string]*prometheus.Desc{}

	for k, desc := range map[string]struct {
		txt  string
		lbls []string
	}{
		"v2ray_traffic_uplink_bytes_total":   {txt: "Number of transmitted bytes", lbls: []string{"dimension", "target"}},
		"v2ray_traffic_downlink_bytes_total": {txt: "Number of received bytes", lbls: []string{"dimension", "target"}},
	}{
		newexporter.metricDescriptions[k] = prometheus.NewDesc(k, desc.txt, desc.lbls, nil)
	}

	registry.MustRegister(&newexporter)

	// v2ray api
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	conn, err := grpc.DialContext(ctx, *V2RayEndpoint, grpc.WithInsecure(), grpc.WithBlock())
	if err != nil {
		level.Error(logger).Log("timeout", err)
		os.Exit(1)
	}

	newexporter.conn = conn
	defer newexporter.conn.Close()

	// http(s) handler
	http.HandleFunc(*metricsPath, func(w http.ResponseWriter, r *http.Request) {
		promhttp.HandlerFor(registry, promhttp.HandlerOpts{ErrorHandling: promhttp.ContinueOnError},).ServeHTTP(w, r)
	})
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`<html>
			<head><title>V2Ray Exporter</title></head>
			<body>
			<h1>V2Ray Exporter</h1>
			<p><a href="` + *metricsPath + `">Metrics</a></p>
			</body>
			</html>`))
	})

	server := &http.Server{Addr: *listenAddress}
	if err := web.ListenAndServe(server, *configFile, logger); err != nil {
		level.Error(logger).Log("err", err)
		os.Exit(1)
	}
}

func (e *Exporter) Collect(ch chan<- prometheus.Metric) {
	e.Lock()
	defer e.Unlock()

	e.scrapeCounter.Inc()
	e.scrapeV2Ray(ch)
	ch <- e.scrapeCounter
}

func (e *Exporter) Describe(ch chan<- *prometheus.Desc) {
	for _, desc := range e.metricDescriptions {
		ch <- desc
	}
	ch <- e.scrapeCounter.Desc()
}

func (e *Exporter) scrapeV2Ray(ch chan<- prometheus.Metric) error {
	client := command.NewStatsServiceClient(e.conn)
	resp, _ := client.QueryStats(context.Background(), &command.QueryStatsRequest{Reset_: false})

	for _, s := range resp.GetStat() {
		// example value: inbound>>>socks-proxy>>>traffic>>>uplink
		p := strings.Split(s.GetName(), ">>>")
		metric := "v2ray" + "_" + p[2] + "_" + p[3] + "_bytes_total"
		dimension := p[0]
		target := p[1]
		val := float64(s.GetValue())

		descr := e.metricDescriptions[metric]
		m, _ := prometheus.NewConstMetric(descr, prometheus.CounterValue, val, dimension, target)
		ch <- m
	}

	return nil
}
