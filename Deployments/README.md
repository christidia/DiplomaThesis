# Custom Load Balancer Deployment

This document provides instructions for configuring the environment variables required for deploying the custom load balancer in a Knative environment.

## Key Environment Variables

The following environment variables are crucial for the load balancer's operation. These must be set correctly to ensure proper integration with RabbitMQ, Redis, and the services.

### 1. **Redis Configuration**
- **`REDIS_URL`**:  
  Specifies the URL of the Redis master service.  
  Example:  
  ```yaml
  REDIS_URL: "my-redis-master.rabbitmq-setup.svc.cluster.local:6379"

- **`REDIS_PASSWORD`**:  
The password for Redis authentication, fetched from a Kubernetes secret.
Example:
```
REDIS_PASSWORD: 
  valueFrom:
    secretKeyRef:
      name: my-redis
      key: redis-password
```

### 2.  RabbitMQ Configuration 

- **`RABBITMQ_URL`**: 
The URL for RabbitMQ management API for accessing queues.
Example:

```
RABBITMQ_URL: "http://rabbitmq.rabbitmq-setup.svc.cluster.local:15672/api/queues"
```

- **`RABBITMQ_USERNAME and RABBITMQ_PASSWORD`**: 
RabbitMQ credentials fetched from Kubernetes secrets.

Example:

```
RABBITMQ_USERNAME: 
  valueFrom:
    secretKeyRef:
      name: rabbitmq-default-user
      key: username
RABBITMQ_PASSWORD: 
  valueFrom:
    secretKeyRef:
      name: rabbitmq-default-user
      key: password
```

### 3. Service Configuration

- **`NUM_SERVICES`**:
Defines the number of consumer services that the load balancer manages.

Example:
```
NUM_SERVICES: "3"
```

### 4. Service-Specific Weights and Admission Rates
Each service has its own set of environment variables that control its routing weights and admission rates:

- **`Initial Weights`**:

SERVICE1_INITIAL_CURR_WEIGHT, SERVICE2_INITIAL_CURR_WEIGHT, etc.: Initial routing weights for each service.

- **`Empty Queue Weights`**:

SERVICE1_INITIAL_EMPTYQ_WEIGHT, SERVICE2_INITIAL_EMPTYQ_WEIGHT, etc.: Weights when the service queue is empty.

- **`Admission Rates`**:

SERVICE1_RAW_ADMISSION_RATE, SERVICE2_RAW_ADMISSION_RATE, etc.: The admission rates for each service.

- **`Alpha and Beta Parameters`**: These parameters control the AIMD algorithm for each service.

SERVICE1_ALPHA, SERVICE1_BETA: Parameters for the first service.
SERVICE2_ALPHA, SERVICE2_BETA: Parameters for the second service.
SERVICE3_ALPHA, SERVICE3_BETA: Parameters for the third service.

Example configuration:

```
SERVICE1_ALPHA: "3"
SERVICE1_BETA: "0.5"
```

### 4. Other Key Variables

- **`CHECK_INTERVAL:`**
Time interval (in milliseconds) for checking the system state.
Example:

```
CHECK_INTERVAL: "1000"
```

- **`MAX_ADMISSION_RATE:`**
The maximum allowed admission rate for services.
Example:
```
MAX_ADMISSION_RATE: "100"
```

## Deployment Steps
- Modify the provided YAML file (loadbalancer.yaml) to set the appropriate environment variable values for your setup.

- Apply the service configuration in Kubernetes:

```
kubectl apply -f loadbalancer.yaml
```
