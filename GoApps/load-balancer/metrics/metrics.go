package metrics

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	services     = []string{"service1", "service2", "service3"}
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
	metricType := "queued_requests"
	metrics := make(map[string]float64)

	// Fetch and store metrics
	fetchAndStoreMetrics(services, metricType, metrics)

	// Print the metrics
	printMetrics(metrics, "📥 Queued Requests")
}

func FetchReplicas() {
	metrics := make(map[string]float64)

	// Prometheus query for autoscaler_actual_pods
	query := `sum(autoscaler_actual_pods{namespace_name="rabbitmq-setup", configuration_name=~"service.*"})`

	// Fetch and store metrics using Prometheus query
	fetchAndStorePrometheusMetrics(query, metrics)

	printMetrics(metrics, "🖇️ Number of Replicas")
}

// Function to fetch and store metrics for all services
func fetchAndStoreMetrics(services []string, metricType string, metrics map[string]float64) {
	for _, service := range services {
		url := fmt.Sprintf("http://%s-metrics.rabbitmq-setup.svc.cluster.local:9095/metrics", service)

		resp, err := http.Get(url)
		if err != nil {
			log.Printf("Error querying metrics for %s: %v", service, err)
			continue
		}

		scanner := bufio.NewScanner(resp.Body)
		for scanner.Scan() {
			line := scanner.Text()
			if strings.HasPrefix(line, metricType) {
				// Parse the metric value
				parts := strings.Fields(line)
				if len(parts) >= 2 {
					valueStr := parts[len(parts)-1]
					value, err := strconv.ParseFloat(valueStr, 64)
					if err != nil {
						log.Printf("Error parsing metric value for %s: %v", service, err)
					} else {
						metrics[service] = value
						log.Printf("📥 Queued Requests for %s: %f", service, value)
					}
				}
			}
		}

		if err := scanner.Err(); err != nil {
			log.Printf("Error reading metrics response for %s: %v", service, err)
		}

		resp.Body.Close()
	}
}

// Function to query Prometheus and extract numerical value
func queryAndExtract(queryURL string) (string, error) {
	client := &http.Client{}
	resp, err := client.Get(queryURL)
	if err != nil {
		return "", fmt.Errorf("error querying Prometheus: %v", err)
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("error reading response body: %v", err)
	}

	var response struct {
		Data struct {
			Result []struct {
				Value []interface{} `json:"value"`
			} `json:"result"`
		} `json:"data"`
	}

	if err := json.Unmarshal(body, &response); err != nil {
		return "", fmt.Errorf("error parsing JSON response: %v", err)
	}

	if len(response.Data.Result) > 0 {
		value := response.Data.Result[0].Value[1].(string) // Assuming the value is in the second position of the array
		return value, nil
	}

	return "", fmt.Errorf("no result found in Prometheus response")
}

// Function to fetch and store Prometheus metrics for all services
func fetchAndStorePrometheusMetrics(query string, metrics map[string]float64) {
	for _, serviceName := range services {
		promQuery := fmt.Sprintf("autoscaler_actual_pods{namespace_name=\"rabbitmq-setup\", configuration_name=\"%s\"}", serviceName)
		url := fmt.Sprintf("http://prometheus-kube-prometheus-prometheus.prometheus:9090/api/v1/query?query=%s", url.QueryEscape(promQuery))

		value, err := queryAndExtract(url)
		if err != nil {
			log.Printf("Error querying Prometheus for service %s: %v", serviceName, err)
			continue
		}

		valFloat, err := strconv.ParseFloat(value, 64)
		if err != nil {
			log.Printf("Error converting metric value for %s: %v", serviceName, err)
			continue
		}
		metrics[serviceName] = valFloat
		log.Printf("🖇️ Number of Replicas for %s: %f", serviceName, valFloat)
	}
}

// Function to print stored metrics
func printMetrics(metrics map[string]float64, queryType string) {
	for service, value := range metrics {
		fmt.Printf("%s for %s: %f\n", queryType, service, value)
	}
}
