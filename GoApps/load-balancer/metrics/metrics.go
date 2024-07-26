package metrics

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	EmptyQWeight = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "emptyqweight",
		Help: "Admission rate at last empty queue event.",
	}, []string{"service"})
)

func init() {
	// Unregister default collectors
	prometheus.Unregister(collectors.NewGoCollector())
	prometheus.Unregister(collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}))
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
	// Register any static metrics here if needed
	EmptyQWeight.WithLabelValues("service1").Set(0)
	EmptyQWeight.WithLabelValues("service2").Set(0)
	EmptyQWeight.WithLabelValues("service3").Set(0)
}

func UpdateMetric(service string, value float64) {
	EmptyQWeight.WithLabelValues(service).Set(value)
	log.Printf("Updated EmptyQWeight for %s to %f", service, value)
}

func FetchQdReqs() {
	metrics := make(map[string]float64)
	url := "http://consumer-metrics.rabbitmq-setup.svc.cluster.local:9095/metrics"
	prefix := "queued_requests"

	// Fetch and store metrics
	fetchAndStoreMetrics(url, prefix, metrics)

	// Print the metrics
	printMetrics(metrics, "📥 Queued Requests")
}

func FetchReplicas() {
	metrics := make(map[string]float64)

	// Prometheus query for autoscaler_actual_pods
	prometheusURL := "http://prometheus-kube-prometheus-prometheus.prometheus:9090"
	query := `sum(autoscaler_actual_pods{namespace_name="rabbitmq-setup", configuration_name=~"service.*"})`

	// Fetch and store metrics using Prometheus query
	fetchAndStorePrometheusQueryMetrics(prometheusURL, query, metrics)

	printMetrics(metrics, "🖇️ Number of Replicas")
}

// Function to fetch and store metrics
func fetchAndStoreMetrics(url string, prefix string, metrics map[string]float64) {
	resp, err := http.Get(url)
	if err != nil {
		log.Printf("Error querying metrics: %v", err)
		return
	}
	defer resp.Body.Close()

	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, prefix) {
			// Example format: prefix{service="service1"} 2
			parts := strings.Split(line, " ")
			if len(parts) == 2 {
				labelValuePair := strings.TrimPrefix(parts[0], prefix+"{")
				labelValuePair = strings.TrimSuffix(labelValuePair, "}")
				labelParts := strings.Split(labelValuePair, "=")
				if len(labelParts) == 2 {
					service := strings.Trim(labelParts[1], "\"")
					valueStr := parts[1]
					value, err := strconv.ParseFloat(valueStr, 64)
					if err != nil {
						log.Printf("Error parsing value for service %s: %v", service, err)
						continue
					}
					metrics[service] = value
				}
			}
		}
	}

	if err := scanner.Err(); err != nil {
		log.Printf("Error reading metrics response: %v", err)
	}
}

// Function to fetch and store metrics from a Prometheus query
func fetchAndStorePrometheusQueryMetrics(url string, query string, metrics map[string]float64) {
	queryURL := fmt.Sprintf("%s/api/v1/query?query=%s", url, query)
	resp, err := http.Get(queryURL)
	if err != nil {
		log.Printf("Error querying Prometheus: %v", err)
		return
	}
	defer resp.Body.Close()

	var response struct {
		Data struct {
			Result []struct {
				Metric struct {
					ConfigurationName string `json:"configuration_name"`
				} `json:"metric"`
				Value []interface{} `json:"value"`
			} `json:"result"`
		} `json:"data"`
	}

	err = json.NewDecoder(resp.Body).Decode(&response)
	if err != nil {
		log.Printf("Error decoding Prometheus response: %v", err)
		return
	}

	for _, result := range response.Data.Result {
		service := result.Metric.ConfigurationName
		valueStr := result.Value[1].(string)
		value, err := strconv.ParseFloat(valueStr, 64)
		if err != nil {
			log.Printf("Error parsing value for service %s: %v", service, err)
			continue
		}
		metrics[service] = value
	}
}

// Function to print stored metrics
func printMetrics(metrics map[string]float64, queryType string) {
	for service, value := range metrics {
		fmt.Printf("%s for %s: %f\n", queryType, service, value)
	}
}
