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


Continue to the next steps for more advanced Kubernetes and controller features! 

# List Kubernetes Deployments with client-go

- Added a new `list` command using [k8s.io/client-go](https://github.com/kubernetes/client-go).
- Lists deployments in the default namespace.
- Supports a `--kubeconfig` flag to specify the kubeconfig file for authentication.
- Uses zerolog for error logging.

**Usage:**
```sh
git switch feature/step6-list-deployments 
go run main.go --log-level debug --kubeconfig ~/.kube/config list
```

**What it does:**
- `list` command:
    - Connects to the Kubernetes cluster using the provided kubeconfig file.
    - Lists all deployments in the namespace provided as `--namespace` command line argument (`default` by default) and prints their names.
- `server` command:
    - Connects to the Kubernetes cluster using the provided kubeconfig file or in-cluster config.
    - Watches for Deployment events (add, update, delete) in the namespace provided as `--namespace` command line argument (`default` by default) and logs them.

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

