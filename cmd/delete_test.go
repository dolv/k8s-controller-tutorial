package cmd

import (
	"context"
	"testing"

	"github.com/dolv/k8s-controller-tutorial/pkg/testutil"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"github.com/dolv/k8s-controller-tutorial/internal/utils"
)

// resetDeleteCmd resets the delete command to its initial state
func resetDeleteCmd() {
	deleteCmd.ResetFlags()
	deleteCmd.ResetCommands()
	deleteCmd.SetArgs([]string{})
}

func TestDeleteDeploymentCmd_Integration(t *testing.T) {
	resetDeleteCmd()
	viper.SetConfigFile("config.yaml")
	_ = viper.ReadInConfig()
	_, clientset, cleanup := testutil.SetupEnv(t)
	defer cleanup()

	namespace = "default"
	kubeconfigPath = "/tmp/envtest.kubeconfig"
	name := "delete-me"

	// Create a real deployment to delete
	dep := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: utils.Int32Ptr(1),
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{"app": name},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{Labels: map[string]string{"app": name}},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{{Name: name, Image: "nginx"}},
				},
			},
		},
	}
	_, err := clientset.AppsV1().Deployments(namespace).Create(context.Background(), dep, metav1.CreateOptions{})
	if err != nil {
		t.Fatalf("failed to create deployment: %v", err)
	}

	// Delete the deployment using the CLI command
	deleteCmd.SetArgs([]string{name})
	err = deleteCmd.Execute()
	require.NoError(t, err)

	// Verify the deployment is gone
	_, err = clientset.AppsV1().Deployments(namespace).Get(context.Background(), name, metav1.GetOptions{})
	require.Error(t, err)
} 