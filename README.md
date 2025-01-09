# Custom Load Balancer in Knative Environment

This repository contains the complete implementation for the system described in the diploma thesis titled **"Custom Load Balancer in a Serverless Distributed System using Knative"**. The project explores dynamic process scheduling, admission control, and load balancing within a Knative-based environment. The primary focus is the development and evaluation of a custom load balancer that utilizes the Additive Increase Multiplicative Decrease (AIMD) algorithm, integrated with Knative, Redis, and RabbitMQ to handle dynamic traffic patterns efficiently.

## Project Structure

### 1. Consumer Services
The repository contains the implementation of multiple consumer services that process incoming events routed by the load balancer. These services are managed as Knative services and scale based on traffic.

### 2. Custom Load Balancer
The core of the project, the custom load balancer, dynamically adjusts admission rates using the AIMD algorithm. The balancer distributes incoming requests across consumer services based on their performance and resource utilization.

### 3. Admission Controllers
Admission controllers are responsible for controlling the flow of requests to the consumer services. They dynamically update their configurations based on real-time traffic metrics and system state.

### 4. YAML Configurations
All necessary Kubernetes YAML files for deploying the Knative components, RabbitMQ broker, Redis, and other system components are provided. These include:
- Knative service and deployment configurations.
- RabbitMQ broker and trigger configurations.
- Redis deployment for caching and state management.

### 5. Installation Scripts
Installation scripts are included for setting up the environment. These scripts help automate the deployment process for the various components (Minikube setup, Knative, RabbitMQ, Redis, and Prometheus/Grafana for monitoring).

## System Architecture

The system is built around a Knative environment using RabbitMQ as a messaging intermediary. The custom load balancer dynamically routes events based on traffic patterns and admission rates, optimizing resource usage and ensuring efficient scaling.

### Key Components:
- **Knative Services**: Consumer services that handle dynamic workloads.
- **RabbitMQ Broker**: Manages message queues and routes events to the appropriate consumers.
- **Redis**: Used for real-time state management and metrics caching.
- **Prometheus/Grafana**: Provides system metrics and visualization for monitoring the performance of the system.

