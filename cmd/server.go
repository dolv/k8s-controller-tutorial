package cmd

import (
	"context"
	"fmt"
	"os"
	"time"

	cfgPkg "github.com/dolv/k8s-controller-tutorial/internal/config"
	"github.com/dolv/k8s-controller-tutorial/pkg/api"
	jaegernginxproxyv1alpha0 "github.com/dolv/k8s-controller-tutorial/pkg/apis/jaeger-nginx-proxy/v1alpha0"
	"github.com/dolv/k8s-controller-tutorial/pkg/ctrl"
	"github.com/dolv/k8s-controller-tutorial/pkg/informer"
	"github.com/google/uuid"
	mcpserver "github.com/mark3labs/mcp-go/server"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"github.com/valyala/fasthttp"
	"github.com/valyala/fasthttprouter"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	ctrlruntime "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	ctrlruntimelog "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/metrics/server"
)

var (
	serverPort                    int
	serverMetricsPort             int
	serverKubeconfig              string
	serverInCluster               bool
	serverEnableLeaderElection    bool
	serverLeaderElectionNamespace string
	serverEnableMCP               bool
	serverMCPPort                 int
)

const (
	requestIDKey = "requestID"
	loggerKey    = "logger"
)

func loggingMiddleware(next fasthttp.RequestHandler) fasthttp.RequestHandler {
	return func(ctx *fasthttp.RequestCtx) {
		start := time.Now()
		requestID := uuid.New().String()
		ctx.SetUserValue(requestIDKey, requestID)
		ctx.Response.Header.Set("X-Request-ID", requestID)
		next(ctx)
		duration := time.Since(start)
		log.Debug().
			Str("method", string(ctx.Method())).
			Str("path", string(ctx.Path())).
			Str("remote_ip", ctx.RemoteIP().String()).
			Int("status", ctx.Response.StatusCode()).
			Dur("latency", duration).
			Str("request_id", requestID).
			Msg("HTTP request")
	}
}

func getServerKubeClient(kubeconfigPath string, inCluster bool) (*kubernetes.Clientset, error) {
	var config *rest.Config
	var err error
	if inCluster {
		config, err = rest.InClusterConfig()
	} else {
		config, err = cfgPkg.GetKubeConfig(kubeconfigPath)
		if err != nil {
			log.Error().Err(err).Msg("Failed to build kubeconfig rest object")
			return nil, err
		}
	}
	if err != nil {
		return nil, err
	}
	return kubernetes.NewForConfig(config)
}

var serverCmd = &cobra.Command{
	Use:   "server",
	Short: "Start a FastHTTP server",
	Run: func(cmd *cobra.Command, args []string) {
		log.Trace().Msg("This is a top-level trace log from serverCmd.Run")
		log.Trace().Msg("Getting clientset instance")
		clientset, err := getServerKubeClient(serverKubeconfig, serverInCluster)
		if err != nil {
			log.Error().Err(err).Msg("Failed to create Kubernetes client")
			os.Exit(1)
		}
		ctx := context.Background()

		log.Trace().Msg("Starting Informer")
		// Use namespaces parameter if provided, otherwise fall back to namespace
		namespaceToWatch := namespace
		if namespaces != "" {
			namespaceToWatch = namespaces
		}
		go informer.StartDeploymentInformer(ctx, clientset, namespaceToWatch)

		// Start controller-runtime manager and controller
		log.Trace().Msg("Starting Controller-runtime manager")

		// Use the same kubeconfig as the server client
		var mgrConfig *rest.Config
		if serverInCluster {
			mgrConfig, err = rest.InClusterConfig()
		} else {
			mgrConfig, err = cfgPkg.GetKubeConfig(serverKubeconfig)
		}
		if err != nil {
			log.Error().Err(err).Msg("Failed to get kubeconfig for controller-runtime manager")
			os.Exit(1)
		}

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

		// Register the JaegerNginxProxy CRD scheme
		if err := jaegernginxproxyv1alpha0.AddToScheme(mgr.GetScheme()); err != nil {
			log.Error().Err(err).Msg("Failed to add JaegerNginxProxy scheme")
			os.Exit(1)
		}

		// Add the JaegerNginxProxy controller
		if err := ctrl.AddJaegerNginxProxyController(mgr); err != nil {
			log.Error().Err(err).Msg("Failed to add JaegerNginxProxy controller")
			os.Exit(1)
		}

		// Webhook registration is handled automatically by kubebuilder markers

		go func() {
			log.Info().Msg("Starting controller-runtime manager...")
			if err := mgr.Start(cmd.Context()); err != nil {
				log.Error().Err(err).Msg("Manager exited with error")
				os.Exit(1)
			}
		}()

		// Set up controller-runtime logging
		ctrlruntimelog.SetLogger(zap.New(zap.UseDevMode(true)))

		// --- API ROUTER SETUP ---
		// Import here to avoid import cycle in code edit
		jaegerapi := requireJaegerNginxProxyAPI(mgr.GetClient(), namespace)
		router := requireFasthttprouter()
		// JaegerNginxProxy API endpoints
		router.GET("/api/jaegernginxproxies", adaptHandler(jaegerapi.ListJaegerNginxProxies))
		router.GET("/api/jaegernginxproxies/:name", adaptHandler(jaegerapi.GetJaegerNginxProxy))
		router.POST("/api/jaegernginxproxies", adaptHandler(jaegerapi.CreateJaegerNginxProxy))
		router.PUT("/api/jaegernginxproxies/:name", adaptHandler(jaegerapi.UpdateJaegerNginxProxy))
		router.PATCH("/api/jaegernginxproxies/:name", adaptHandler(jaegerapi.PatchJaegerNginxProxy))
		router.DELETE("/api/jaegernginxproxies/:name", adaptHandler(jaegerapi.DeleteJaegerNginxProxy))
		// Swagger documentation endpoints
		router.GET("/docs/swagger.json", adaptHandler(serveSwaggerJSON))
		router.GET("/swagger", adaptHandler(serveSwaggerUI))
		router.GET("/swagger/", adaptHandler(serveSwaggerUI))
		// --- END API ROUTER SETUP ---

		log.Trace().Msg("Getting handler instance")
		handler := func(ctx *fasthttp.RequestCtx) {
			logger, ok := ctx.UserValue(loggerKey).(zerolog.Logger)
			if !ok {
				logger = log.Logger
			}
			logger.Trace().Msg("Handler entered")

			// Check for /deployments endpoint first (before router)
			if string(ctx.Path()) == "/deployments" {
				logger.Info().Msg("Deployments request received")
				ctx.Response.Header.Set("Content-Type", "application/json")

				// Check if user wants to see deployments with namespace info
				queryArgs := ctx.QueryArgs()
				if queryArgs.Has("with-namespace") {
					deployments := informer.GetDeploymentNamesWithNamespace()
					logger.Info().Msgf("Deployments with namespace: %v", deployments)
					ctx.SetStatusCode(200)
					ctx.Write([]byte("["))
					for i, deployment := range deployments {
						ctx.WriteString("{\"name\":\"")
						ctx.WriteString(deployment["name"])
						ctx.WriteString("\",\"namespace\":\"")
						ctx.WriteString(deployment["namespace"])
						ctx.WriteString("\"}")
						if i < len(deployments)-1 {
							ctx.WriteString(",")
						}
					}
					ctx.Write([]byte("]"))
				} else {
					deployments := informer.GetDeploymentNames()
					logger.Info().Msgf("Deployments: %v", deployments)
					ctx.SetStatusCode(200)
					ctx.Write([]byte("["))
					for i, name := range deployments {
						ctx.WriteString("\"")
						ctx.WriteString(name)
						ctx.WriteString("\"")
						if i < len(deployments)-1 {
							ctx.WriteString(",")
						}
					}
					ctx.Write([]byte("]"))
				}
				return
			}

			// API router takes precedence for all other routes
			if router != nil {
				router.Handler(ctx)
				if ctx.Response.StatusCode() != 0 {
					return
				}
			}

			// Default handler for unmatched routes
			logger.Info().Msg("Default request received")
			fmt.Fprintf(ctx, "Hello from FastHTTP! Your request ID: %s", ctx.UserValue(requestIDKey))
			logger.Trace().Msg("Handler exiting")
		}
		log.Trace().Msg("Adding loggingMiddleware to handler instance")
		wrappedHandler := loggingMiddleware(handler)

		if serverEnableMCP {
			go func() {
				mcpServer := NewMCPServer("K8s Controller MCP", appVersion)

				// Set the global API instance for MCP handlers
				api.JaegerNginxProxyAPIInst = &api.JaegerNginxProxyAPI{
					K8sClient: mgr.GetClient(),
					Namespace: namespace,
				}

				sseServer := mcpserver.NewSSEServer(mcpServer,
					mcpserver.WithBaseURL(fmt.Sprintf("http://:%d", serverMCPPort)),
				)
				log.Info().Msgf("Starting MCP server in SSE mode on port %d", serverMCPPort)
				if err := sseServer.Start(fmt.Sprintf(":%d", serverMCPPort)); err != nil {
					log.Fatal().Err(err).Msg("MCP SSE server error")
				}
			}()
			log.Info().Msgf("MCP server ready on port %d", serverMCPPort)
		}

		addr := fmt.Sprintf(":%d", serverPort)
		log.Info().Msgf("Starting FastHTTP server on %s", addr)
		if err := fasthttp.ListenAndServe(addr, wrappedHandler); err != nil {
			log.Error().Err(err).Msg("Error starting FastHTTP server")
			os.Exit(1)
		}
	},
}

// --- Helper functions for API router wiring ---
func requireJaegerNginxProxyAPI(k8sClient client.Client, ns string) *api.JaegerNginxProxyAPI {
	return &api.JaegerNginxProxyAPI{
		K8sClient: k8sClient,
		Namespace: ns,
	}
}

func requireFasthttprouter() *fasthttprouter.Router {
	// Import here to avoid import cycle in code edit
	return fasthttprouter.New()
}

func adaptHandler(h func(ctx *fasthttp.RequestCtx)) fasthttprouter.Handle {
	return func(ctx *fasthttp.RequestCtx, ps fasthttprouter.Params) {
		// Set URL parameters in the context so they can be accessed via ctx.UserValue
		for _, param := range ps {
			ctx.SetUserValue(param.Key, param.Value)
		}
		h(ctx)
	}
}

// --- End helper functions ---

// serveSwaggerJSON serves the generated swagger.json file
// @Summary Get Swagger JSON specification
// @Description Returns the OpenAPI/Swagger JSON specification for the API
// @Tags documentation
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Router /docs/swagger.json [get]
func serveSwaggerJSON(ctx *fasthttp.RequestCtx) {
	ctx.Response.Header.Set("Content-Type", "application/json")
	ctx.Response.Header.Set("Access-Control-Allow-Origin", "*")
	ctx.Response.Header.Set("Access-Control-Allow-Methods", "GET, OPTIONS")
	ctx.Response.Header.Set("Access-Control-Allow-Headers", "Content-Type")

	// Read the swagger.json file
	swaggerData, err := os.ReadFile("docs/swagger.json")
	if err != nil {
		ctx.SetStatusCode(fasthttp.StatusInternalServerError)
		ctx.SetBodyString(`{"error":"Failed to read swagger.json"}`)
		return
	}

	ctx.SetBody(swaggerData)
}

// serveSwaggerUI serves the Swagger UI HTML page
// @Summary Get Swagger UI
// @Description Returns the Swagger UI HTML page for API documentation
// @Tags documentation
// @Produce html
// @Success 200 {string} string "HTML page"
// @Router /swagger [get]
func serveSwaggerUI(ctx *fasthttp.RequestCtx) {
	ctx.Response.Header.Set("Content-Type", "text/html; charset=utf-8")

	// Read the swagger/index.html file
	swaggerHTML, err := os.ReadFile("swagger/index.html")
	if err != nil {
		ctx.SetStatusCode(fasthttp.StatusInternalServerError)
		ctx.SetBodyString(`<html><body><h1>Error</h1><p>Failed to read swagger UI</p></body></html>`)
		return
	}

	ctx.SetBody(swaggerHTML)
}

// serveMCPSSE serves the MCP SSE endpoint
// @Summary Get MCP SSE stream
// @Description Returns the Model Context Protocol Server-Sent Events stream for tool capabilities
// @Tags mcp
// @Produce text/event-stream
// @Success 200 {string} string "SSE stream"
// @Router /sse [get]
// NOTE: This function is only for Swagger documentation. The real /sse endpoint is served by the MCP server.
func serveMCPSSE(ctx *fasthttp.RequestCtx) {
	// This is handled by the MCP SSE server, not directly by FastHTTP
	ctx.SetStatusCode(fasthttp.StatusNotFound)
	ctx.SetBodyString(`{"error":"MCP SSE endpoint is served on a different port"}`)
}

// serveMCPMessage serves the MCP message endpoint
// @Summary Send MCP message
// @Description Send a JSON-RPC message to the MCP server for tool invocation
// @Tags mcp
// @Accept json
// @Produce json
// @Param body body map[string]interface{} true "JSON-RPC message"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]string
// @Router /message [post]
func serveMCPMessage(ctx *fasthttp.RequestCtx) {
	// This is handled by the MCP SSE server, not directly by FastHTTP
	ctx.SetStatusCode(fasthttp.StatusNotFound)
	ctx.SetBodyString(`{"error":"MCP message endpoint is served on a different port"}`)
}

// API endpoints:
//   GET    /api/jaegernginxproxies         - List all JaegerNginxProxy resources
//   GET    /api/jaegernginxproxies/:name   - Get a JaegerNginxProxy by name
//   POST   /api/jaegernginxproxies         - Create a JaegerNginxProxy
//   PUT    /api/jaegernginxproxies/:name   - Update a JaegerNginxProxy (full update)
//   PATCH  /api/jaegernginxproxies/:name   - Patch a JaegerNginxProxy (partial update)
//   DELETE /api/jaegernginxproxies/:name   - Delete a JaegerNginxProxy
//   GET    /deployments                    - List deployment names from informer cache
//   GET    /docs/swagger.json              - Get Swagger JSON specification
//   GET    /swagger                        - Get Swagger UI
//   GET    /sse                            - MCP SSE stream (port 9090)
//   POST   /message                        - MCP JSON-RPC messages (port 9090)

func init() {
	rootCmd.AddCommand(serverCmd)
	serverCmd.Flags().IntVar(&serverPort, "port", 8080, "Port to run the server on")
	serverCmd.Flags().IntVar(&serverMetricsPort, "metrics-port", 8081, "Port for controller manager metrics")
	serverCmd.Flags().BoolVar(&serverEnableLeaderElection, "enable-leader-election", true, "Enable leader election for controller manager")
	serverCmd.Flags().StringVar(&serverLeaderElectionNamespace, "leader-election-namespace", "default", "Namespace for leader election")
	serverCmd.Flags().StringVar(&serverKubeconfig, "kubeconfig", "", "Path to the kubeconfig file")
	serverCmd.Flags().BoolVar(&serverInCluster, "in-cluster", false, "Use in-cluster Kubernetes config")
	serverCmd.Flags().BoolVar(&serverEnableMCP, "enable-mcp", false, "Enable MCP server")
	serverCmd.Flags().IntVar(&serverMCPPort, "mcp-port", 9090, "Port for MCP server")
}
