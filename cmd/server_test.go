package cmd

import (
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/valyala/fasthttp"
	"github.com/valyala/fasthttprouter"
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

func TestGetServerKubeClient_InvalidPath(t *testing.T) {
	_, err := getServerKubeClient("/invalid/path", false)
	if err == nil {
		t.Error("expected error for invalid kubeconfig path")
	}
}

func TestAdaptHandlerParameterExtraction(t *testing.T) {
	// Create a test handler that checks if the name parameter is set
	var capturedName string
	testHandler := func(ctx *fasthttp.RequestCtx) {
		nameVal := ctx.UserValue("name")
		if nameVal != nil {
			capturedName = nameVal.(string)
		}
	}

	// Create the adapted handler
	adaptedHandler := adaptHandler(testHandler)

	// Create a mock request context
	ctx := &fasthttp.RequestCtx{}

	// Create mock parameters
	params := fasthttprouter.Params{
		{Key: "name", Value: "test-proxy"},
	}

	// Call the adapted handler
	adaptedHandler(ctx, params)

	// Verify the parameter was extracted correctly
	assert.Equal(t, "test-proxy", capturedName, "Parameter should be extracted and set in context")
}

func TestAdaptHandlerWithoutParameters(t *testing.T) {
	// Create a test handler that checks if the name parameter is set
	var capturedName string
	testHandler := func(ctx *fasthttp.RequestCtx) {
		nameVal := ctx.UserValue("name")
		if nameVal != nil {
			capturedName = nameVal.(string)
		}
	}

	// Create the adapted handler
	adaptedHandler := adaptHandler(testHandler)

	// Create a mock request context
	ctx := &fasthttp.RequestCtx{}

	// Create empty parameters (simulating a route without parameters)
	params := fasthttprouter.Params{}

	// Call the adapted handler
	adaptedHandler(ctx, params)

	// Verify no parameter was captured (as expected)
	assert.Equal(t, "", capturedName, "No parameter should be captured when params are empty")
}
