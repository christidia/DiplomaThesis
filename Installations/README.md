# Prerequisites

To perform the installations, ensure that the following are installed and running on the machine:
- [Java Runtime Enviroment](https://ubuntu.com/tutorials/install-jre#2-installing-openjdk-jre)
- [Helm](https://helm.sh/docs/intro/install/)
- [kubectl](https://kubernetes.io/docs/tasks/tools/install-kubectl-linux/)
   
## Docker Installation Guide for Ubuntu
[Docker Installation Guide for Ubuntu](https://docs.docker.com/engine/install/ubuntu/)

To run this project, you need to have Docker installed on your Ubuntu system. Follow the steps below to install Docker using the APT repository.

### Step 1: Update the Package Index

First, update your existing list of packages:

```bash
sudo apt update
```

### Step 2: Install Required Packages
Install a few prerequisite packages which let apt use packages over HTTPS:

```bash
sudo apt install apt-transport-https ca-certificates curl software-properties-common
```

### Step 3: Add Docker’s Official GPG Key
Add the GPG key for the official Docker repository to your system:

```bash
curl -fsSL https://download.docker.com/linux/ubuntu/gpg | sudo gpg --dearmor -o /usr/share/keyrings/docker-archive-keyring.gpg
```

### Step 4: Add the Docker Repository
Add the Docker repository to APT sources:

```bash
echo "deb [arch=amd64 signed-by=/usr/share/keyrings/docker-archive-keyring.gpg] https://download.docker.com/linux/ubuntu $(lsb_release -cs) stable" | sudo tee /etc/apt/sources.list.d/docker.list > /dev/null
```

### Step 5: Update the Package Index Again

```bash
sudo apt update
```

### Step 6: Install Docker

```bash
sudo apt install docker-ce docker-ce-cli containerd.io
```

### Step 8: Test Docker Installation
This command downloads a test image and runs it in a container. When the container runs, it prints a confirmation message and exits.

```bash
sudo docker run hello-world
```

### Step 9: Add your user to the Docker group
Add your user to the Docker group to run Docker commands without sudo

```bash
sudo usermod -aG docker ${USER}
newgrp docker
```

## Minikube Installation Guide for Ubuntu
[Minikube Installation Guide for Ubuntu](https://minikube.sigs.k8s.io/docs/start/?arch=%2Flinux%2Fx86-64%2Fstable%2Fbinary+download)

To install the latest minikube stable release on x86-64 Linux using binary download:
```bash
curl -LO https://storage.googleapis.com/minikube/releases/latest/minikube-linux-amd64
sudo install minikube-linux-amd64 /usr/local/bin/minikube && rm minikube-linux-amd64
```

# Setting up the Minikube Node

To setup the minikube node and run the experiments, run the script: [minikube_setup.sh](./minikube_setup.sh) and perform the following configurations: 

## Edit the Metrics Server Deployment
Sometimes the metrics server deployment needs to be edited to work correctly in certain environments. You might need to add --kubelet-insecure-tls to the args:

```bash
kubectl edit deployment metrics-server -n kube-system
```

Add the following under spec.containers.args:
```bash
- --kubelet-insecure-tls
```
