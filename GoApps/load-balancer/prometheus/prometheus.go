package prom

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"time"

	rdb "load-balancer/redis"
)

// Function to extract service names into a slice
func GetServiceNames(services rdb.ServicesMap) []string {
	var names []string
	for _, service := range services {
		names = append(names, service.Name)
	}
	return names
}

func queryPrometheus() {
	// Define service names
	serviceNames := GetServiceNames(rdb.ServicesMap)

	// Create an HTTP client
	client := http.Client{
		Timeout: 5 * time.Second,
	}

	// Function to query Prometheus and extract numerical value
	queryAndExtract := func(queryURL string) (string, error) {
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
			value := response.Data.Result[0].Value[1].(string) // the value is in the second position of the array
			return value, nil
		}

		return "", fmt.Errorf("no result found in Prometheus response")
	}

	// Query and extract numerical values for each service
	for _, serviceName := range serviceNames {
		query := fmt.Sprintf("autoscaler_actual_pods{namespace_name=\"rabbitmq-setup\", configuration_name=\"%s\"}", serviceName)
		url := fmt.Sprintf("http://prometheus-kube-prometheus-prometheus.prometheus:9090/api/v1/query?query=%s", url.QueryEscape(query))

		value, err := queryAndExtract(url)
		if err != nil {
			log.Printf("Error querying Prometheus for service %s: %v", serviceName, err)
			continue
		}

		log.Printf("Prometheus Response for service %s (autoscaler_actual_pods): %s\n", serviceName, value)
	}

}
