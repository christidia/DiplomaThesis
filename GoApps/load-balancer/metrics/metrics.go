package metrics

import (
	"log"
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Define a custom Prometheus registry
var CustomRegistry = prometheus.NewRegistry()

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

func init() {
	// Register the custom metric with the custom registry
	CustomRegistry.MustRegister(EmptyQWeight)
}

// StartMetricsServer starts the Prometheus metrics server using the custom registry
func StartMetricsServer() {
	log.Println("🚀Starting metrics server on :2112")
	http.Handle("/metrics", promhttp.HandlerFor(CustomRegistry, promhttp.HandlerOpts{}))
	log.Fatal(http.ListenAndServe(":2112", nil))
}
