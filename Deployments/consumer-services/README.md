# Consumer Services with Autoscaling

This directory contains YAML configuration files for deploying three consumer services (consumer-service-1, consumer-service-2, and consumer-service-3) in a Knative environment. These services are configured with two different autoscaling mechanisms: Knative Pod Autoscaler (KPA) and Kubernetes Horizontal Pod Autoscaler (HPA).

## Files
- **all-services-kpa.yaml**: Configures the services to use Knative Pod Autoscaler (KPA).
- **all-services-hpa.yaml**: Configures the services to use Kubernetes Horizontal Pod Autoscaler (HPA).


## Knative Pod Autoscaler (KPA)
The Knative Pod Autoscaler (KPA) scales services based on concurrency or request volume. KPA adjusts the number of replicas based on real-time traffic load. The key autoscaling metrics for KPA are:

- Concurrency: Controls scaling based on the number of concurrent requests.
- Annotations:

  - autoscaling.knative.dev/maxScale: Maximum number of replicas.
  - autoscaling.knative.dev/metric: Metric used for scaling (e.g., concurrency).

**Example usage**
```
autoscaling.knative.dev/maxScale: "3"
autoscaling.knative.dev/metric: "concurrency"
```

## Horizontal Pod Autoscaler (HPA)
The Kubernetes Horizontal Pod Autoscaler (HPA) scales services based on CPU or memory usage. HPA adjusts the number of replicas according to resource utilization metrics. The key autoscaling metrics for HPA are:

- CPU: Controls scaling based on CPU usage.
- Annotations:
    - autoscaling.knative.dev/class: Set to "hpa.autoscaling.knative.dev" for HPA.
    - autoscaling.knative.dev/metric: Metric used for scaling (e.g., cpu).
    - autoscaling.knative.dev/target: Target resource utilization percentage.

**Example usage**
```
autoscaling.knative.dev/class: "hpa.autoscaling.knative.dev"
autoscaling.knative.dev/metric: "cpu"
autoscaling.knative.dev/target: "65"
```
