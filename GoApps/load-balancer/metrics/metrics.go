package metrics

import (
	"log"
	"math/rand"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
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
	// Unregister the default Go and process collectors
	CustomRegistry.Unregister(collectors.NewGoCollector())
	CustomRegistry.Unregister(collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}))

	// Register the custom metric with the custom registry
	CustomRegistry.MustRegister(EmptyQWeight)
}

// StartMetricsServer starts the Prometheus metrics server using the custom registry
func StartMetricsServer() {
	log.Println("🚀 Starting metrics server on :2112")
	http.Handle("/metrics", promhttp.HandlerFor(CustomRegistry, promhttp.HandlerOpts{EnableOpenMetrics: false}))
	log.Fatal(http.ListenAndServe(":2112", nil))
}

// UpdateMetrics updates the EmptyQWeight metric periodically
func UpdateMetrics() {
	for {
		// Simulate updating the metric with random values
		serviceNames := []string{"service1", "service2", "service3"}
		for _, service := range serviceNames {
			newValue := rand.Float64() * 100
			EmptyQWeight.WithLabelValues(service).Set(newValue)
			log.Printf("Updated EmptyQWeight for %s to %f", service, newValue)
		}

		// Sleep for 10 seconds before updating again
		time.Sleep(10 * time.Second)
	}
}
