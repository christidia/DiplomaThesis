package metrics

import (
	"log"
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	AdmissionRateMetric = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "admission_rate",
		Help: "Current admission rate",
	})
)

func StartMetricsServer() {
	mux := http.NewServeMux()
	server := &http.Server{
		Addr:    ":9095",
		Handler: mux,
	}

	// Health check endpoint for readiness probe
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	mux.Handle("/metrics", promhttp.Handler())
	log.Printf("Prometheus metrics available at :9095/metrics")
	log.Fatal(server.ListenAndServe())
}

func UpdateMetric(value float64) {
	AdmissionRateMetric.Set(value)
	log.Printf("Updated admission rate metric to %f", value)
}
