package config

import (
	"os"
	"testing"

	"github.com/rs/zerolog"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
)

// Helper to create a test Cobra command with log-level flag
func newTestCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "test"}
	cmd.Flags().String("log-level", "", "log level")
	return cmd
}

// Helper to reset viper and env
func cleanup() {
	_ = os.Unsetenv("K8SCTRL_LOG_LEVEL")
	viper.Set("log-level", "")
}

func TestResolveLogLevel_Default(t *testing.T) {
	cleanup()
	cmd := newTestCmd()
	got := ResolveLogLevel(cmd)
	assert.Equal(t, zerolog.InfoLevel, got)
	assert.Equal(t, "info", viper.GetString("log-level"))
}

func TestResolveLogLevel_ConfigOnly(t *testing.T) {
	cleanup()
	cmd := newTestCmd()
	viper.Set("log-level", "warn")
	got := ResolveLogLevel(cmd)
	assert.Equal(t, zerolog.WarnLevel, got)
	assert.Equal(t, "warn", viper.GetString("log-level"))
}

func TestResolveLogLevel_EnvOnly(t *testing.T) {
	cleanup()
	cmd := newTestCmd()
	os.Setenv("K8SCTRL_LOG_LEVEL", "debug")
	got := ResolveLogLevel(cmd)
	assert.Equal(t, zerolog.DebugLevel, got)
	assert.Equal(t, "debug", viper.GetString("log-level"))
}

func TestResolveLogLevel_FlagOnly(t *testing.T) {
	cleanup()
	cmd := newTestCmd()
	cmd.Flags().Set("log-level", "trace")
	got := ResolveLogLevel(cmd)
	assert.Equal(t, zerolog.TraceLevel, got)
	assert.Equal(t, "trace", viper.GetString("log-level"))
}

func TestResolveLogLevel_EnvAndConfig(t *testing.T) {
	cleanup()
	cmd := newTestCmd()
	viper.Set("log-level", "warn")
	os.Setenv("K8SCTRL_LOG_LEVEL", "error")
	got := ResolveLogLevel(cmd)
	assert.Equal(t, zerolog.ErrorLevel, got)
	assert.Equal(t, "error", viper.GetString("log-level"))
}

func TestResolveLogLevel_FlagAndEnv(t *testing.T) {
	cleanup()
	cmd := newTestCmd()
	os.Setenv("K8SCTRL_LOG_LEVEL", "debug")
	cmd.Flags().Set("log-level", "warn")
	got := ResolveLogLevel(cmd)
	assert.Equal(t, zerolog.WarnLevel, got)
	assert.Equal(t, "warn", viper.GetString("log-level"))
}

func TestResolveLogLevel_FlagEnvConfig(t *testing.T) {
	cleanup()
	cmd := newTestCmd()
	viper.Set("log-level", "warn")
	os.Setenv("K8SCTRL_LOG_LEVEL", "debug")
	cmd.Flags().Set("log-level", "error")
	got := ResolveLogLevel(cmd)
	assert.Equal(t, zerolog.ErrorLevel, got)
	assert.Equal(t, "error", viper.GetString("log-level"))
}

func TestResolveLogLevel_UnknownLevelFallback(t *testing.T) {
	cleanup()
	cmd := newTestCmd()
	cmd.Flags().Set("log-level", "notalevel")
	got := ResolveLogLevel(cmd)
	assert.Equal(t, zerolog.InfoLevel, got)
	assert.Equal(t, "notalevel", viper.GetString("log-level"))
}

func TestResolveLogLevel_CaseInsensitive(t *testing.T) {
	cleanup()
	cmd := newTestCmd()
	cmd.Flags().Set("log-level", "ERROR")
	got := ResolveLogLevel(cmd)
	assert.Equal(t, zerolog.ErrorLevel, got)
	assert.Equal(t, "ERROR", viper.GetString("log-level"))
}
