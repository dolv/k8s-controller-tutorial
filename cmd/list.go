package cmd

import (
	"fmt"
	"context"
	"os"
	"github.com/spf13/cobra"
	"github.com/rs/zerolog/log"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List all resources",
	Long:  `List all resources in the Kubernetes cluster`,
	Run: func(cmd *cobra.Command, args []string) {
		log.debug().Msg("Listing resources")
		clientset, err := getKubeClient(kubeconfig)
		if err != nil {
			log.Error().Err(err).Msg("Failed to create Kubernetes client")
			os.Exit(1)
		}
		deployments, err := clientset.AppsV1().Deployments(namespace).List(context.Background(), metav1.ListOptions{})
		if err != nil {
			log.Error().Err(err).Msg("Failed to list deployments")
			os.Exit(1)
		}
		log.Info().Msgf("Found %d deployments in '%s' namespace:\n", len(deployments.Items), namespace)
		for _, d := range deployments.Items {
			log.Info().Msg("-", d.Name)
		}
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
	listCmd.Flags().StringVar(&kubeconfig, "kubeconfig", "", "Path to the kubeconfig file")
}