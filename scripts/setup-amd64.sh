#!/bin/bash

# Exit on error
set -e

echo "Setting up Kubernetes control plane for AMD64..."
APP_NAME="k8s-controller"
LOG_DIR="/var/log/${APP_NAME}"

HOST_IP=$(hostname -I | awk '{print $1}')
export HOST_IP


# Function to check if a process is running
is_running() {
    pgrep -f "$1" >/dev/null
}

# Function to check if all components are running
check_running() {
    is_running "etcd" && \
    is_running "kube-apiserver" && \
    is_running "kube-controller-manager" && \
    is_running "cloud-controller-manager" && \
    is_running "kube-scheduler" && \
    is_running "kubelet" && \
    is_running "containerd"
}

# Function to kill process if running
stop_process() {
    if is_running "$1"; then
        echo "Stopping $1..."
        sudo pkill -f "$1" || true
        while is_running "$1"; do
            sleep 1
        done
    fi
}

download_components() {
    # Create necessary directories if they don't exist
    sudo mkdir -p ./kubebuilder/bin
    sudo mkdir -p /etc/cni/net.d
    sudo mkdir -p /var/lib/kubelet
    sudo mkdir -p /etc/kubernetes/manifests
    sudo mkdir -p /var/log/kubernetes
    sudo mkdir -p "${LOG_DIR}"
    sudo mkdir -p /etc/containerd/
    sudo mkdir -p /run/containerd


    # Download kubebuilder tools if not present
    if [ ! -f "kubebuilder/bin/etcd" ]; then
        echo "Downloading kubebuilder tools..."
        curl -L https://storage.googleapis.com/kubebuilder-tools/kubebuilder-tools-1.30.0-linux-amd64.tar.gz -o /tmp/kubebuilder-tools.tar.gz
        sudo tar -C ./kubebuilder --strip-components=1 -zxf /tmp/kubebuilder-tools.tar.gz
        rm /tmp/kubebuilder-tools.tar.gz
        sudo chmod -R 755 ./kubebuilder/bin
    fi

    if [ ! -f "kubebuilder/bin/kubelet" ]; then
        echo "Downloading kubelet..."
        sudo curl -L "https://dl.k8s.io/v1.30.0/bin/linux/amd64/kubelet" -o kubebuilder/bin/kubelet
        sudo chmod 755 kubebuilder/bin/kubelet
    fi

    # Install CNI components if not present
    if [ ! -d "/opt/cni" ]; then
        sudo mkdir -p /opt/cni
        
        echo "Installing containerd..."
        wget https://github.com/containerd/containerd/releases/download/v2.0.5/containerd-static-2.0.5-linux-amd64.tar.gz -O /tmp/containerd.tar.gz
        sudo tar zxf /tmp/containerd.tar.gz -C /opt/cni/
        rm /tmp/containerd.tar.gz

        echo "Installing runc..."
        sudo curl -L "https://github.com/opencontainers/runc/releases/download/v1.2.6/runc.amd64" -o /opt/cni/bin/runc

        echo "Installing CNI plugins..."
        wget https://github.com/containernetworking/plugins/releases/download/v1.6.2/cni-plugins-linux-amd64-v1.6.2.tgz -O /tmp/cni-plugins.tgz
        sudo tar zxf /tmp/cni-plugins.tgz -C /opt/cni/bin/
        rm /tmp/cni-plugins.tgz

        # Set permissions for all CNI components
        sudo chmod -R 755 /opt/cni
    fi

    if [ ! -f "kubebuilder/bin/kube-controller-manager" ]; then
        echo "Downloading additional components..."
        sudo curl -L "https://dl.k8s.io/v1.30.0/bin/linux/amd64/kube-controller-manager" -o kubebuilder/bin/kube-controller-manager
        sudo curl -L "https://dl.k8s.io/v1.30.0/bin/linux/amd64/kube-scheduler" -o kubebuilder/bin/kube-scheduler
        sudo curl -L "https://dl.k8s.io/v1.30.0/bin/linux/amd64/cloud-controller-manager" -o kubebuilder/bin/cloud-controller-manager
        sudo chmod 755 kubebuilder/bin/kube-controller-manager
        sudo chmod 755 kubebuilder/bin/kube-scheduler
        sudo chmod 755 kubebuilder/bin/cloud-controller-manager
    fi
}

setup_configs() {
    # Generate certificates and tokens if they don't exist
    if [ ! -f "/tmp/sa.key" ]; then
        openssl genrsa -out /tmp/sa.key 2048
        openssl rsa -in /tmp/sa.key -pubout -out /tmp/sa.pub
    fi

    if [ ! -f "/tmp/token.csv" ]; then
        TOKEN="1234567890"
        echo "${TOKEN},admin,admin,system:masters" > /tmp/token.csv
    fi

    # Always regenerate and copy CA certificate to ensure it exists
    echo "Generating CA certificate..."
    openssl genrsa -out /tmp/ca.key 2048
    openssl req -x509 -new -nodes -key /tmp/ca.key -subj "/CN=kubelet-ca" -days 365 -out /tmp/ca.crt
    sudo mkdir -p /var/lib/kubelet/pki
    sudo cp /tmp/ca.crt /var/lib/kubelet/ca.crt
    sudo cp /tmp/ca.crt /var/lib/kubelet/pki/ca.crt

    # Set up kubeconfig if not already configured
    if ! sudo kubebuilder/bin/kubectl config current-context | grep -q "test-context"; then
        sudo kubebuilder/bin/kubectl config set-credentials test-user --token=1234567890
        sudo kubebuilder/bin/kubectl config set-cluster test-env --server=https://127.0.0.1:6443 --certificate-authority=/tmp/ca.crt
        sudo kubebuilder/bin/kubectl config set-context test-context --cluster=test-env --user=test-user --namespace=default 
        sudo kubebuilder/bin/kubectl config use-context test-context
    fi

    # Configure CNI
    cat <<EOF | sudo tee /etc/cni/net.d/10-mynet.conf
{
    "cniVersion": "0.3.1",
    "name": "mynet",
    "type": "bridge",
    "bridge": "cni0",
    "isGateway": true,
    "ipMasq": true,
    "ipam": {
        "type": "host-local",
        "subnet": "10.22.0.0/16",
        "routes": [
            { "dst": "0.0.0.0/0" }
        ]
    }
}
EOF

    # Configure containerd
    cat <<EOF | sudo tee /etc/containerd/config.toml
version = 3

[grpc]
  address = "/run/containerd/containerd.sock"

[plugins.'io.containerd.cri.v1.runtime']
  enable_selinux = false
  enable_unprivileged_ports = true
  enable_unprivileged_icmp = true
  device_ownership_from_security_context = false

[plugins.'io.containerd.cri.v1.images']
  snapshotter = "native"
  disable_snapshot_annotations = true

[plugins.'io.containerd.cri.v1.runtime'.cni]
  bin_dir = "/opt/cni/bin"
  conf_dir = "/etc/cni/net.d"

[plugins.'io.containerd.cri.v1.runtime'.containerd.runtimes.runc]
  runtime_type = "io.containerd.runc.v2"

[plugins.'io.containerd.cri.v1.runtime'.containerd.runtimes.runc.options]
  SystemdCgroup = false
EOF

    # Ensure containerd data directory exists with correct permissions
    sudo mkdir -p /var/lib/containerd
    sudo chmod 711 /var/lib/containerd

    # Configure kubelet
    cat << EOF | sudo tee /var/lib/kubelet/config.yaml
apiVersion: kubelet.config.k8s.io/v1beta1
kind: KubeletConfiguration
authentication:
  anonymous:
    enabled: true
  webhook:
    enabled: true
  x509:
    clientCAFile: "/var/lib/kubelet/ca.crt"
authorization:
  mode: AlwaysAllow
clusterDomain: "cluster.local"
clusterDNS:
  - "10.0.0.10"
resolvConf: "/etc/resolv.conf"
runtimeRequestTimeout: "15m"
failSwapOn: false
seccompDefault: true
serverTLSBootstrap: false
containerRuntimeEndpoint: "unix:///run/containerd/containerd.sock"
staticPodPath: "/etc/kubernetes/manifests"
EOF

    # Create required directories with proper permissions
    sudo mkdir -p /var/lib/kubelet/pods
    sudo chmod 750 /var/lib/kubelet/pods
    sudo mkdir -p /var/lib/kubelet/plugins
    sudo chmod 750 /var/lib/kubelet/plugins
    sudo mkdir -p /var/lib/kubelet/plugins_registry
    sudo chmod 750 /var/lib/kubelet/plugins_registry

    # Ensure proper permissions
    sudo chmod 644 /var/lib/kubelet/ca.crt
    sudo chmod 644 /var/lib/kubelet/config.yaml

    # Generate self-signed kubelet serving certificate if not present
    if [ ! -f "/var/lib/kubelet/pki/kubelet.crt" ] || [ ! -f "/var/lib/kubelet/pki/kubelet.key" ]; then
    echo "Generating kubelet serving key..."
    sudo openssl genrsa -out /var/lib/kubelet/pki/kubelet.key 2048

    echo "Generating kubelet certificate signing request (CSR)..."
    cat <<EOF > /tmp/kubelet-csr.conf
[req]
req_extensions = v3_req
distinguished_name = req_distinguished_name
[req_distinguished_name]
[ v3_req ]
basicConstraints = CA:FALSE
keyUsage = nonRepudiation, digitalSignature, keyEncipherment
extendedKeyUsage = serverAuth
subjectAltName = @alt_names
[ alt_names ]
DNS.1 = $(hostname)
IP.1 = $HOST_IP
EOF

    sudo openssl req -new -key /var/lib/kubelet/pki/kubelet.key \
        -subj "/CN=kubelet" \
        -out /tmp/kubelet.csr \
        -config /tmp/kubelet-csr.conf

    echo "Signing kubelet certificate with cluster CA..."
    sudo openssl x509 -req -in /tmp/kubelet.csr \
        -CA /tmp/ca.crt -CAkey /tmp/ca.key -CAcreateserial \
        -out /var/lib/kubelet/pki/kubelet.crt \
        -days 365 \
        -extensions v3_req -extfile /tmp/kubelet-csr.conf

    sudo chmod 600 /var/lib/kubelet/pki/kubelet.key
    sudo chmod 644 /var/lib/kubelet/pki/kubelet.crt
    fi


    # Generate apiserver key
    sudo openssl genrsa -out /tmp/apiserver.key 2048

    # Create a CSR config (save as /tmp/apiserver-csr.cnf)
    cat <<EOF > /tmp/apiserver-csr.cnf
[req]
req_extensions = v3_req
distinguished_name = req_distinguished_name
[req_distinguished_name]
[ v3_req ]
basicConstraints = CA:FALSE
keyUsage = nonRepudiation, digitalSignature, keyEncipherment
extendedKeyUsage = serverAuth
subjectAltName = @alt_names
[ alt_names ]
DNS.1 = kubernetes
DNS.2 = kubernetes.default
DNS.3 = kubernetes.default.svc
DNS.4 = kubernetes.default.svc.cluster.local
IP.1 = 10.0.0.1
IP.2 = 127.0.0.1
IP.3 = $HOST_IP
EOF

    # Generate the CSR
    sudo openssl req -new -key /tmp/apiserver.key -subj "/CN=kube-apiserver" -out /tmp/apiserver.csr -config /tmp/apiserver-csr.cnf

    # Sign the cert
    sudo openssl x509 -req -in /tmp/apiserver.csr -CA /tmp/ca.crt -CAkey /tmp/ca.key -CAcreateserial \
    -out /tmp/apiserver.crt -days 365 -extensions v3_req -extfile /tmp/apiserver-csr.cnf

}

start() {
    if check_running; then
        echo "Kubernetes components are already running"
        return 0
    fi
    
    # Download components if needed
    download_components
    
    # Setup configurations
    setup_configs

    # Start components if not running
    if ! is_running "etcd"; then
        sudo bash -c '
            {  
                echo "Starting etcd..."
                nohup kubebuilder/bin/etcd \
                --advertise-client-urls http://'"$HOST_IP"':2379 \
                --listen-client-urls http://0.0.0.0:2379 \
                --data-dir ./etcd \
                --listen-peer-urls http://0.0.0.0:2380 \
                --initial-cluster default=http://'"$HOST_IP"':2380 \
                --initial-advertise-peer-urls http://'"$HOST_IP"':2380 \
                --initial-cluster-state new \
                --initial-cluster-token test-token
            } 2>&1 | tee '"${LOG_DIR}"'/etcd.out.log &
        '
    fi

    if ! is_running "kube-apiserver"; then
        sudo bash -c '
            {
                echo "Starting kube-apiserver..."
                echo "use application/vnd.kubernetes.protobuf for better performance"
                kubebuilder/bin/kube-apiserver \
                    --etcd-servers=http://'"$HOST_IP"':2379 \
                    --service-cluster-ip-range=10.0.0.0/24 \
                    --bind-address=0.0.0.0 \
                    --secure-port=6443 \
                    --advertise-address='"$HOST_IP"' \
                    --authorization-mode=AlwaysAllow \
                    --token-auth-file=/tmp/token.csv \
                    --enable-priority-and-fairness=false \
                    --allow-privileged=true \
                    --profiling=false \
                    --storage-backend=etcd3 \
                    --storage-media-type=application/json \
                    --v=0 \
                    --service-account-issuer=https://kubernetes.default.svc.cluster.local \
                    --service-account-key-file=/tmp/sa.pub \
                    --service-account-signing-key-file=/tmp/sa.key \
                    --tls-cert-file=/tmp/apiserver.crt \
                    --tls-private-key-file=/tmp/apiserver.key \
                    --client-ca-file=/tmp/ca.crt
            } 2>&1 | tee '"${LOG_DIR}"'/kube-apiserver.out.log &'
    fi

    if ! is_running "containerd"; then
        sudo bash -c '
            {
                echo "Starting containerd..."
                export PATH='"$PATH"':/opt/cni/bin:kubebuilder/bin
                PATH='"$PATH"':/opt/cni/bin:/usr/sbin /opt/cni/bin/containerd -c /etc/containerd/config.toml 
            } 2>&1 | tee '"${LOG_DIR}"'/containerd.out.log &'
    fi

    if ! is_running "kube-scheduler"; then
        sudo bash -c '
            {
                echo "Starting kube-scheduler..."
                kubebuilder/bin/kube-scheduler \
                    --kubeconfig=/root/.kube/config \
                    --leader-elect=false \
                    --v=2 \
                    --bind-address=0.0.0.0
            } 2>&1 | tee '"${LOG_DIR}"'/kube-scheduler.out.log &'
    fi

    # Set up kubelet kubeconfig
    sudo cp /root/.kube/config /var/lib/kubelet/kubeconfig
    export KUBECONFIG=~/.kube/config

    # Create service account and configmap if they don't exist
    sudo kubebuilder/bin/kubectl create sa default 2>/dev/null || true
    sudo kubebuilder/bin/kubectl create configmap kube-root-ca.crt --from-file=ca.crt=/tmp/ca.crt -n default 2>/dev/null || true


    if ! is_running "kubelet"; then
        sudo bash -c '
            {
                echo "Starting kubelet..."
                PATH=$PATH:/opt/cni/bin:/usr/sbin kubebuilder/bin/kubelet \
                    --kubeconfig=/var/lib/kubelet/kubeconfig \
                    --config=/var/lib/kubelet/config.yaml \
                    --root-dir=/var/lib/kubelet \
                    --cert-dir=/var/lib/kubelet/pki \
                    --tls-cert-file=/var/lib/kubelet/pki/kubelet.crt \
                    --tls-private-key-file=/var/lib/kubelet/pki/kubelet.key \
                    --hostname-override='"$(hostname)"' \
                    --pod-infra-container-image=registry.k8s.io/pause:3.10 \
                    --node-ip='"$HOST_IP"' \
                    --cgroup-driver=cgroupfs \
                    --max-pods=10  \
                    --v=1
            } 2>&1 | tee '"${LOG_DIR}"'/kubelet.out.log &'
    fi

    # Label the node so static pods with nodeSelector can be scheduled
    NODE_NAME=$(hostname)
    sudo kubebuilder/bin/kubectl label node "$NODE_NAME" node-role.kubernetes.io/master="" --overwrite || true

    # Remove the uninitialized taint so pods can be scheduled
    #sudo kubebuilder/bin/kubectl taint nodes "$NODE_NAME" node.cloudprovider.kubernetes.io/uninitialized:NoSchedule- || true

    if ! is_running "kube-controller-manager"; then
        sudo bash -c '
            {
                echo "Starting kube-controller-manager..."
                PATH='"$PATH"':/opt/cni/bin:/usr/sbin kubebuilder/bin/kube-controller-manager \
                    --kubeconfig=/var/lib/kubelet/kubeconfig \
                    --leader-elect=false \
                    --service-cluster-ip-range=10.0.0.0/24 \
                    --cluster-name=kubernetes \
                    --root-ca-file=/var/lib/kubelet/ca.crt \
                    --service-account-private-key-file=/tmp/sa.key \
                    --use-service-account-credentials=true \
                    --v=2
            } 2>&1 | tee '"${LOG_DIR}"'/kube-controller-manager.out.log &'
    fi

    echo "Waiting for components to be ready..."
    sleep 10

    echo "Verifying setup..."
    sudo kubebuilder/bin/kubectl get nodes
    sudo kubebuilder/bin/kubectl get all -A
    sudo kubebuilder/bin/kubectl get componentstatuses || true
    sudo kubebuilder/bin/kubectl get --raw='/readyz?verbose'

    sleep 5
    echo "Deploy Kube-Proxy..."
    sudo kubectl apply -f config/kube-proxy/
    sleep 10
    echo "Deploy CoreDNS..."
    sudo kubectl apply -f config/codedns/
}

stop() {
    echo "Stopping Kubernetes components..."
    stop_process "cloud-controller-manager"
    stop_process "gce_metadata_server"
    stop_process "kube-controller-manager"
    stop_process "kubelet"
    stop_process "kube-scheduler"
    stop_process "kube-apiserver"
    stop_process "containerd"
    stop_process "etcd"
    echo "All components stopped"
}

cleanup() {
    stop
    echo "Cleaning up..."
    sudo rm -rf ./etcd
    sudo rm -rf /var/lib/kubelet/*
    sudo rm -rf /run/containerd/*
    sudo rm -f /tmp/sa.key /tmp/sa.pub /tmp/token.csv /tmp/ca.key /tmp/ca.crt /tmp/ca.srl /tmp/apiserver.csr /tmp/apiserver.key /tmp/apiserver.crt 
    echo "Cleanup complete"
}

case "${1:-}" in
    start)
        start
        ;;
    stop)
        stop
        ;;
    cleanup)
        cleanup
        ;;
    *)
        echo "Usage: $0 {start|stop|cleanup}"
        exit 1
        ;;
esac 