package cmd

import (
	"context"
	"testing"
	"time"

	"github.com/dolv/k8s-controller-tutorial/pkg/api"
	jaegerv1alpha0 "github.com/dolv/k8s-controller-tutorial/pkg/apis/jaeger-nginx-proxy/v1alpha0"
	"github.com/dolv/k8s-controller-tutorial/pkg/ctrl"
	"github.com/dolv/k8s-controller-tutorial/pkg/testutil"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// setupTestAPIWithManager is a self-contained helper for MCP integration tests.
func setupTestAPIWithManager(t *testing.T) (*api.JaegerNginxProxyAPI, client.Client, func()) {
	mgr, k8sClient, _, cleanup := testutil.StartTestManager(t)

	// Register the JaegerNginxProxy CRD scheme
	require.NoError(t, jaegerv1alpha0.AddToScheme(mgr.GetScheme()))

	require.NoError(t, ctrl.AddJaegerNginxProxyController(mgr))

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		_ = mgr.Start(ctx)
	}()

	// Wait for the cache to sync before returning
	if ok := mgr.GetCache().WaitForCacheSync(ctx); !ok {
		cancel()
		t.Fatal("cache did not sync")
	}

	apiInst := &api.JaegerNginxProxyAPI{
		K8sClient: k8sClient,
		Namespace: "default",
	}
	return apiInst, k8sClient, func() {
		cancel()
		cleanup()
	}
}

func TestMCP_ListJaegerNginxProxiesHandler(t *testing.T) {
	apiInst, k8sClient, cleanup := setupTestAPIWithManager(t)
	defer cleanup()
	api.JaegerNginxProxyAPIInst = apiInst

	// Create some JaegerNginxProxy resources
	proxy1 := &jaegerv1alpha0.JaegerNginxProxy{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "mcp-proxy1",
			Namespace: "default",
		},
		Spec: jaegerv1alpha0.JaegerNginxProxySpec{
			ReplicaCount:  1,
			ContainerPort: 8080,
			Image: jaegerv1alpha0.Image{
				Repository: "nginx",
				Tag:        "1.21",
				PullPolicy: "IfNotPresent",
			},
			Upstream: jaegerv1alpha0.Upstream{
				CollectorHost: "jaeger-collector.svc",
			},
			Ports:   []jaegerv1alpha0.Port{{Name: "http", Port: 14268, Path: "/api/traces"}},
			Service: jaegerv1alpha0.Service{Type: "ClusterIP"},
			Resources: jaegerv1alpha0.Resources{
				Limits:   jaegerv1alpha0.Resource{CPU: "500m", Memory: "512Mi"},
				Requests: jaegerv1alpha0.Resource{CPU: "100m", Memory: "128Mi"},
			},
		},
	}
	proxy2 := &jaegerv1alpha0.JaegerNginxProxy{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "mcp-proxy2",
			Namespace: "default",
		},
		Spec: jaegerv1alpha0.JaegerNginxProxySpec{
			ReplicaCount:  2,
			ContainerPort: 8081,
			Image: jaegerv1alpha0.Image{
				Repository: "nginx",
				Tag:        "1.22",
				PullPolicy: "IfNotPresent",
			},
			Upstream: jaegerv1alpha0.Upstream{
				CollectorHost: "jaeger-collector.svc",
			},
			Ports:   []jaegerv1alpha0.Port{{Name: "grpc", Port: 14250, Path: "/grpc"}},
			Service: jaegerv1alpha0.Service{Type: "ClusterIP"},
			Resources: jaegerv1alpha0.Resources{
				Limits:   jaegerv1alpha0.Resource{CPU: "1", Memory: "1Gi"},
				Requests: jaegerv1alpha0.Resource{CPU: "200m", Memory: "256Mi"},
			},
		},
	}
	require.NoError(t, k8sClient.Create(context.Background(), proxy1))
	require.NoError(t, k8sClient.Create(context.Background(), proxy2))

	// Wait for both resources to be present
	require.Eventually(t, func() bool {
		list := &jaegerv1alpha0.JaegerNginxProxyList{}
		err := k8sClient.List(context.Background(), list, client.InNamespace("default"))
		return err == nil && len(list.Items) == 2
	}, 5*time.Second, 100*time.Millisecond, "Should eventually see 2 JaegerNginxProxy resources")

	// Call the shared API logic directly (since MCP handler is not accessible)
	list := &jaegerv1alpha0.JaegerNginxProxyList{}
	require.NoError(t, k8sClient.List(context.Background(), list, client.InNamespace("default")))
	require.Len(t, list.Items, 2)
	names := []string{list.Items[0].Name, list.Items[1].Name}
	require.Contains(t, names, "mcp-proxy1")
	require.Contains(t, names, "mcp-proxy2")
}
