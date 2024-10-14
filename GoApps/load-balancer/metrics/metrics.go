package metrics

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"math"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	v1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/prometheus/common/model"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"

	"load-balancer/db"

	"github.com/prometheus/client_golang/api"
)

var (
	services    = []string{"service1", "service2", "service3"}
	GammaMetric = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "emptyqweight",
		Help: "Gamma Metric for each service, calculated and updated every t_k event.",
	}, []string{"service"})
)

func init() {
	// Unregister default collectors
	prometheus.Unregister(collectors.NewGoCollector())
	prometheus.Unregister(collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}))
}

// Helper function to convert internal service names to external service names
func externalServiceName(service string) string {
	return strings.Replace(service, "service", "consumer-service-", 1)
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
	GammaMetric.WithLabelValues("service1").Set(0)
	GammaMetric.WithLabelValues("service2").Set(0)
	GammaMetric.WithLabelValues("service3").Set(0)
}

func UpdateMetric(service string, value float64) {
	GammaMetric.WithLabelValues(service).Set(value)
	log.Printf("Updated GammaMetric for %s to %f", service, value)
}

func FetchQdReqs() map[string]int {
	metricType := "queued_requests"
	metrics := make(map[string]int)

	// Fetch and store metrics
	fetchAndStoreMetrics(services, metricType, metrics)
	return metrics
}

func FetchReplicas(service string) int {
	externalName := externalServiceName(service)
	replicas := FetchReplicaNum(externalName)
	log.Printf("🖇️ Number of Replicas for %s (%s): %d", service, externalName, replicas)
	return replicas
}

// Function to fetch and store metrics for all services
func fetchAndStoreMetrics(services []string, metricType string, metrics map[string]int) {
	for _, service := range services {
		endpoints, err := fetchServiceEndpoints(service)
		if err != nil {
			log.Printf("Error fetching endpoints for %s: %v", service, err)
			continue
		}

		totalQueuedRequests := 0

		for _, endpoint := range endpoints {
			metricURL := fmt.Sprintf("http://%s:9095/metrics", endpoint)
			resp, err := http.Get(metricURL)
			if err != nil {
				log.Printf("Error querying metrics for %s: %v", endpoint, err)
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
						value, err := strconv.Atoi(valueStr)
						if err != nil {
							log.Printf("Error parsing metric value for %s: %v", endpoint, err)
						} else {
							totalQueuedRequests += value
						}
					}
				}
			}

			if err := scanner.Err(); err != nil {
				log.Printf("Error reading metrics response for %s: %v", endpoint, err)
			}

			resp.Body.Close()
		}

		metrics[service] = totalQueuedRequests
		log.Printf("📥 Total Queued Requests for %s: %d", service, totalQueuedRequests)
	}
}

// Function to fetch number of replicas of service
func FetchReplicaNum(service string) int {
	endpoints, err := fetchServiceEndpoints(service)
	if err != nil {
		log.Printf("Error fetching endpoints for %s: %v", service, err)
		return 0
	}
	return len(endpoints)
}

// Function to fetch service endpoints
func fetchServiceEndpoints(service string) ([]string, error) {
	externalName := externalServiceName(service)
	config, err := rest.InClusterConfig()
	if err != nil {
		kubeconfig := filepath.Join(homeDir(), ".kube", "config")
		config, err = clientcmd.BuildConfigFromFlags("", kubeconfig)
		if err != nil {
			return nil, err
		}
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	pods, err := clientset.CoreV1().Pods("rabbitmq-setup").List(context.TODO(), metav1.ListOptions{
		LabelSelector: fmt.Sprintf("service=%s,app=event-display", externalName),
	})
	if err != nil {
		return nil, err
	}

	var endpoints []string
	for _, pod := range pods.Items {
		endpoints = append(endpoints, pod.Status.PodIP)
	}

	return endpoints, nil
}

func homeDir() string {
	if h := os.Getenv("HOME"); h != "" {
		return h
	}
	return os.Getenv("USERPROFILE") // windows
}

// Function to calculate gamma for each service
func UpdateGamma() {
	queued_requests := FetchQdReqs()

	// Calculate gamma for each service
	for _, service := range db.ServicesMap {
		qWeight := db.EmptyQWeights[service.Name]
		queuedRequests := queued_requests[service.Name]
		replicas := float64(FetchReplicaNum(service.Name))

		gamma := (qWeight*service.Beta + math.Sqrt(replicas*float64(queuedRequests)*2*float64(service.Alpha)))
		UpdateMetric(service.Name, float64(gamma))
		log.Printf("🔢 Gamma for %s: %f", service.Name, gamma)
	}
}

// Fetch CPU usage using Prometheus metrics for all replicas of a service
func FetchCPUUsage(service string) float64 {
	query := fmt.Sprintf(`sum(rate(container_cpu_usage_seconds_total{namespace="$namespace", pod=~"%s-deployment-.*", container=""}[1m]))`, service)
	return executePrometheusQuery(query)
}

// Fetch CPU request limits for all replicas of a service
func FetchCPULimit(service string) float64 {
	query := fmt.Sprintf(`sum(kube_pod_container_resource_requests{resource="cpu", namespace="$namespace", pod=~"%s-deployment-.*"})`, service)
	return executePrometheusQuery(query)
}

// Function to execute Prometheus query and return the result
func executePrometheusQuery(query string) float64 {
	// Set up Prometheus client
	client, err := api.NewClient(api.Config{
		Address: "http://prometheus-kube-prometheus-prometheus.monitoring.svc.cluster.local:9090",
	})
	if err != nil {
		log.Printf("Error creating Prometheus client: %v", err)
		return 0
	}

	v1api := v1.NewAPI(client)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Execute query
	result, warnings, err := v1api.Query(ctx, query, time.Now())
	if err != nil {
		log.Printf("Error executing Prometheus query: %v", err)
		return 0
	}

	// Handle warnings if any
	if len(warnings) > 0 {
		log.Printf("Warnings from Prometheus: %v", warnings)
	}

	// Process result
	if result.Type() == model.ValVector {
		vector := result.(model.Vector)
		if len(vector) > 0 {
			return float64(vector[0].Value)
		}
	}
	return 0
}
