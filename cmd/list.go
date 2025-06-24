package cmd

import (
	"context"
	"fmt"
	"os"

	cfgPkg "github.com/dolv/k8s-controller-tutorial/internal/config"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List Kubernetes deployments in the arbitrary namespace",
	Run: func(cmd *cobra.Command, args []string) {
		kubeconfig, err := cfgPkg.GetKubeConfig(kubeconfigPath)
		if err != nil {
			log.Error().Err(err).Msg("Failed to build kubeconfig rest object")
			os.Exit(1)
		}
		clientset, err := kubernetes.NewForConfig(kubeconfig)
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

func init() {
	rootCmd.AddCommand(listCmd)
}
