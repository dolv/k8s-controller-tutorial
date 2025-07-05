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

## Project Structure
- `.github/workflows/` — GitHub Actions workflows for CI/CD.
- `charts/app` - helm chart
- `cmd/` — Contains your CLI commands.
    - `cmd/server.go` - fasthttp server
    - `cmd/list.go` - list cli command
- `pkg/informer` - informer implementation
- `pkg/testutil` - envtest kit
- `main.go` — Entry point for your application.
- `Makefile` — Build automation tasks.
- `Dockerfile` — Distroless Dockerfile for secure containerization.

## License

MIT License. See [LICENSE](LICENSE) for details.

