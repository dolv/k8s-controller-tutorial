package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/dolv/k8s-controller-tutorial/internal/config"
)

var (
	logLevel       string
	namespace      string
	kubeconfigPath string
	appVersion     = "dev"
)

var rootCmd = &cobra.Command{
	Use:   "k8s-controller-tutorial",
	Short: "A brief description of your application (version: " + appVersion + ")",
	Long: `A longer description that spans multiple lines and likely contains
examples and usage of using your application. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.
`,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		configureLogger(config.ResolveLogLevel(cmd))
		log.Info().
			Msgf("Logger initialized with effective log-level=%s", viper.GetString("log-level"))
	},
	Run: func(cmd *cobra.Command, args []string) {
		level := parseLogLevel(logLevel)
		configureLogger(level)
		log.Info().Msg("This is an info log")
		log.Debug().Msg("This is a debug log")
		log.Trace().Msg("This is a trace log")
		log.Warn().Msg("This is a warn log")
		log.Error().Msg("This is an error log")
		fmt.Println("Welcome to k8s-controller-tutorial CLI!")
	},
}

func parseLogLevel(lvl string) zerolog.Level {
	switch strings.ToLower(lvl) {
	case "trace":
		return zerolog.TraceLevel
	case "debug":
		return zerolog.DebugLevel
	case "info":
		return zerolog.InfoLevel
	case "warn":
		return zerolog.WarnLevel
	case "error":
		return zerolog.ErrorLevel
	default:
		return zerolog.InfoLevel
	}
}

func configureLogger(level zerolog.Level) {
	fmt.Println("Setting log level to:", level)
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnixMs
	zerolog.SetGlobalLevel(level)
	switch level {
	case zerolog.TraceLevel:
		zerolog.CallerMarshalFunc = func(pc uintptr, file string, line int) string {
			return fmt.Sprintf("%s:%d", file, line)
		}
		zerolog.CallerFieldName = "caller"
		log.Logger = log.Output(zerolog.ConsoleWriter{
			Out:        os.Stderr,
			TimeFormat: "2006-01-02 15:04:05.000",
			PartsOrder: []string{
				zerolog.TimestampFieldName,
				zerolog.LevelFieldName,
				zerolog.CallerFieldName,
				zerolog.MessageFieldName,
			},
		}).With().Caller().Logger()
	case zerolog.DebugLevel:
		log.Logger = log.Output(zerolog.ConsoleWriter{
			Out:        os.Stderr,
			TimeFormat: "2006-01-02 15:04:05.000",
			PartsOrder: []string{
				zerolog.TimestampFieldName,
				zerolog.LevelFieldName,
				zerolog.MessageFieldName,
			},
		})
	default:
		log.Logger = log.Output(os.Stderr)
	}
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	viper.SetEnvPrefix("K8SCTRL")
	viper.AutomaticEnv()
	viper.SetDefault("log-level", "info")

	rootCmd.PersistentFlags().StringVar(&logLevel, "log-level", "info", "Set log level: trace, debug, info, warn, error")
	rootCmd.PersistentFlags().StringVarP(&namespace, "namespace", "n", "default", "Kubernetes namespace to use")
	rootCmd.PersistentFlags().StringVarP(&kubeconfigPath, "kubeconfig", "k", "~/.kube/config", "Path to the kubeconfig file")

	// Config file (config.yaml in cwd)
	viper.SetConfigName("config")
	viper.AddConfigPath(".")
	_ = viper.ReadInConfig()
}
