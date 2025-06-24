package config

import (
	"os"

	"github.com/dolv/k8s-controller-tutorial/internal/utils"
	"github.com/rs/zerolog/log"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

func GetKubeConfig(kubeconfigPath string) (*rest.Config, error) {
	log.Debug().Msg("Obtain k8s clientset")
	path, err := utils.ExpandPath(kubeconfigPath)
	if err != nil {
		log.Error().Err(err).Msg("Failed to expand kubeconfig path")
		os.Exit(1)
	}
	log.Debug().Msgf("Resolved kubeconfig path: %s", path)
	config, err := clientcmd.BuildConfigFromFlags("", path)
	if err != nil {
		return nil, err
	}
	return config, nil
}
