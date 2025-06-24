package cmd

import (
	"testing"

	"github.com/spf13/viper"
)

func TestServerCommandDefined(t *testing.T) {
	viper.SetConfigFile("config.yaml")
	_ = viper.ReadInConfig()
	if serverCmd == nil {
		t.Fatal("serverCmd should be defined")
	}
	if serverCmd.Use != "server" {
		t.Errorf("expected command use 'server', got %s", serverCmd.Use)
	}
	portFlag := serverCmd.Flags().Lookup("port")
	if portFlag == nil {
		t.Error("expected 'port' flag to be defined")
	}
}
