package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/dolv/k8s-controller-tutorial/pkg/api"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/rs/zerolog/log"
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

	// Track tool names for list_tools
	toolNames := []string{}

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
		mcp.WithString("upstreamCollectorHost", mcp.Description("Upstream collector host")),
		mcp.WithArray("ports", mcp.Description("List of ports for the service")),
		// Add more fields as needed for full spec
	)
	// TODO: Add update and delete tools as needed
	log.Info().Msg("[MCP] Registering list_jaegernginxproxies tool")
	s.AddTool(listTool, listJaegerNginxProxiesHandler)
	log.Info().Msg("[MCP] Registering list_jaegernginxproxies create_jaegernginxproxy")
	s.AddTool(createTool, createJaegerNginxProxyHandler)
	toolNames = append(toolNames, "list_jaegernginxproxies", "create_jaegernginxproxy")
	// TODO: Register update/delete handlers

	// Add list_tools and aliases
	listTools := mcp.NewTool("list_tools", mcp.WithDescription("List all registered tools"))
	toolCapabilities := mcp.NewTool("tool.capabilities", mcp.WithDescription("List all registered tools (alias)"))
	toolsList := mcp.NewTool("tools.list", mcp.WithDescription("List all registered tools (alias)"))

	listToolsHandler := func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return mcp.NewToolResultText(strings.Join(toolNames, ",")), nil
	}

	s.AddTool(listTools, listToolsHandler)
	s.AddTool(toolCapabilities, listToolsHandler)
	s.AddTool(toolsList, listToolsHandler)
	toolNames = append(toolNames, "list_tools", "tool.capabilities", "tools.list")

	return s
}

// listJaegerNginxProxiesHandler handles the list_jaegernginxproxies MCP tool
// Lists all JaegerNginxProxy resources in the configured namespace
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

// createJaegerNginxProxyHandler handles the create_jaegernginxproxy MCP tool
// Creates a new JaegerNginxProxy resource with the specified parameters
func createJaegerNginxProxyHandler(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	if api.JaegerNginxProxyAPIInst == nil {
		return mcp.NewToolResultText("JaegerNginxProxyAPI is not initialized"), nil
	}
	name := req.GetString("name", "")
	replicaCount := req.GetInt("replicaCount", 1)
	containerPort := req.GetInt("containerPort", 8080)
	imageRepository := req.GetString("imageRepository", "nginx")
	imageTag := req.GetString("imageTag", "1.28.0")
	imagePullPolicy := req.GetString("imagePullPolicy", "IfNotPresent")
	upstreamCollectorHost := req.GetString("upstreamCollectorHost", "jaeger-collector.tracing.svc.cluster.local")

	// Parse ports argument (array of objects)
	var ports []jaegerv1alpha0.Port
	if args := req.GetArguments(); args != nil {
		if arr, ok := args["ports"].([]interface{}); ok {
			for _, p := range arr {
				if portMap, ok := p.(map[string]interface{}); ok {
					port := jaegerv1alpha0.Port{}
					if v, ok := portMap["name"].(string); ok {
						port.Name = v
					}
					if v, ok := portMap["port"].(float64); ok {
						port.Port = int(v)
					}
					if v, ok := portMap["path"].(string); ok {
						port.Path = v
					}
					ports = append(ports, port)
				}
			}
		}
	}
	// If ports is still empty, use defaults from examples/jaegernginxproxy.yaml
	if len(ports) == 0 {
		ports = []jaegerv1alpha0.Port{
			{Name: "http", Port: 14268, Path: "/api/traces"},
			{Name: "grpc", Port: 14250, Path: "/jaeger.api.v2.CollectorService/PostSpans"},
		}
	}

	// Parse service argument (object)
	var service jaegerv1alpha0.Service
	service.Type = "ClusterIP" // default
	if args := req.GetArguments(); args != nil {
		if svc, ok := args["service"].(map[string]interface{}); ok {
			if v, ok := svc["type"].(string); ok {
				service.Type = v
			}
		}
	}

	// Parse resources argument (object)
	var resources jaegerv1alpha0.Resources
	resources.Limits.CPU = "500m"
	resources.Limits.Memory = "512Mi"
	resources.Requests.CPU = "100m"
	resources.Requests.Memory = "128Mi"
	if args := req.GetArguments(); args != nil {
		if res, ok := args["resources"].(map[string]interface{}); ok {
			if limits, ok := res["limits"].(map[string]interface{}); ok {
				if v, ok := limits["cpu"].(string); ok {
					resources.Limits.CPU = v
				}
				if v, ok := limits["memory"].(string); ok {
					resources.Limits.Memory = v
				}
			}
			if requests, ok := res["requests"].(map[string]interface{}); ok {
				if v, ok := requests["cpu"].(string); ok {
					resources.Requests.CPU = v
				}
				if v, ok := requests["memory"].(string); ok {
					resources.Requests.Memory = v
				}
			}
		}
	}

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
			Ports:     ports,
			Service:   service,
			Resources: resources,
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
