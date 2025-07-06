# Golang Kubernetes Controller Tutorial

This project is a step-by-step tutorial for DevOps and SRE engineers to learn about building Golang CLI applications and Kubernetes controllers. Each step is implemented as a feature branch and includes a README section with explanations and command history.

---

## Step 1: Golang CLI Application using Cobra

- Initialized a new CLI application using [cobra-cli](https://github.com/spf13/cobra).
- Provides a basic command-line interface.

**Command history:**
```sh
git checkout -b step1-cobra-cli
cobra-cli init --pkg-name github.com/yourusername/k8s-controller-tutorial
# edited main.go, cmd/root.go
```

---

## Step 2: Zerolog for Log Levels

- Integrated [zerolog](https://github.com/rs/zerolog) for structured logging.
- Supports log levels: info, debug, trace, warn, error.

**Command history:**
```sh
git checkout -b step2-zerolog
go get github.com/rs/zerolog
# edited cmd/root.go to add zerolog logging
```

---

## Step 3: pflag for Log Level Flags

- Added [pflag](https://github.com/spf13/pflag) to support a `--log-level` flag.
- Users can set log level via CLI flag.

**Usage:**
```sh
go run main.go --log-level debug
```

**Command history:**
```sh
git checkout -b step3-pflag-loglevel
# edited cmd/root.go to add log-level flag
```
---
## Step 4: FastHTTP Server Command

- Added a new `server` command using [fasthttp](https://github.com/valyala/fasthttp).
- The command starts a FastHTTP server with a configurable port (default: 8080).
- Uses zerolog for logging.

**Usage:**
```sh
go run main.go server --port 8080
```

**What it does:**
- Starts a FastHTTP server on the specified port.
- Responds with "Hello from FastHTTP!" to any request.

**Command history:**
```sh
git checkout -b step4-fasthttp-server
go get github.com/valyala/fasthttp
# created cmd/server.go, added server command
# added cmd/server_test.go for basic tests
go mod tidy
git add .
git commit -m "step4: add fasthttp server command with port flag"
```
---
## Step 5: Makefile, Dockerfile and GitHub workflow

This step introduces the Makefile for build automation, a distroless Dockerfile for secure containerization, a GitHub workflow for CI/CD, and initial test coverage to ensure code quality and deployment readiness.

---
## Step 6: List/Create/Delete Kubernetes Deployments with client-go

- Added a new `list` command using [k8s.io/client-go](https://github.com/kubernetes/client-go).
- Added a new `create` command using [k8s.io/client-go](https://github.com/kubernetes/client-go).
- Added a new `delete` command using [k8s.io/client-go](https://github.com/kubernetes/client-go).
- Lists deployments in the namespace provided as `--namespace` command line argument (`default` by default).
- Supports a `--kubeconfig` flag to specify the kubeconfig file for authentication.
- Supports a `--namespace`,`-n` flag to specify the namespace for the operation.
- Uses zerolog for error logging.

**Usage:**
```sh
git switch feature/step6-list-deployments 
go run main.go --log-level debug --kubeconfig ~/.kube/config list --namespace default
```

**What it does:**
- `list` command:
    - Connects to the Kubernetes cluster using the provided kubeconfig file.
    - Lists all deployments in the namespace provided as `--namespace` command line argument (`default` by default) and prints their names.

---

## Step 7: Deployment Informer with client-go
- Added a Go function to start a shared informer for Deployments in the default namespace using [k8s.io/client-go](https://github.com/kubernetes/client-go).
- The function supports both kubeconfig and in-cluster authentication:
  - If inCluster is true, uses in-cluster config.
  - If kubeconfig is set, uses the provided path.
  - One of these must be set; there is no default to `~/.kube/config`.
- Logs add, update, and delete events for Deployments using zerolog.


**What it does:**
- `server` command:
    - Connects to the Kubernetes cluster using the provided kubeconfig file or in-cluster config.
    - Watches for Deployment events (add, update, delete) in the namespace provided as `--namespace` command line argument (`default` by default) and logs them.

### Informer Test Coverage

The file `pkg/informer/informer_test.go` contains three main test functions:

1. **TestStartDeploymentInformer**
   - Tests the deployment informer event handling and ensures deployment add events are captured.
2. **TestGetDeploymentName**
   - Unit test for the `getDeploymentName` utility, checking both valid and invalid input cases.
3. **TestStartDeploymentInformer_CoversFunction**
   - Ensures the `StartDeploymentInformer` function runs without error.

Each test runs independently when executing `go test ./pkg/informer`. This provides coverage for both informer event handling and utility logic.


### Testing with envtest and Inspecting with kubectl

This project uses [envtest](https://book.kubebuilder.io/reference/envtest.html) to spin up a local Kubernetes API server for integration tests. The test environment writes a kubeconfig to `/tmp/envtest.kubeconfig` so you can inspect the in-memory cluster with `kubectl` while tests are running.

#### How to Run and Inspect

1. **Ensure envtest is installed:**
   ```sh
   ./install-envtest.sh 
   ```

2. **Run the informer test:**
   ```sh
   export KUBEBUILDER_ASSETS="$(pwd)/$(./bin/setup-envtest use --bin-dir ./bin -p path)"
   go test ./pkg/informer -run TestStartDeploymentInformer
   ```
   This will:
   - Start envtest and create sample Deployments
   - Write a kubeconfig to `/tmp/envtest.kubeconfig`
   - Sleep for 5 minutes at the end of the test so you can inspect the cluster

3. **In another terminal, use kubectl:**
   ```sh
   kubectl --kubeconfig=/tmp/envtest.kubeconfig get all -A
   kubectl --kubeconfig=/tmp/envtest.kubeconfig get deployments -n default
   kubectl --kubeconfig=/tmp/envtest.kubeconfig describe pod -n default
   ```
   You can use any standard kubectl commands to inspect resources created by the test.

3. **Notes:**
   - The envtest cluster only exists while the test is running. Once the test finishes, the API server is shut down and the kubeconfig is no longer valid.
   - You can adjust the sleep duration in `TestStartDeploymentInformer` if you need more or less time for inspection.
For more details, see the code in `pkg/testutil/envtest.go` and `pkg/informer/informer_test.go`.

---
## Step 8: /deployments JSON API Endpoint

- Added a `/deployments` endpoint to the FastHTTP server.
- Returns a JSON array of deployment names from the informer's cache (default namespace).
- Uses the informer's local cache, not a live API call.

**Usage:**
```sh
git switch feature/step8-api-handler

go run main.go --log-level trace --kubeconfig ~/.kube/config server

curl http://localhost:8080/deployments
# Output: ["deployment1","deployment2",...]
```

**What it does:**
- Serves a JSON array of deployment names currently in the informer cache.
- Does not query the Kubernetes API directly for each request (fast, efficient).

---
## Step 9: Controller-runtime Deployment Controller

- Integrated [controller-runtime](https://github.com/kubernetes-sigs/controller-runtime) into the project.
- Added a deployment controller that logs each reconcile event for Deployments in the (default namespace by default).
- The controller is started alongside the FastHTTP server.

**What it does:**
- Uses controller-runtime's manager to run a controller for Deployments.
- Logs every reconcile event (creation, update, deletion) for Deployments.

**Usage:**
```sh
git switch feature/step9-controller-runtime

go run main.go --log-level trace --kubeconfig  ~/.kube/config server
```

---
## Step 10: Leader Election and Metrics for Controller Manager

- Added leader election support using a Lease resource (enabled by default, can be disabled with a flag).
- Added a flag to set the metrics port for the controller manager.
- Both features are configurable via CLI flags.

**New flags:**
- `--enable-leader-election` (default: true) — Enable/disable leader election for the controller manager.
- `--metrics-port` (default: 8081) — Port for controller manager metrics endpoint.

**What it does:**
- Ensures only one instance of the controller manager is active at a time (HA support).
- Exposes controller metrics on the specified port.

**Usage:**
```sh
git switch feature/step10-leader-election 

go run main.go server --enable-leader-election=false --metrics-port=9090
```
---

## Step 11: JaegerNginxProxy CRD, Controller, and Webhook Implementation

- Added the Go type for the JaegerNginxProxy custom resource in `pkg/apis/jaeger-nginx-proxy/v1alpha1/resource.go`.
- Created `groupversion_info.go` to define the group, version, and scheme for the CRD.
- Used [controller-gen](https://github.com/kubernetes-sigs/controller-tools) to generate CRD manifests and deepcopy code.
- Implemented a controller for the JaegerNginxProxy CRD using controller-runtime in `pkg/ctrl/JaegerNginxProxy_controller.go`.
- The controller watches JaegerNginxProxy resources and manages both a Deployment and a ConfigMap:
  - Creates/updates a ConfigMap containing the `spec.contents` from the JaegerNginxProxy CR.
  - Creates/updates a Deployment that mounts the ConfigMap as a volume and uses the image/replicas from the CR spec.
  - Cleans up both the Deployment and ConfigMap when the JaegerNginxProxy is deleted.
- Registered and started the controller with the manager in `cmd/server.go`:

```go
      mgr, err := ctrlruntime.NewManager(mgrConfig, manager.Options{
			LeaderElection:          serverEnableLeaderElection,
			LeaderElectionID:        "jaeger-nginx-proxy-controller-leader-election",
			LeaderElectionNamespace: serverLeaderElectionNamespace,
			Metrics:                 server.Options{BindAddress: fmt.Sprintf(":%d", serverMetricsPort)},
		},
		)
		if err != nil {
			log.Error().Err(err).Msg("Failed to create controller-runtime manager")
			os.Exit(1)
		}
```
- **Custom Resource Definition (CRD):** Defines the `JaegerNginxProxy` resource for declarative management of NGINX proxies for Jaeger collectors.
- **Controller:** Watches `JaegerNginxProxy` resources and ensures a Deployment and ConfigMap are created/updated/deleted as needed. Handles status updates reflecting the health of the managed resources.
- **Validating Webhook:** Ensures that only valid `JaegerNginxProxy` resources are admitted by the Kubernetes API server. Performs deep validation of the spec and generated NGINX config.


**What it does:**
- Defines the JaegerNginxProxy CRD structure and registers it with the Kubernetes API machinery.
- Generates the CRD YAML and deepcopy methods required for Kubernetes controllers.
- Reconciles JaegerNginxProxy resources to ensure a matching Deployment and ConfigMap exist in the cluster.
- Updates the Deployment and ConfigMap if the JaegerNginxProxy spec changes.
- Handles creation, update, and cleanup logic for Deployments and ConfigMaps owned by JaegerNginxProxy resources.
- **CRD:**
  - Lets you define a Jaeger NGINX proxy declaratively, including upstreams, ports, image, resources, etc.
- **Controller:**
  - Reconciles the desired state (from the CR) with the actual state in the cluster.
  - Manages a Deployment (NGINX) and a ConfigMap (nginx config) for each CR instance.
  - Updates the CR status to reflect readiness and error messages.
- **Webhook:**
  - Validates new and updated CRs for required fields, port uniqueness, valid port numbers, image fields, and that the generated NGINX config is syntactically valid.
  - Rejects invalid resources before they are persisted.

### CRD Schema (Example)

```yaml
apiVersion: jaeger-nginx-proxy.platform-engineer.stream/v1alpha0
kind: JaegerNginxProxy
metadata:
  name: test-proxy
spec:
  replicaCount: 2
  containerPort: 8080
  image:
    repository: nginx
    tag: "1.21"
    pullPolicy: IfNotPresent
  upstream:
    collectorHost: jaeger-collector.tracing.svc.cluster.local
  ports:
    - name: http
      port: 14268
      path: /api/traces
    - name: grpc
      port: 14250
      path: /jaeger.api.v2.CollectorService/PostSpans
  service:
    type: ClusterIP
  resources:
    limits:
      cpu: 500m
      memory: 512Mi
    requests:
      cpu: 100m
      memory: 128Mi
```

**Usage:**
```sh
git switch feature/step11-jaeger-proxy-crd 
# Add Go types and group version info for JaegerNginxProxy (done already)
# (edit pkg/apis/jaeger-nginx-proxy/v1alpha0/resource.go and groupversion_info.go) (done already)

# install controller-gen binary in your $GOPATH/bin (usually ~/go/bin) 
go install sigs.k8s.io/controller-tools/cmd/controller-gen@latest
export PATH=$PATH:$(go env GOPATH)/bin
# Run controller-gen to generate CRD and deepcopy code
controller-gen crd:crdVersions=v1 paths=./pkg/apis/... output:crd:dir=./config/crd object paths=./pkg/apis/...

# Scaffold and implement the advanced JaegerNginxProxy controller
# created pkg/ctrl/JaegerNginxProxy_controller.go and implemented controller logic for Deployment and ConfigMap management
# registered the controller in cmd/server.go

# Run the server to start the controller
go run main.go --log-level trace --kubeconfig  ~/.kube/config server
```

### Running Tests

This project uses [envtest](https://book.kubebuilder.io/reference/envtest.html) and controller-runtime for integration and controller tests.

### Prerequisites
- Go (see go.mod for version)
- Make
- The `setup-envtest` binary (automatically handled by the Makefile)
- CRD YAMLs present in `config/crd/`


### How to Run

1. **Generate CRD and Webhook Manifests:**
   ```sh
   make generate manifests
   # or manually:
   controller-gen crd:crdVersions=v1 paths=./pkg/apis/... output:crd:dir=./config/crd object paths=./pkg/apis/...
   controller-gen webhook paths=./pkg/apis/... output:webhook:dir=./config/webhook
   ```
2. **Deploy CRD to Cluster:**
   ```sh
   kubectl apply -f config/crd/
   ```
3. **Run the Controller (with webhook server):**
   - Locally:
     ```sh
     make run
     # or
     go run main.go --log-level trace --kubeconfig ~/.kube/config server
     ```
   - In-cluster: Deploy the controller Deployment and Service (see Helm chart or manifests).
4. **Create a JaegerNginxProxy resource:**
   ```sh
   kubectl apply -f examples/jaegernginxproxy.yaml
   ```
5. **Check status and managed resources:**
   ```sh
   kubectl get jaegernginxproxies
   kubectl describe jaegernginxproxy <name>
   kubectl get deployment,cm
   ```

### Webhook Details

- **Type:** Validating Admission Webhook
- **Path:** `/validate-jaeger-nginx-proxy-platform-engineer-stream-v1alpha0-jaegernginxproxy`
- **Operations:** create, update
- **Validation performed:**
  - Required fields (replicaCount, image, ports, etc.)
  - Port uniqueness and valid ranges
  - Image fields are non-empty
  - Resources (CPU/memory) are set
  - NGINX config can be generated and passes basic validation
- **Failure Policy:** fail (invalid CRs are rejected)
- **How it is wired:** Registered with the controller-runtime manager in the main application. The webhook server is started automatically when running the controller.

## Project Structure

- `.github/workflows/` — GitHub Actions workflows for CI/CD.
- `charts/app` — Helm chart for deployment
- `cmd/` — CLI commands
    - `cmd/server.go` — FastHTTP server
    - `cmd/list.go` — List CLI command
    - `cmd/delete.go` — Delete CLI command
    - `cmd/create.go` — Create CLI command
- `config/crd/` — CRD definitions
- `config/webhook/` — Webhook configuration manifests
- `pkg/apis/` — CRD Go types and deepcopy
- `pkg/ctrl/` — Controller logic (reconcilers)
- `pkg/informer/` — Informer implementation
- `pkg/testutil/` — envtest kit
- `pkg/webhook/` — Webhook implementation (validation logic)
- `main.go` — Entry point
- `Makefile` — Build automation
- `Dockerfile` — Distroless Dockerfile

---


## License

MIT License. See [LICENSE](LICENSE) for details.
