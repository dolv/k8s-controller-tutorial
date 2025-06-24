package cmd

import (
	"testing"

	cfgPkg "github.com/dolv/k8s-controller-tutorial/internal/config"
)

func TestGetKubeClient_InvalidPath(t *testing.T) {
	_, err := cfgPkg.GetKubeConfig("/invalid/path")
	if err == nil {
		t.Error("expected error for invalid kubeconfig path")
	}
}
