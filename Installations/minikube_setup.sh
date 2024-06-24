#!/bin/bash

# Start Minikube (uncomment if needed)
# minikube start -p knative --memory 14336 --cpus=16 --driver=docker

# Apply Kubernetes resources

# Knative-Serving component
kubectl apply -f https://github.com/knative/serving/releases/download/knative-v1.12.0/serving-crds.yaml
kubectl apply -f https://github.com/knative/serving/releases/download/knative-v1.12.0/serving-core.yaml

# Install a networking layer (istio)
kubectl apply -l knative.dev/crd-install=true -f https://github.com/knative/net-istio/releases/download/knative-v1.12.0/istio.yaml
kubectl apply -f https://github.com/knative/net-istio/releases/download/knative-v1.12.0/istio.yaml
kubectl apply -f https://github.com/knative/net-istio/releases/download/knative-v1.12.0/net-istio.yaml

# Install Knative Eventing
kubectl apply -f https://github.com/knative/eventing/releases/download/knative-v1.12.0/eventing-crds.yaml
kubectl apply -f https://github.com/knative/eventing/releases/download/knative-v1.12.0/eventing-core.yaml
kubectl apply -f https://github.com/knative/eventing/releases/download/knative-v1.12.0/in-memory-channel.yaml

# Eventing RabbitMQ Broker
kubectl apply -f https://github.com/knative-extensions/eventing-rabbitmq/releases/latest/download/rabbitmq-broker.yaml

# Eventing RabbitMQ Source
## Prerequisites
kubectl apply -f https://github.com/rabbitmq/cluster-operator/releases/latest/download/cluster-operator.yml
kubectl apply -f https://github.com/jetstack/cert-manager/releases/latest/download/cert-manager.yaml
kubectl apply -f https://github.com/rabbitmq/messaging-topology-operator/releases/latest/download/messaging-topology-operator-with-certmanager.yaml

kubectl apply -f https://github.com/knative-extensions/eventing-rabbitmq/releases/latest/download/rabbitmq-source.yaml

# Create Namespace if not exists
kubectl get namespace monitoring || kubectl create namespace monitoring

# Setting up Prometheus
helm repo add prometheus-community https://prometheus-community.github.io/helm-charts
helm repo update

# Check if Prometheus is already installed and upgrade if necessary
helm list -n monitoring | grep prometheus || helm install prometheus prometheus-community/kube-prometheus-stack -n monitoring -f values.yaml

# Service Monitors
kubectl apply -f https://raw.githubusercontent.com/knative-extensions/monitoring/main/servicemonitor.yaml
kubectl apply -f https://raw.githubusercontent.com/rabbitmq/cluster-operator/main/observability/prometheus/monitors/rabbitmq-servicemonitor.yml
kubectl apply -f https://raw.githubusercontent.com/rabbitmq/cluster-operator/main/observability/prometheus/monitors/rabbitmq-cluster-operator-podmonitor.yml

# Metrics API
kubectl apply -f https://github.com/kubernetes-sigs/metrics-server/releases/latest/download/components.yaml

# (Optional) Install HPA Knative Extension
# kubectl apply -f https://github.com/knative/serving/releases/download/knative-v1.12.2/serving-hpa.yaml

# Create Namespace rabbitmq-setup if not exists
kubectl get namespace rabbitmq-setup || kubectl create namespace rabbitmq-setup

# Deploy Redis Master and Replicas
helm repo add bitnami https://charts.bitnami.com/bitnami
helm repo update

# Check if Redis is already installed and upgrade if necessary
helm list -n rabbitmq-setup | grep my-redis || helm install my-redis bitnami/redis --namespace rabbitmq-setup

