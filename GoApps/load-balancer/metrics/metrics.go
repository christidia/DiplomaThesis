package metrics

import (
	"log"
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	EmptyQWeight = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "emptyqweight",
		Help: "Admission rate at last empty queue event.",
	}, []string{"service"})
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
	EmptyQWeight.WithLabelValues("service1").Set(0)
	EmptyQWeight.WithLabelValues("service2").Set(0)
	EmptyQWeight.WithLabelValues("service3").Set(0)
}

func UpdateMetric(service string, value float64) {
	EmptyQWeight.WithLabelValues(service).Set(value)
	log.Printf("Updated EmptyQWeight for %s to %f", service, value)
}
