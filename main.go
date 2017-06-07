package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	_ "net/http/pprof"
	"os"
	"strconv"
	"strings"
	"time"

	hpe "github.com/appscode/haproxy_exporter/exporter"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/common/log"
	"github.com/prometheus/common/version"
)

func main() {
	const pidFileHelpText = `Path to HAProxy pid file.

	If provided, the standard process metrics get exported for the HAProxy
	process, prefixed with 'haproxy_process_...'. The haproxy_process exporter
	needs to have read access to files owned by the HAProxy process. Depends on
	the availability of /proc.

	https://prometheus.io/docs/instrumenting/writing_clientlibs/#process-metrics.`

	var (
		listenAddress             = flag.String("web.listen-address", ":9101", "Address to listen on for web interface and telemetry.")
		metricsPath               = flag.String("web.telemetry-path", "/metrics", "Path under which to expose metrics.")
		haProxyScrapeURI          = flag.String("haproxy.scrape-uri", "http://localhost/;csv", "URI on which to scrape HAProxy.")
		haProxyServerMetricFields = flag.String("haproxy.server-metric-fields", hpe.ServerMetrics.String(), "Comma-separated list of exported server metrics. See http://cbonte.github.io/haproxy-dconv/configuration-1.5.html#9.1")
		haProxyTimeout            = flag.Duration("haproxy.timeout", 5*time.Second, "Timeout for trying to get stats from HAProxy.")
		haProxyPidFile            = flag.String("haproxy.pid-file", "", pidFileHelpText)
		showVersion               = flag.Bool("version", false, "Print version information.")
	)
	flag.Parse()

	if *showVersion {
		fmt.Fprintln(os.Stdout, version.Print("haproxy_exporter"))
		os.Exit(0)
	}

	selectedServerMetrics, err := hpe.FilterServerMetrics(*haProxyServerMetricFields)
	if err != nil {
		log.Fatal(err)
	}

	log.Infoln("Starting haproxy_exporter", version.Info())
	log.Infoln("Build context", version.BuildContext())

	exporter, err := hpe.NewExporter(*haProxyScrapeURI, selectedServerMetrics, *haProxyTimeout)
	if err != nil {
		log.Fatal(err)
	}
	prometheus.MustRegister(exporter)
	prometheus.MustRegister(version.NewCollector("haproxy_exporter"))

	if *haProxyPidFile != "" {
		procExporter := prometheus.NewProcessCollectorPIDFn(
			func() (int, error) {
				content, err := ioutil.ReadFile(*haProxyPidFile)
				if err != nil {
					return 0, fmt.Errorf("Can't read pid file: %s", err)
				}
				value, err := strconv.Atoi(strings.TrimSpace(string(content)))
				if err != nil {
					return 0, fmt.Errorf("Can't parse pid file: %s", err)
				}
				return value, nil
			}, hpe.Namespace)
		prometheus.MustRegister(procExporter)
	}

	log.Infoln("Listening on", *listenAddress)
	http.Handle(*metricsPath, prometheus.Handler())
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`<html>
             <head><title>Haproxy Exporter</title></head>
             <body>
             <h1>Haproxy Exporter</h1>
             <p><a href='` + *metricsPath + `'>Metrics</a></p>
             </body>
             </html>`))
	})
	log.Fatal(http.ListenAndServe(*listenAddress, nil))
}
