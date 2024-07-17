package metrics

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	// Define a custom Prometheus registry
	customRegistry = prometheus.NewRegistry()

	// Define the Prometheus gauge metric
	EmptyQWeight = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "emptyqweight",
			Help: "Admission rate at last empty queue event.",
		},
		[]string{"service"},
	)
)

func init() {
	// Register the custom metric with the custom registry
	customRegistry.MustRegister(EmptyQWeight)
}

// StartMetricsServer starts the Prometheus metrics server using the custom registry
func StartMetricsServer() {
	http.Handle("/metrics", promhttp.HandlerFor(customRegistry, promhttp.HandlerOpts{}))
	http.ListenAndServe(":2112", nil)
}
