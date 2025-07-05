package cmd

import (
	"context"
	"fmt"
	"testing"

	"github.com/dolv/k8s-controller-tutorial/pkg/testutil"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// resetCreateDeploymentCmd resets the command to its initial state
func resetCreateDeploymentCmd() {
	createDeploymentCmd.ResetFlags()
	createDeploymentCmd.ResetCommands()
	createDeploymentCmd.SetArgs([]string{})
	// Re-add the flags that were set in init()
	createDeploymentCmd.Flags().Int32VarP(&replicas, "replicas", "r", 1, "Number of replicas for the deployment")
	createDeploymentCmd.Flags().StringVarP(&image, "image", "i", "nginx", "Docker image for the deployment")
}

func TestCreateDeploymentCmd_Integration(t *testing.T) {
	resetCreateDeploymentCmd()
	
	viper.SetConfigFile("config.yaml")
	_ = viper.ReadInConfig()
	_, clientset, cleanup := testutil.SetupEnv(t)
	defer cleanup()

	namespace = "default"
	kubeconfigPath = "/tmp/envtest.kubeconfig"
	name := "my-deployment"
	createDeploymentCmd.SetArgs([]string{
		name,
		"--kubeconfig", kubeconfigPath,
		"--namespace", namespace,
		"--replicas", fmt.Sprint(replicas),
		"--image", image,
	})

	// Run the command
	err := createDeploymentCmd.Execute()
	require.NoError(t, err)

	// Verify the deployment was created
	dep, err := clientset.AppsV1().Deployments(namespace).Get(context.Background(), name, metav1.GetOptions{})
	require.NoError(t, err)
	require.Equal(t, name, dep.Name)
	require.Equal(t, namespace, dep.Namespace)
	require.Equal(t, int32(1), *dep.Spec.Replicas)
	require.Equal(t, "nginx", dep.Spec.Template.Spec.Containers[0].Image)
}

func TestCreateDeploymentCmd_MissingArgs(t *testing.T) {
	resetCreateDeploymentCmd()
	
	createDeploymentCmd.SetArgs([]string{})
	err := createDeploymentCmd.Execute()
	require.Error(t, err)
}
