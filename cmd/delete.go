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

var deleteCmd = &cobra.Command{
	Use:   "delete <name>",
	Short: "Delete a Kubernetes deployment in the provided namespace",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		name := args[0]
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

		log.Debug().Msgf("Deleting deployment '%s' in namespace '%s'", name, namespace)
		err = clientset.AppsV1().Deployments(namespace).Delete(context.Background(), name, metav1.DeleteOptions{})
		if err != nil {
			log.Error().Err(err).Msg("Failed to delete deployment")
			os.Exit(1)
		}
		fmt.Printf("Deployment '%s' deleted from namespace '%s'!\n", name, namespace)
	},
}

func init() {
	rootCmd.AddCommand(deleteCmd)
} 