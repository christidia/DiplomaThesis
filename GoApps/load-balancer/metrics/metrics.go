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

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"

	"load-balancer/db"
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

func FetchQdReqs() map[string]int {
	metricType := "queued_requests"
	metrics := make(map[string]int)

	// Fetch and store metrics
	fetchAndStoreMetrics(services, metricType, metrics)
	return metrics
}

func FetchReplicas(service string) int {
	replicas := FetchReplicaNum(service)
	log.Printf("🖇️ Number of Replicas for %s: %d", service, replicas)
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
							//log.Printf("📥 Queued Requests for %s: %d", endpoint, value)
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
	config, err := rest.InClusterConfig()
	if err != nil {
		kubeconfig := filepath.Join(
			homeDir(), ".kube", "config",
		)
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
		LabelSelector: fmt.Sprintf("service=%s,app=event-display", service),
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
func CalculateGamma() {
	queued_requests := FetchQdReqs()

	// Calculate gamma for each service
	for _, service := range db.ServicesMap {
		qWeight := db.EmptyQWeights[service.Name]
		queuedRequests := queued_requests[service.Name]
		replicas := float64(FetchReplicaNum(service.Name))

		gamma := (qWeight*service.Beta + math.Sqrt(replicas*float64(queuedRequests)*2*float64(service.Alpha)))
		log.Printf("🔢 Gamma for %s: %f", service.Name, gamma)
	}
}

// // Function to fetch and store metrics for all services
// func fetchAndStoreMetrics(services []string, metricType string, metrics map[string]int) {
// 	for _, service := range services {
// 		url := fmt.Sprintf("http://%s-metrics.rabbitmq-setup.svc.cluster.local:9095/metrics", service)

// 		resp, err := http.Get(url)
// 		if err != nil {
// 			log.Printf("Error querying metrics for %s: %v", service, err)
// 			continue
// 		}

// 		scanner := bufio.NewScanner(resp.Body)
// 		for scanner.Scan() {
// 			line := scanner.Text()
// 			if strings.HasPrefix(line, metricType) {
// 				// Parse the metric value
// 				parts := strings.Fields(line)
// 				if len(parts) >= 2 {
// 					valueStr := parts[len(parts)-1]
// 					value, err := strconv.Atoi(valueStr)
// 					if err != nil {
// 						log.Printf("Error parsing metric value for %s: %v", service, err)
// 					} else {
// 						metrics[service] = value
// 						log.Printf("📥 Queued Requests for %s: %d", service, value)
// 					}
// 				}
// 			}
// 		}

// 		if err := scanner.Err(); err != nil {
// 			log.Printf("Error reading metrics response for %s: %v", service, err)
// 		}

// 		resp.Body.Close()
// 	}
// }

// // Function to query Prometheus and extract numerical value
// func queryAndExtract(queryURL string) (string, error) {
// 	client := &http.Client{}
// 	resp, err := client.Get(queryURL)
// 	if err != nil {
// 		return "", fmt.Errorf("error querying Prometheus: %v", err)
// 	}
// 	defer resp.Body.Close()

// 	body, err := ioutil.ReadAll(resp.Body)
// 	if err != nil {
// 		return "", fmt.Errorf("error reading response body: %v", err)
// 	}

// 	var response struct {
// 		Data struct {
// 			Result []struct {
// 				Value []interface{} `json:"value"`
// 			} `json:"result"`
// 		} `json:"data"`
// 	}

// 	if err := json.Unmarshal(body, &response); err != nil {
// 		return "", fmt.Errorf("error parsing JSON response: %v", err)
// 	}

// 	if len(response.Data.Result) > 0 {
// 		value := response.Data.Result[0].Value[1].(string) // Assuming the value is in the second position of the array
// 		return value, nil
// 	}

// 	return "", fmt.Errorf("no result found in Prometheus response")
// }

// // Function to fetch and store Prometheus metrics for all services
// func fetchAndStorePrometheusMetrics(metrics map[string]int) {
// 	for _, serviceName := range services {
// 		promQuery := fmt.Sprintf("autoscaler_actual_pods{namespace_name=\"rabbitmq-setup\", configuration_name=\"%s\"}", serviceName)
// 		url := fmt.Sprintf("http://prometheus-kube-prometheus-prometheus.monitoring:9090/api/v1/query?query=%s", url.QueryEscape(promQuery))

// 		value, err := queryAndExtract(url)
// 		if err != nil {
// 			log.Printf("Error querying Prometheus for service %s: %v", serviceName, err)
// 			continue
// 		}

// 		valInt, err := strconv.Atoi(value)
// 		if err != nil {
// 			log.Printf("Error converting metric value for %s: %v", serviceName, err)
// 			continue
// 		}
// 		metrics[serviceName] = valInt
// 		log.Printf("🖇️ Number of Replicas for %s: %d", serviceName, valInt)
// 	}
// }
