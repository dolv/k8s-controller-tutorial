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
// @Summary Update a JaegerNginxProxy
// @Description Update an existing JaegerNginxProxy
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
