package cmd

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/dolv/k8s-controller-tutorial/pkg/api"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	jaegerv1alpha0 "github.com/dolv/k8s-controller-tutorial/pkg/apis/jaeger-nginx-proxy/v1alpha0"
)

// NewMCPServer creates and configures a new MCP server for JaegerNginxProxy tools
func NewMCPServer(serverName, version string) *server.MCPServer {
	s := server.NewMCPServer(
		serverName,
		version,
		server.WithToolCapabilities(true),
		server.WithLogging(),
		server.WithRecovery(),
	)

	// List tool
	listTool := mcp.NewTool("list_jaegernginxproxies",
		mcp.WithDescription("List all JaegerNginxProxy resources"),
	)
	// Create tool
	createTool := mcp.NewTool("create_jaegernginxproxy",
		mcp.WithDescription("Create a new JaegerNginxProxy resource"),
		mcp.WithString("name", mcp.Description("Name of the JaegerNginxProxy")),
		mcp.WithNumber("replicaCount", mcp.Description("Number of replicas")),
		mcp.WithNumber("containerPort", mcp.Description("Container port")),
		mcp.WithString("imageRepository", mcp.Description("Image repository")),
		mcp.WithString("imageTag", mcp.Description("Image tag")),
		mcp.WithString("imagePullPolicy", mcp.Description("Image pull policy")),
		mcp.WithString("upstreamCollectorHost", mcp.Description("Upstream collector host")),
		// Add more fields as needed for full spec
	)
	// TODO: Add update and delete tools as needed

	s.AddTool(listTool, listJaegerNginxProxiesHandler)
	s.AddTool(createTool, createJaegerNginxProxyHandler)
	// TODO: Register update/delete handlers

	return s
}

func listJaegerNginxProxiesHandler(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	if api.JaegerNginxProxyAPIInst == nil {
		return mcp.NewToolResultText("JaegerNginxProxyAPI is not initialized"), nil
	}
	list := &api.JaegerNginxProxyListDoc{}
	// Use the API to get all JaegerNginxProxy resources
	proxies := &[]api.JaegerNginxProxyDoc{}
	// Use the underlying K8sClient to list resources
	k8sList := &jaegerv1alpha0.JaegerNginxProxyList{}
	err := api.JaegerNginxProxyAPIInst.K8sClient.List(ctx, k8sList, client.InNamespace(api.JaegerNginxProxyAPIInst.Namespace))
	if err != nil {
		return mcp.NewToolResultText(fmt.Sprintf("Error listing JaegerNginxProxies: %v", err)), nil
	}
	for _, item := range k8sList.Items {
		*proxies = append(*proxies, api.JaegerNginxProxyDoc{
			Name:   item.Name,
			Spec:   item.Spec,
			Status: item.Status,
		})
	}
	list.Items = *proxies
	jsonBytes, err := json.MarshalIndent(list, "", "  ")
	if err != nil {
		return mcp.NewToolResultText(fmt.Sprintf("Error marshaling result: %v", err)), nil
	}
	return mcp.NewToolResultText(string(jsonBytes)), nil
}

func createJaegerNginxProxyHandler(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	if api.JaegerNginxProxyAPIInst == nil {
		return mcp.NewToolResultText("JaegerNginxProxyAPI is not initialized"), nil
	}
	name := req.GetString("name", "")
	replicaCount := req.GetInt("replicaCount", 1)
	containerPort := req.GetInt("containerPort", 8080)
	imageRepository := req.GetString("imageRepository", "nginx")
	imageTag := req.GetString("imageTag", "latest")
	imagePullPolicy := req.GetString("imagePullPolicy", "IfNotPresent")
	upstreamCollectorHost := req.GetString("upstreamCollectorHost", "")
	// TODO: Add more fields as needed

	obj := &jaegerv1alpha0.JaegerNginxProxy{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: api.JaegerNginxProxyAPIInst.Namespace,
		},
		Spec: jaegerv1alpha0.JaegerNginxProxySpec{
			ReplicaCount:  int(replicaCount),
			ContainerPort: int(containerPort),
			Image: jaegerv1alpha0.Image{
				Repository: imageRepository,
				Tag:        imageTag,
				PullPolicy: imagePullPolicy,
			},
			Upstream: jaegerv1alpha0.Upstream{
				CollectorHost: upstreamCollectorHost,
			},
			// You may want to add Ports, Service, Resources, etc.
		},
	}
	err := api.JaegerNginxProxyAPIInst.K8sClient.Create(ctx, obj)
	if err != nil {
		return mcp.NewToolResultText(fmt.Sprintf("Error creating JaegerNginxProxy: %v", err)), nil
	}
	jsonBytes, err := json.MarshalIndent(obj, "", "  ")
	if err != nil {
		return mcp.NewToolResultText(fmt.Sprintf("Error marshaling result: %v", err)), nil
	}
	return mcp.NewToolResultText(string(jsonBytes)), nil
}
