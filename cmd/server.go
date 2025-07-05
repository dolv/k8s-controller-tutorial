package cmd

import (
	"context"
	"fmt"
	"os"
	"time"

	cfgPkg "github.com/dolv/k8s-controller-tutorial/internal/config"
	"github.com/dolv/k8s-controller-tutorial/pkg/informer"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"github.com/valyala/fasthttp"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

var (
	serverPort       int
	serverKubeconfig string
	serverInCluster  bool
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
			os.Exit(1)
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
		go informer.StartDeploymentInformer(ctx, clientset, namespace)
		log.Trace().Msg("Getting handler instance")
		handler := func(ctx *fasthttp.RequestCtx) {
			logger, ok := ctx.UserValue(loggerKey).(zerolog.Logger)
			if !ok {
				logger = log.Logger
			}
			logger.Trace().Msg("Handler entered")
			switch string(ctx.Path()) {
			case "/deployments":
				logger.Info().Msg("Deployments request received")
				ctx.Response.Header.Set("Content-Type", "application/json")
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
				return
			default:
				logger.Info().Msg("Default request received")
				fmt.Fprintf(ctx, "Hello from FastHTTP! Your request ID: %s", ctx.UserValue(requestIDKey))
			}
			logger.Trace().Msg("Handler exiting")
		}
		log.Trace().Msg("Adding loggingMiddleware to handler instance")
		wrappedHandler := loggingMiddleware(handler)
		addr := fmt.Sprintf(":%d", serverPort)
		log.Info().Msgf("Starting FastHTTP server on %s", addr)
		if err := fasthttp.ListenAndServe(addr, wrappedHandler); err != nil {
			log.Error().Err(err).Msg("Error starting FastHTTP server")
			os.Exit(1)
		}
	},
}

func init() {
	rootCmd.AddCommand(serverCmd)
	serverCmd.Flags().IntVar(&serverPort, "port", 8080, "Port to run the server on")
	serverCmd.Flags().StringVar(&serverKubeconfig, "kubeconfig", "", "Path to the kubeconfig file")
	serverCmd.Flags().BoolVar(&serverInCluster, "in-cluster", false, "Use in-cluster Kubernetes config")
}
