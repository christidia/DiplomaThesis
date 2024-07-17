package metrics

import (
	"log"
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	// Define the Prometheus gauge metric
	EmptyQWeight = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "emptyqweight",
			Help: "Admission rate at last empty queue event.",
		},
		[]string{"service"},
	)
)

// StartMetricsServer starts the Prometheus metrics server using the custom registry
func StartMetricsServer() {
	CustomRegistry := prometheus.NewRegistry()
	// Set the default registry to the custom registry
	prometheus.DefaultRegisterer = CustomRegistry
	prometheus.DefaultGatherer = CustomRegistry

	prometheus.MustRegister(EmptyQWeight)

	log.Println("🚀Starting metrics server on :2112")
	http.Handle("/metrics", promhttp.HandlerFor(prometheus.DefaultGatherer, promhttp.HandlerOpts{EnableOpenMetrics: false}))
	log.Fatal(http.ListenAndServe(":2112", nil))
}
