package metrics

import (
	"bufio"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

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

// Function to fetch and print metrics every 5 seconds
func FetchAndPrintMetrics() {
	url := "http://consumer-metrics.rabbitmq-setup.svc.cluster.local:9095/metrics"

	for {
		resp, err := http.Get(url)
		if err != nil {
			log.Printf("Error querying metrics: %v", err)
			time.Sleep(5 * time.Second)
			continue
		}

		scanner := bufio.NewScanner(resp.Body)
		for scanner.Scan() {
			line := scanner.Text()
			if strings.HasPrefix(line, "queued_requests") {
				fmt.Println(line)
			}
		}

		if err := scanner.Err(); err != nil {
			log.Printf("Error reading metrics response: %v", err)
		}

		resp.Body.Close()
		time.Sleep(5 * time.Second)
	}
}
