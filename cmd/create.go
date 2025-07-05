package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/dolv/k8s-controller-tutorial/internal/config"
	"github.com/dolv/k8s-controller-tutorial/internal/utils"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

var (
	replicas int32
	image    string
)

var createDeploymentCmd = &cobra.Command{
	Use:   "create <name>",
	Short: "Create Kubernetes deployment in the arbitrary namespace",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		name := args[0]
		kubeconfig, err := config.GetKubeConfig(kubeconfigPath)
		if err != nil {
			log.Error().Err(err).Msg("Failed to build kubeconfig rest object")
			os.Exit(1)
		}
		clientset, err := kubernetes.NewForConfig(kubeconfig)
		if err != nil {
			log.Error().Err(err).Msg("Failed to create Kubernetes client")
			os.Exit(1)
		}

		log.Debug().Msg("Creating deployment")
		dep := &appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: namespace,
			},
			Spec: appsv1.DeploymentSpec{
				Replicas: utils.Int32Ptr(replicas),
				Selector: &metav1.LabelSelector{
					MatchLabels: map[string]string{"app": name},
				},
				Template: corev1.PodTemplateSpec{
					ObjectMeta: metav1.ObjectMeta{Labels: map[string]string{"app": name}},
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{{Name: name, Image: image}},
					},
				},
			},
		}
		_, err = clientset.AppsV1().Deployments(namespace).Create(context.Background(), dep, metav1.CreateOptions{})
		if err != nil {
			log.Error().Err(err).Msg("Failed to create deployment")
			os.Exit(1)
		}
		fmt.Printf("Deployment '%s' created in namespace '%s'!\n", name, namespace)
	},
}

func init() {
	createDeploymentCmd.Flags().Int32VarP(&replicas, "replicas", "r", 1, "Number of replicas for the deployment")
	createDeploymentCmd.Flags().StringVarP(&image, "image", "i", "nginx", "Docker image for the deployment")
	rootCmd.AddCommand(createDeploymentCmd)
}
