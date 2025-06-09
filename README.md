# k8s-controller-tutorial

A starter template for building Kubernetes controllers or CLI tools in Go using [cobra-cli](https://github.com/spf13/cobra-cli).

## Prerequisites

- [Go](https://golang.org/dl/) 1.18 or newer
- [cobra-cli](https://github.com/spf13/cobra-cli) installed:
  ```sh
  go install github.com/spf13/cobra-cli@latest
  ```

## Getting Started

1. **Clone this repository and initialize the Go module:**
   ```sh
   git clone https://github.com/yourusername/k8s-controller-tutorial.git
   cd k8s-controller-tutorial
   go mod init github.com/yourusername/k8s-controller-tutorial
   ```
   Make sure the LICENSE file is set to MIT and updated with your name and year (e.g., 2025 Denys Vasyliev).

2. **Add zerolog for structured logging, support log-level flag, and build your CLI:**
   Integrate [zerolog](https://github.com/rs/zerolog) for structured logging with log levels: info, debug, trace, warn, and error. Add a `--log-level` flag to control the log level at runtime.

   Install zerolog:
   ```sh
   go get github.com/rs/zerolog/log
   ```

   Example usage in your `main.go`:
   ```go
   import (
       "os"
       "github.com/rs/zerolog"
       "github.com/rs/zerolog/log"
       "github.com/spf13/cobra"
   )

   var logLevel string

   func main() {
       rootCmd := &cobra.Command{
           Use:   "controller",
           Short: "A Kubernetes controller CLI",
           Run: func(cmd *cobra.Command, args []string) {
               level, err := zerolog.ParseLevel(logLevel)
               if err != nil {
                   level = zerolog.InfoLevel
               }
               zerolog.SetGlobalLevel(level)
               log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})

               log.Info().Msg("info message")
               log.Debug().Msg("debug message")
               log.Trace().Msg("trace message")
               log.Warn().Msg("warn message")
               log.Error().Msg("error message")
           },
       }

       rootCmd.PersistentFlags().StringVar(&logLevel, "log-level", "info", "Set log level: trace, debug, info, warn, error")
       rootCmd.Execute()
   }
   ```

   Build your CLI:
   ```sh
   go build -o controller
   ```

   You can now run your CLI with different log levels:
   ```sh
   ./controller --log-level debug
   ./controller --log-level trace
   ```

3. **FastHTTP Server Command with Log Level Flag:**

   - Added a new `server` command using [fasthttp](https://github.com/valyala/fasthttp).
   - The command starts a FastHTTP server with a configurable port (default: 8080).
   - Supports the `--log-level` flag for controlling log verbosity.
   - Uses zerolog for logging.

   **Usage:**
   ```sh
   go run main.go server --port 8080 --log-level debug
   ```

   **What it does:**
   - Starts a FastHTTP server on the specified port.
   - Responds with "Hello from FastHTTP!" to any request.
   - Respects the log level set by the `--log-level` flag.

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

## Project Structure

- `cmd/` — Contains your CLI commands.
- `main.go` — Entry point for your application.

## License

MIT License. See [LICENSE](LICENSE) for details.
