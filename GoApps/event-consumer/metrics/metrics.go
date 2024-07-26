package metrics

import (
	"log"
	"net/http"

	"consumer/config"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	serviceName    string
	QueuedRequests = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "queued_requests",
		Help: "Number of requests weighting to be processed in internal queue.",
	}, []string{"service"})
)

func init() {
	// Unregister default collectors
	prometheus.Unregister(collectors.NewGoCollector())
	prometheus.Unregister(collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}))

	// QueuedRequests is already registered by promauto, no need to register it again
}

func StartMetricsServer() {
	mux := http.NewServeMux()
	server := &http.Server{
		Addr:    ":9095",
		Handler: mux,
	}
	mux.Handle("/metrics", promhttp.Handler())
	log.Printf("prometheus: listening on port %s", "9095")
	log.Fatal(server.ListenAndServe())
}

func InitMetrics() {
	serviceName = config.ServiceName
	QueuedRequests.WithLabelValues(serviceName).Set(0)
	log.Printf("Metrics for %s initialized", serviceName)

}

func UpdateMetric(value float64) {
	QueuedRequests.WithLabelValues(serviceName).Set(value)
	log.Printf("Updated QueuedRequests for %s to %f", serviceName, value)
}
