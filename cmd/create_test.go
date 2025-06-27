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

func TestCreateDeploymentCmd_Integration(t *testing.T) {
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
	createDeploymentCmd.SetArgs([]string{})
	err := createDeploymentCmd.Execute()
	require.Error(t, err)
}
