package metrics

import (
	"log"
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	// Define the Prometheus metric
	EmptyQWeight = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "emptyqweight",
			Help: "admission rate at last empty queue event.",
		},
		[]string{"service"},
	)
)

func init() {
	// Register the metric with Prometheus
	prometheus.MustRegister(EmptyQWeight)
}

// StartMetricsServer starts the Prometheus metrics server
func StartMetricsServer() {
	log.Println("🚀 Metrics server is running on port 2112")
	http.Handle("/metrics", promhttp.Handler())
	http.ListenAndServe(":2112", nil)
}
