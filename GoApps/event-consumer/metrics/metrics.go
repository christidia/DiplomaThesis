package metrics

import (
	"log"
	"net/http"

	"consumer/config"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	serviceName    = config.ServiceName
	QueuedRequests = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "queued_requests",
		Help: "Number of requests waiting to be processed in internal queue.",
	})
)

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
	// Register any static metrics here if needed
	QueuedRequests.WithLabelValues("serviceName").Set(0)
}

func UpdateMetric(value float64) {
	QueuedRequests.WithLabelValues(serviceName).Set(value)
	log.Printf("Updated QueuedRequests for %s to %f", serviceName, value)

}
