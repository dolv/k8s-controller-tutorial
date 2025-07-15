package api

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/valyala/fasthttp"
	"sigs.k8s.io/controller-runtime/pkg/client"

	jaegerv1alpha0 "github.com/dolv/k8s-controller-tutorial/pkg/apis/jaeger-nginx-proxy/v1alpha0"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// JaegerNginxProxyAPI provides handlers for JaegerNginxProxy resources.
type JaegerNginxProxyAPI struct {
	K8sClient client.Client
	Namespace string // default namespace for simplicity
}

var JaegerNginxProxyAPIInst *JaegerNginxProxyAPI

// JaegerNginxProxyDoc is a simplified version for API docs
// @Description JaegerNginxProxy resource (Swagger only)
type JaegerNginxProxyDoc struct {
	Name   string                                `json:"name" example:"example-proxy"`
	Spec   jaegerv1alpha0.JaegerNginxProxySpec   `json:"spec"`
	Status jaegerv1alpha0.JaegerNginxProxyStatus `json:"status,omitempty"`
}

type JaegerNginxProxyListDoc struct {
	Items []JaegerNginxProxyDoc `json:"items"`
}

// ListJaegerNginxProxies godoc
// @Summary List all JaegerNginxProxies
// @Description Get all JaegerNginxProxy resources
// @Tags jaegernginxproxies
// @Produce json
// @Success 200 {object} JaegerNginxProxyListDoc
// @Router /api/jaegernginxproxies [get]
func (api *JaegerNginxProxyAPI) ListJaegerNginxProxies(ctx *fasthttp.RequestCtx) {
	list := &jaegerv1alpha0.JaegerNginxProxyList{}
	err := api.K8sClient.List(context.Background(), list, client.InNamespace(api.Namespace))
	if err != nil {
		ctx.SetStatusCode(fasthttp.StatusInternalServerError)
		ctx.SetBodyString(fmt.Sprintf(`{"error":"%v"}`, err))
		return
	}
	ctx.SetContentType("application/json")
	json.NewEncoder(ctx).Encode(list.Items)
}

// GetJaegerNginxProxy godoc
// @Summary Get a JaegerNginxProxy
// @Description Get a JaegerNginxProxy by name
// @Tags jaegernginxproxies
// @Produce json
// @Param name path string true "JaegerNginxProxy name"
// @Success 200 {object} JaegerNginxProxyDoc
// @Failure 404 {object} map[string]string
// @Router /api/jaegernginxproxies/{name} [get]
func (api *JaegerNginxProxyAPI) GetJaegerNginxProxy(ctx *fasthttp.RequestCtx) {
	nameVal := ctx.UserValue("name")
	if nameVal == nil {
		ctx.SetStatusCode(fasthttp.StatusBadRequest)
		ctx.SetBodyString(`{"error":"missing name parameter"}`)
		return
	}
	name := nameVal.(string)
	obj := &jaegerv1alpha0.JaegerNginxProxy{}
	err := api.K8sClient.Get(context.Background(), client.ObjectKey{Namespace: api.Namespace, Name: name}, obj)
	if err != nil {
		ctx.SetStatusCode(fasthttp.StatusNotFound)
		ctx.SetBodyString(fmt.Sprintf(`{"error":"%v"}`, err))
		return
	}
	ctx.SetContentType("application/json")
	json.NewEncoder(ctx).Encode(obj)
}

// CreateJaegerNginxProxy godoc
// @Summary Create a JaegerNginxProxy
// @Description Create a new JaegerNginxProxy
// @Tags jaegernginxproxies
// @Accept json
// @Produce json
// @Param body body JaegerNginxProxyDoc true "JaegerNginxProxy object"
// @Success 201 {object} JaegerNginxProxyDoc
// @Failure 400 {object} map[string]string
// @Router /api/jaegernginxproxies [post]
func (api *JaegerNginxProxyAPI) CreateJaegerNginxProxy(ctx *fasthttp.RequestCtx) {
	obj := &jaegerv1alpha0.JaegerNginxProxy{}
	if err := json.Unmarshal(ctx.PostBody(), obj); err != nil {
		ctx.SetStatusCode(fasthttp.StatusBadRequest)
		ctx.SetBodyString(fmt.Sprintf(`{"error":"%v"}`, err))
		return
	}
	// Set namespace before creation
	obj.Namespace = api.Namespace
	if err := api.K8sClient.Create(context.Background(), obj); err != nil {
		ctx.SetStatusCode(fasthttp.StatusInternalServerError)
		ctx.SetBodyString(fmt.Sprintf(`{"error":"%v"}`, err))
		return
	}
	ctx.SetStatusCode(fasthttp.StatusCreated)
	ctx.SetContentType("application/json")
	json.NewEncoder(ctx).Encode(obj)
}

// UpdateJaegerNginxProxy godoc
// @Summary Update a JaegerNginxProxy (full update)
// @Description Update an existing JaegerNginxProxy with full spec replacement
// @Tags jaegernginxproxies
// @Accept json
// @Produce json
// @Param name path string true "JaegerNginxProxy name"
// @Param body body JaegerNginxProxyDoc true "JaegerNginxProxy object"
// @Success 200 {object} JaegerNginxProxyDoc
// @Failure 400 {object} map[string]string
// @Router /api/jaegernginxproxies/{name} [put]
func (api *JaegerNginxProxyAPI) UpdateJaegerNginxProxy(ctx *fasthttp.RequestCtx) {
	nameVal := ctx.UserValue("name")
	if nameVal == nil {
		ctx.SetStatusCode(fasthttp.StatusBadRequest)
		ctx.SetBodyString(`{"error":"missing name parameter"}`)
		return
	}
	name := nameVal.(string)

	// Fetch the existing object to get the current resourceVersion
	existing := &jaegerv1alpha0.JaegerNginxProxy{}
	err := api.K8sClient.Get(context.Background(), client.ObjectKey{Namespace: api.Namespace, Name: name}, existing)
	if err != nil {
		ctx.SetStatusCode(fasthttp.StatusNotFound)
		ctx.SetBodyString(fmt.Sprintf(`{"error":"%v"}`, err))
		return
	}

	// Unmarshal the new spec and update only the Spec fields
	var patch struct {
		Spec jaegerv1alpha0.JaegerNginxProxySpec `json:"spec"`
	}
	if err := json.Unmarshal(ctx.PostBody(), &patch); err != nil {
		ctx.SetStatusCode(fasthttp.StatusBadRequest)
		ctx.SetBodyString(fmt.Sprintf(`{"error":"%v"}`, err))
		return
	}
	existing.Spec = patch.Spec

	if err := api.K8sClient.Update(context.Background(), existing); err != nil {
		ctx.SetStatusCode(fasthttp.StatusInternalServerError)
		ctx.SetBodyString(fmt.Sprintf(`{"error":"%v"}`, err))
		return
	}
	ctx.SetContentType("application/json")
	json.NewEncoder(ctx).Encode(existing)
}

// PatchJaegerNginxProxy godoc
// @Summary Patch a JaegerNginxProxy (partial update)
// @Description Update specific fields of an existing JaegerNginxProxy
// @Tags jaegernginxproxies
// @Accept json
// @Produce json
// @Param name path string true "JaegerNginxProxy name"
// @Param body body map[string]interface{} true "Partial update object"
// @Success 200 {object} JaegerNginxProxyDoc
// @Failure 400 {object} map[string]string
// @Router /api/jaegernginxproxies/{name} [patch]
func (api *JaegerNginxProxyAPI) PatchJaegerNginxProxy(ctx *fasthttp.RequestCtx) {
	nameVal := ctx.UserValue("name")
	if nameVal == nil {
		ctx.SetStatusCode(fasthttp.StatusBadRequest)
		ctx.SetBodyString(`{"error":"missing name parameter"}`)
		return
	}
	name := nameVal.(string)

	// Fetch the existing object
	existing := &jaegerv1alpha0.JaegerNginxProxy{}
	err := api.K8sClient.Get(context.Background(), client.ObjectKey{Namespace: api.Namespace, Name: name}, existing)
	if err != nil {
		ctx.SetStatusCode(fasthttp.StatusNotFound)
		ctx.SetBodyString(fmt.Sprintf(`{"error":"%v"}`, err))
		return
	}

	// Parse the patch request
	var patchData map[string]interface{}
	if err := json.Unmarshal(ctx.PostBody(), &patchData); err != nil {
		ctx.SetStatusCode(fasthttp.StatusBadRequest)
		ctx.SetBodyString(fmt.Sprintf(`{"error":"invalid JSON: %v"}`, err))
		return
	}

	// Apply partial updates to the spec
	if err := api.applyPartialSpecUpdate(existing, patchData); err != nil {
		ctx.SetStatusCode(fasthttp.StatusBadRequest)
		ctx.SetBodyString(fmt.Sprintf(`{"error":"%v"}`, err))
		return
	}

	// Update the resource
	if err := api.K8sClient.Update(context.Background(), existing); err != nil {
		ctx.SetStatusCode(fasthttp.StatusInternalServerError)
		ctx.SetBodyString(fmt.Sprintf(`{"error":"%v"}`, err))
		return
	}

	ctx.SetContentType("application/json")
	json.NewEncoder(ctx).Encode(existing)
}

// applyPartialSpecUpdate applies partial updates to the spec
func (api *JaegerNginxProxyAPI) applyPartialSpecUpdate(existing *jaegerv1alpha0.JaegerNginxProxy, patchData map[string]interface{}) error {
	// Handle spec updates
	if specData, ok := patchData["spec"].(map[string]interface{}); ok {
		// Update replicaCount
		if replicaCount, ok := specData["replicaCount"].(float64); ok {
			existing.Spec.ReplicaCount = int(replicaCount)
		}

		// Update containerPort
		if containerPort, ok := specData["containerPort"].(float64); ok {
			existing.Spec.ContainerPort = int(containerPort)
		}

		// Update image
		if imageData, ok := specData["image"].(map[string]interface{}); ok {
			if repository, ok := imageData["repository"].(string); ok {
				existing.Spec.Image.Repository = repository
			}
			if tag, ok := imageData["tag"].(string); ok {
				existing.Spec.Image.Tag = tag
			}
			if pullPolicy, ok := imageData["pullPolicy"].(string); ok {
				existing.Spec.Image.PullPolicy = pullPolicy
			}
		}

		// Update upstream
		if upstreamData, ok := specData["upstream"].(map[string]interface{}); ok {
			if collectorHost, ok := upstreamData["collectorHost"].(string); ok {
				existing.Spec.Upstream.CollectorHost = collectorHost
			}
		}

		// Update service
		if serviceData, ok := specData["service"].(map[string]interface{}); ok {
			if serviceType, ok := serviceData["type"].(string); ok {
				existing.Spec.Service.Type = serviceType
			}
		}

		// Update resources
		if resourcesData, ok := specData["resources"].(map[string]interface{}); ok {
			// Update limits
			if limitsData, ok := resourcesData["limits"].(map[string]interface{}); ok {
				if cpu, ok := limitsData["cpu"].(string); ok {
					existing.Spec.Resources.Limits.CPU = cpu
				}
				if memory, ok := limitsData["memory"].(string); ok {
					existing.Spec.Resources.Limits.Memory = memory
				}
			}
			// Update requests
			if requestsData, ok := resourcesData["requests"].(map[string]interface{}); ok {
				if cpu, ok := requestsData["cpu"].(string); ok {
					existing.Spec.Resources.Requests.CPU = cpu
				}
				if memory, ok := requestsData["memory"].(string); ok {
					existing.Spec.Resources.Requests.Memory = memory
				}
			}
		}

		// Update ports (replace entire array)
		if portsData, ok := specData["ports"].([]interface{}); ok {
			var newPorts []jaegerv1alpha0.Port
			for _, portInterface := range portsData {
				if portData, ok := portInterface.(map[string]interface{}); ok {
					port := jaegerv1alpha0.Port{}
					if name, ok := portData["name"].(string); ok {
						port.Name = name
					}
					if portNum, ok := portData["port"].(float64); ok {
						port.Port = int(portNum)
					}
					if path, ok := portData["path"].(string); ok {
						port.Path = path
					}
					newPorts = append(newPorts, port)
				}
			}
			if len(newPorts) > 0 {
				existing.Spec.Ports = newPorts
			}
		}
	}

	return nil
}

// DeleteJaegerNginxProxy godoc
// @Summary Delete a JaegerNginxProxy
// @Description Delete a JaegerNginxProxy by name
// @Tags jaegernginxproxies
// @Param name path string true "JaegerNginxProxy name"
// @Success 204 {object} nil
// @Failure 404 {object} map[string]string
// @Router /api/jaegernginxproxies/{name} [delete]
func (api *JaegerNginxProxyAPI) DeleteJaegerNginxProxy(ctx *fasthttp.RequestCtx) {
	nameVal := ctx.UserValue("name")
	if nameVal == nil {
		ctx.SetStatusCode(fasthttp.StatusBadRequest)
		ctx.SetBodyString(`{"error":"missing name parameter"}`)
		return
	}
	name := nameVal.(string)
	obj := &jaegerv1alpha0.JaegerNginxProxy{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: api.Namespace,
		},
	}
	if err := api.K8sClient.Delete(context.Background(), obj); err != nil {
		ctx.SetStatusCode(fasthttp.StatusNotFound)
		ctx.SetBodyString(fmt.Sprintf(`{"error":"%v"}`, err))
		return
	}
	ctx.SetStatusCode(fasthttp.StatusNoContent)
}

// Note: Wire these handlers into your FastHTTP server in cmd/server.go using the appropriate routing logic.
//       You can use a router like fasthttprouter or manually parse the path and method.
