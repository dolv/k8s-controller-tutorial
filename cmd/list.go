package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/dolv/k8s-controller-tutorial/internal/utils"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List Kubernetes deployments in the arbitrary namespace",
	Run: func(cmd *cobra.Command, args []string) {
		log.Debug().Msg("Obtain k8s clientset")
		path, err := utils.ExpandPath(kubeconfigPath)
		if err != nil {
			log.Error().Err(err).Msg("Failed to expand kubeconfig path")
			os.Exit(1)
		}
		log.Debug().Msgf("Resolved kubeconfig path: %s", path)
		clientset, err := getKubeClient(path)
		if err != nil {
			log.Error().Err(err).Msg("Failed to create Kubernetes client")
			os.Exit(1)
		}

		log.Debug().Msg("Listing resources")
		deployments, err := clientset.AppsV1().Deployments(namespace).List(context.Background(), metav1.ListOptions{})
		if err != nil {
			log.Error().Err(err).Msg("Failed to list deployments")
			os.Exit(1)
		}
		fmt.Printf("Found %d deployments in '%s' namespace:\n", len(deployments.Items), namespace)

		deploymentNames := []string{}
		for _, d := range deployments.Items {
			deploymentNames = append(deploymentNames, d.Name)

			fmt.Println("-", d.Name)
		}
		log.Debug().
			Strs("deployments", deploymentNames).
			Msgf("Found %d deployments in '%s' namespace.", len(deployments.Items), namespace)
	},
}

func getKubeClient(kubeconfigPath string) (*kubernetes.Clientset, error) {
	config, err := clientcmd.BuildConfigFromFlags("", kubeconfigPath)
	if err != nil {
		return nil, err
	}
	return kubernetes.NewForConfig(config)
}

func init() {
	rootCmd.AddCommand(listCmd)
}
