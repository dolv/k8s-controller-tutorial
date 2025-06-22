package config

import (
	"os"
	"strings"

	"github.com/rs/zerolog"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func ResolveLogLevel(cmd *cobra.Command) zerolog.Level {
	// 1. Flag wins if set
	flagChanged := cmd.Flags().Changed("log-level")
	flagVal, _ := cmd.Flags().GetString("log-level")
	// 2. Env var (if flag not set)
	envVal := os.Getenv("K8SCTRL_LOG_LEVEL")
	// 3. Config file (if neither flag nor env)
	configVal := viper.GetString("log-level")

	var effectiveLogLevel string
	switch {
	case flagChanged && flagVal != "":
		effectiveLogLevel = flagVal
	case envVal != "":
		effectiveLogLevel = envVal
	case configVal != "":
		effectiveLogLevel = configVal
	default:
		effectiveLogLevel = "info"
	}
	viper.Set("log-level", effectiveLogLevel) // for global access
	switch strings.ToLower(effectiveLogLevel) {
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
