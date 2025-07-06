//go:build testtools

package api

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"

	jaegerv1alpha0 "github.com/dolv/k8s-controller-tutorial/pkg/apis/jaeger-nginx-proxy/v1alpha0"
	myctrl "github.com/dolv/k8s-controller-tutorial/pkg/ctrl"
	"github.com/dolv/k8s-controller-tutorial/pkg/testutil"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/valyala/fasthttp"
	"github.com/valyala/fasthttprouter"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
)

// Helper: Convert JaegerNginxProxy to JaegerNginxProxyDoc
func toDoc(proxy *jaegerv1alpha0.JaegerNginxProxy) JaegerNginxProxyDoc {
	return JaegerNginxProxyDoc{
		Name:   proxy.Name,
		Spec:   proxy.Spec,
		Status: proxy.Status,
	}
}

// Helper: Convert list
func toDocList(list []jaegerv1alpha0.JaegerNginxProxy) []JaegerNginxProxyDoc {
	docs := make([]JaegerNginxProxyDoc, len(list))
	for i, proxy := range list {
		docs[i] = toDoc(&proxy)
	}
	return docs
}

// Adapter to use func(*fasthttp.RequestCtx) as fasthttprouter.Handle
func adaptHandler(h func(ctx *fasthttp.RequestCtx)) fasthttprouter.Handle {
	return func(ctx *fasthttp.RequestCtx, _ fasthttprouter.Params) {
		h(ctx)
	}
}

func setupTestAPIWithManager(t *testing.T) (*JaegerNginxProxyAPI, client.Client, func()) {
	mgr, k8sClient, _, cleanup := testutil.StartTestManager(t)

	require.NoError(t, myctrl.AddJaegerNginxProxyController(mgr))

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		_ = mgr.Start(ctx)
	}()

	// Wait for the cache to sync before returning
	if ok := mgr.GetCache().WaitForCacheSync(ctx); !ok {
		cancel()
		t.Fatal("cache did not sync")
	}

	api := &JaegerNginxProxyAPI{
		K8sClient: k8sClient,
		Namespace: "default",
	}
	return api, k8sClient, func() {
		cancel()
		cleanup()
	}
}

func doRequest(router *fasthttprouter.Router, method, uri string, body []byte) *fasthttp.Response {
	ctx := &fasthttp.RequestCtx{}
	req := &ctx.Request
	resp := &ctx.Response
	ctx.Init(req, nil, nil)
	req.Header.SetMethod(method)
	req.SetRequestURI(uri)
	if body != nil {
		req.SetBody(body)
	}
	// Manually set the user value for :name routes (GET, PUT, DELETE)
	if (method == http.MethodGet || method == http.MethodPut || method == http.MethodDelete) &&
		strings.HasPrefix(uri, "/api/jaegernginxproxies/") {
		parts := strings.Split(uri, "/")
		if len(parts) > 3 {
			ctx.SetUserValue("name", parts[3])
		}
	}
	router.Handler(ctx)
	return resp
}

// cleanupJaegerNginxProxies deletes all JaegerNginxProxy resources in the given namespace.
func cleanupJaegerNginxProxies(t *testing.T, c client.Client, ns string) {
	ctx := context.Background()
	var proxies jaegerv1alpha0.JaegerNginxProxyList
	require.NoError(t, c.List(ctx, &proxies, client.InNamespace(ns)))
	for _, p := range proxies.Items {
		require.NoError(t, c.Delete(ctx, &p))
	}
}

func getDeployment(t *testing.T, c client.Client, name, ns string, timeout time.Duration) *appsv1.Deployment {
	var dep appsv1.Deployment
	var lastErr error
	end := time.Now().Add(timeout)
	for time.Now().Before(end) {
		t.Logf("Checking for deployment %s/%s", ns, name)
		err := c.Get(context.Background(), client.ObjectKey{Name: name, Namespace: ns}, &dep)
		if err == nil {
			return &dep
		}
		lastErr = err
		time.Sleep(200 * time.Millisecond)
	}
	t.Fatalf("Deployment %s/%s not found after %v: %v", ns, name, timeout, lastErr)
	return nil
}

func TestJaegerNginxProxyAPI_E2E(t *testing.T) {
	log.SetLogger(zap.New(zap.UseDevMode(true)))

	id := uuid.NewString()[:8]
	resourceName := "test-jaeger-proxy-" + id

	api, k8sClient, cleanup := setupTestAPIWithManager(t)
	defer cleanup()

	cleanupJaegerNginxProxies(t, k8sClient, "default")

	router := fasthttprouter.New()
	router.GET("/api/jaegernginxproxies", adaptHandler(api.ListJaegerNginxProxies))
	router.GET("/api/jaegernginxproxies/:name", adaptHandler(api.GetJaegerNginxProxy))
	router.POST("/api/jaegernginxproxies", adaptHandler(api.CreateJaegerNginxProxy))
	router.PUT("/api/jaegernginxproxies/:name", adaptHandler(api.UpdateJaegerNginxProxy))
	router.DELETE("/api/jaegernginxproxies/:name", adaptHandler(api.DeleteJaegerNginxProxy))

	// --- Create ---
	t.Logf("[TEST] POST /api/jaegernginxproxies (name=%s)", resourceName)
	createSpec := jaegerv1alpha0.JaegerNginxProxySpec{
		ReplicaCount:  2,
		ContainerPort: 8080,
		Image: jaegerv1alpha0.Image{
			Repository: "nginx",
			Tag:        "1.21",
			PullPolicy: "IfNotPresent",
		},
		Upstream: jaegerv1alpha0.Upstream{
			CollectorHost: "jaeger-collector.tracing.svc.cluster.local",
		},
		Ports: []jaegerv1alpha0.Port{
			{Name: "http", Port: 14268, Path: "/api/traces"},
			{Name: "grpc", Port: 14250, Path: "/jaeger.api.v2.CollectorService/PostSpans"},
		},
		Service: jaegerv1alpha0.Service{Type: "ClusterIP"},
		Resources: jaegerv1alpha0.Resources{
			Limits:   jaegerv1alpha0.Resource{CPU: "500m", Memory: "512Mi"},
			Requests: jaegerv1alpha0.Resource{CPU: "100m", Memory: "128Mi"},
		},
	}
	createObj := &jaegerv1alpha0.JaegerNginxProxy{
		ObjectMeta: metav1.ObjectMeta{
			Name:      resourceName,
			Namespace: "default",
		},
		Spec:     createSpec,
		TypeMeta: jaegerv1alpha0.JaegerNginxProxy{}.TypeMeta,
	}
	body, _ := json.Marshal(createObj)
	resp := doRequest(router, http.MethodPost, "/api/jaegernginxproxies", body)
	t.Logf("Create response body: %s", resp.Body())
	require.Equal(t, http.StatusCreated, resp.StatusCode())

	// Wait for controller to create Deployment
	dep := getDeployment(t, k8sClient, resourceName, "default", 2*time.Second)
	t.Logf("Deployment after create: name=%s replicas=%v image=%s", dep.Name, *dep.Spec.Replicas, dep.Spec.Template.Spec.Containers[0].Image)

	// --- Update ---
	t.Logf("[TEST] PUT /api/jaegernginxproxies/%s", resourceName)
	// Fetch the existing JaegerNginxProxy to get its resourceVersion
	var existing jaegerv1alpha0.JaegerNginxProxy
	require.NoError(t, k8sClient.Get(context.Background(), client.ObjectKey{
		Name:      resourceName,
		Namespace: "default",
	}, &existing))

	updateSpec := createSpec
	updateSpec.ReplicaCount = 1
	updateObj := &jaegerv1alpha0.JaegerNginxProxy{
		ObjectMeta: metav1.ObjectMeta{
			Name:            resourceName,
			Namespace:       "default",
			ResourceVersion: existing.ResourceVersion,
		},
		Spec:     updateSpec,
		TypeMeta: jaegerv1alpha0.JaegerNginxProxy{}.TypeMeta,
	}
	body, _ = json.Marshal(updateObj)
	resp = doRequest(router, http.MethodPut, "/api/jaegernginxproxies/"+resourceName, body)
	t.Logf("Update response body: %s", resp.Body())
	require.Equal(t, http.StatusOK, resp.StatusCode())

	// Wait for controller to update Deployment
	dep = getDeployment(t, k8sClient, resourceName, "default", 5*time.Second)
	t.Logf("Deployment after update: name=%s replicas=%v image=%s", dep.Name, *dep.Spec.Replicas, dep.Spec.Template.Spec.Containers[0].Image)

	// --- Delete ---
	t.Logf("[TEST] DELETE /api/jaegernginxproxies/%s", resourceName)
	resp = doRequest(router, http.MethodDelete, "/api/jaegernginxproxies/"+resourceName, nil)
	require.Equal(t, http.StatusNoContent, resp.StatusCode())

	// Wait for Deployment to be deleted
	end := time.Now().Add(2 * time.Second)
	for time.Now().Before(end) {
		var dep appsv1.Deployment
		err := k8sClient.Get(context.Background(), client.ObjectKey{Name: resourceName, Namespace: "default"}, &dep)
		if err != nil {
			t.Logf("Deployment deleted as expected")
			break
		}
		time.Sleep(100 * time.Millisecond)
	}
}
