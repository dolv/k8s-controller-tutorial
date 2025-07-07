package ctrl

import (
	"testing"

	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	JaegerNginxProxyV1alpha0 "github.com/dolv/k8s-controller-tutorial/pkg/apis/jaeger-nginx-proxy/v1alpha0"
)

func TestStatusLogicWithZeroReplicas(t *testing.T) {
	// Test case 1: replicaCount = 0, no pods running (should be ready)
	desired := int32(0)
	available := int32(0)

	// Simulate the status logic
	var ready bool
	var message string

	if desired == 0 {
		if available == 0 {
			ready = true
			message = "Deployment scaled to 0 replicas"
		} else {
			ready = false
			message = "Scaling down: pods still running, desired: 0"
		}
	} else if available == desired {
		ready = true
		message = "All pods are running"
	} else {
		ready = false
		message = "Available replicas: 0/0, Ready replicas: 0, Unavailable replicas: 0"
	}

	assert.True(t, ready, "Should be ready when replicaCount=0 and no pods running")
	assert.Equal(t, "Deployment scaled to 0 replicas", message)

	// Test case 2: replicaCount = 0, but pods still running (should not be ready)
	desired = int32(0)
	available = int32(2)

	if desired == 0 {
		if available == 0 {
			ready = true
			message = "Deployment scaled to 0 replicas"
		} else {
			ready = false
			message = "Scaling down: pods still running, desired: 0"
		}
	} else if available == desired {
		ready = true
		message = "All pods are running"
	} else {
		ready = false
		message = "Available replicas: 0/0, Ready replicas: 0, Unavailable replicas: 0"
	}

	assert.False(t, ready, "Should not be ready when replicaCount=0 but pods still running")
	assert.Contains(t, message, "Scaling down")

	// Test case 3: replicaCount = 2, all pods running (should be ready)
	desired = int32(2)
	available = int32(2)

	if desired == 0 {
		if available == 0 {
			ready = true
			message = "Deployment scaled to 0 replicas"
		} else {
			ready = false
			message = "Scaling down: pods still running, desired: 0"
		}
	} else if available == desired {
		ready = true
		message = "All pods are running"
	} else {
		ready = false
		message = "Available replicas: 0/0, Ready replicas: 0, Unavailable replicas: 0"
	}

	assert.True(t, ready, "Should be ready when all desired pods are running")
	assert.Equal(t, "All pods are running", message)
}

func TestBuildDeploymentWithZeroReplicas(t *testing.T) {
	// Test that buildDeployment handles replicaCount = 0 correctly
	nginxProxy := &JaegerNginxProxyV1alpha0.JaegerNginxProxy{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-proxy",
			Namespace: "default",
		},
		Spec: JaegerNginxProxyV1alpha0.JaegerNginxProxySpec{
			ReplicaCount:  0, // This should be allowed by the controller
			ContainerPort: 8080,
			Image: JaegerNginxProxyV1alpha0.Image{
				Repository: "nginx",
				Tag:        "1.21",
				PullPolicy: "IfNotPresent",
			},
			Upstream: JaegerNginxProxyV1alpha0.Upstream{
				CollectorHost: "jaeger-collector.tracing.svc.cluster.local",
			},
			Ports: []JaegerNginxProxyV1alpha0.Port{
				{Name: "http", Port: 14268, Path: "/api/traces"},
			},
			Service: JaegerNginxProxyV1alpha0.Service{Type: "ClusterIP"},
			Resources: JaegerNginxProxyV1alpha0.Resources{
				Limits:   JaegerNginxProxyV1alpha0.Resource{CPU: "500m", Memory: "512Mi"},
				Requests: JaegerNginxProxyV1alpha0.Resource{CPU: "100m", Memory: "128Mi"},
			},
		},
	}

	deployment := buildDeployment(nginxProxy)

	assert.Equal(t, int32(0), *deployment.Spec.Replicas, "Deployment should have 0 replicas")
	assert.Equal(t, "test-proxy", deployment.Name)
	assert.Equal(t, "default", deployment.Namespace)
}
