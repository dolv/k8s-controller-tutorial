package testutil

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"

	"github.com/dolv/k8s-controller-tutorial/internal/utils"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

// findProjectRoot recursively searches upwards from the current directory for 'go.mod' and returns its absolute path.
func findProjectRoot() (string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	dir := cwd
	for {
		candidate := filepath.Join(dir, "go.mod")
		if stat, err := os.Stat(candidate); err == nil && !stat.IsDir() {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break // reached root
		}
		dir = parent
	}
	return "", fmt.Errorf("go.mod not found in any parent directory of %s", cwd)
}

// findCRDDir returns the absolute path to config/crd inside the project root.
func findCRDDir() (string, error) {
	projectRoot, err := findProjectRoot()
	if err != nil {
		return "", err
	}
	crdPath := filepath.Join(projectRoot, "config", "crd")
	if stat, err := os.Stat(crdPath); err == nil && stat.IsDir() {
		return crdPath, nil
	}
	return "", fmt.Errorf("config/crd directory not found in project root %s", projectRoot)
}

// StartTestManager sets up envtest, scheme, manager, and returns them with cleanup.
func StartTestManager(t *testing.T) (mgr manager.Manager, k8sClient client.Client, restCfg *rest.Config, cleanup func()) {
	t.Helper()
	testScheme := runtime.NewScheme()

	// Add the core Kubernetes schemes
	require.NoError(t, scheme.AddToScheme(testScheme))
	require.NoError(t, apiextensionsv1.AddToScheme(testScheme))

	crdAbsPath, errCRD := findCRDDir()
	if errCRD != nil {
		t.Fatalf("[envtest] %v", errCRD)
	}
	fmt.Printf("[envtest] Using CRD directory: %s\n", crdAbsPath)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	env := &envtest.Environment{
		ErrorIfCRDPathMissing:    true,
		AttachControlPlaneOutput: false,
		CRDDirectoryPaths:        []string{crdAbsPath},
	}
	startErr := make(chan error)
	var cfg *rest.Config
	var errEnv error

	go func() {
		cfg, errEnv = env.Start()
		startErr <- errEnv
	}()

	// Wait for environment to start with timeout
	select {
	case errEnv := <-startErr:
		require.NoError(t, errEnv, "Failed to start test environment")
	case <-ctx.Done():
		t.Fatal("Timeout waiting for test environment to start")
	}

	require.NotNil(t, cfg)

	mgr, errEnv = manager.New(cfg, manager.Options{Scheme: testScheme, LeaderElection: false})
	require.NoError(t, errEnv)

	ctx, cancel = context.WithCancel(context.Background())
	go func() {
		_ = mgr.Start(ctx)
	}()

	k8sClient = mgr.GetClient()

	cleanup = func() {
		cancel()
		_ = env.Stop()
	}
	return mgr, k8sClient, cfg, cleanup
}

// SetupEnv starts envtest, creates a clientset, populates the cluster with sample Deployments, and returns env, clientset, and cleanup.
func SetupEnv(t *testing.T) (*envtest.Environment, *kubernetes.Clientset, func()) {
	t.Helper()
	ctx := context.Background()
	env := &envtest.Environment{}

	cfg, err := env.Start()
	require.NoError(t, err)
	require.NotNil(t, cfg)

	// Write kubeconfig to /tmp/envtest.kubeconfig
	kubeconfig := clientcmdapi.NewConfig()
	kubeconfig.Clusters["envtest"] = &clientcmdapi.Cluster{
		Server:                   cfg.Host,
		CertificateAuthorityData: cfg.CAData,
	}
	kubeconfig.AuthInfos["envtest-user"] = &clientcmdapi.AuthInfo{
		ClientCertificateData: cfg.CertData,
		ClientKeyData:         cfg.KeyData,
	}
	kubeconfig.Contexts["envtest-context"] = &clientcmdapi.Context{
		Cluster:  "envtest",
		AuthInfo: "envtest-user",
	}
	kubeconfig.CurrentContext = "envtest-context"

	kubeconfigBytes, err := clientcmd.Write(*kubeconfig)
	require.NoError(t, err)
	err = os.WriteFile("/tmp/envtest.kubeconfig", kubeconfigBytes, 0o644)
	require.NoError(t, err)

	clientset, err := kubernetes.NewForConfig(cfg)
	require.NoError(t, err)

	// Create sample Deployments
	for i := 1; i <= 2; i++ {
		dep := &appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:      fmt.Sprintf("sample-deployment-%d", i),
				Namespace: "default",
			},
			Spec: appsv1.DeploymentSpec{
				Replicas: utils.Int32Ptr(1),
				Selector: &metav1.LabelSelector{
					MatchLabels: map[string]string{"app": "test"},
				},
				Template: corev1.PodTemplateSpec{
					ObjectMeta: metav1.ObjectMeta{Labels: map[string]string{"app": "test"}},
					Spec:       corev1.PodSpec{Containers: []corev1.Container{{Name: "nginx", Image: "nginx"}}},
				},
			},
		}
		_, err := clientset.AppsV1().Deployments("default").Create(ctx, dep, metav1.CreateOptions{})
		require.NoError(t, err)
	}

	cleanup := func() {
		_ = env.Stop()
	}
	return env, clientset, cleanup
}
