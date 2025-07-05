package cmd

import (
	"context"
	"fmt"
	"os"
	"time"

	cfgPkg "github.com/dolv/k8s-controller-tutorial/internal/config"
	"github.com/dolv/k8s-controller-tutorial/pkg/ctrl"
	"github.com/dolv/k8s-controller-tutorial/pkg/informer"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"github.com/valyala/fasthttp"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	ctrlruntime "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/metrics/server"
	ctrlruntimelog "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
)

var (
	serverPort       int
	serverMetricsPort int
	serverKubeconfig string
	serverInCluster  bool
	serverEnableLeaderElection bool
	serverLeaderElectionNamespace string
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
		go informer.StartDeploymentInformer(ctx, clientset, namespace)

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
				LeaderElectionID:        "k8s-controllers-leader-election",
				LeaderElectionNamespace: serverLeaderElectionNamespace,
				Metrics:                 server.Options{BindAddress: fmt.Sprintf(":%d", serverMetricsPort)},
			},
		)
		if err != nil {
			log.Error().Err(err).Msg("Failed to create controller-runtime manager")
			os.Exit(1)
		}
		if err := ctrl.AddDeploymentController(mgr); err != nil {
			log.Error().Err(err).Msg("Failed to add deployment controller")
			os.Exit(1)
		}
		go func() {
			log.Info().Msg("Starting controller-runtime manager...")
			if err := mgr.Start(cmd.Context()); err != nil {
				log.Error().Err(err).Msg("Manager exited with error")
				os.Exit(1)
			}
		}()

		// Set up controller-runtime logging
		ctrlruntimelog.SetLogger(zap.New(zap.UseDevMode(true)))

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
	serverCmd.Flags().IntVar(&serverMetricsPort, "metrics-port", 8081, "Port for controller manager metrics")
	serverCmd.Flags().BoolVar(&serverEnableLeaderElection, "enable-leader-election", true, "Enable leader election for controller manager")
	serverCmd.Flags().StringVar(&serverLeaderElectionNamespace, "leader-election-namespace", "default", "Namespace for leader election")
	serverCmd.Flags().StringVar(&serverKubeconfig, "kubeconfig", "", "Path to the kubeconfig file")
	serverCmd.Flags().BoolVar(&serverInCluster, "in-cluster", false, "Use in-cluster Kubernetes config")
}
