// @title K8s Controller Tutorial API
// @version 1.0
// @description A Kubernetes controller tutorial with JaegerNginxProxy CRD and MCP (Model Context Protocol) support. The API provides REST endpoints for managing JaegerNginxProxy resources and includes an MCP server for AI/LLM integration.
// @termsOfService http://swagger.io/terms/

// @contact.name API Support
// @contact.url http://www.swagger.io/support
// @contact.email support@swagger.io

// @license.name Apache 2.0
// @license.url http://www.apache.org/licenses/LICENSE-2.0.html

// @host localhost:8080
// @BasePath /

// @tag.name jaegernginxproxies
// @tag.description Operations about JaegerNginxProxy resources - CRUD operations for managing nginx proxy configurations for Jaeger tracing

// @tag.name documentation
// @tag.description API documentation endpoints - Swagger UI and JSON specification

// @tag.name deployments
// @tag.description Deployment listing operations - View Kubernetes deployments from the informer cache

// @tag.name mcp
// @tag.description Model Context Protocol (MCP) server endpoints - AI/LLM integration for intelligent DevOps operations. MCP server runs on port 9090 with SSE stream at /sse and JSON-RPC messages at /message

// @securityDefinitions.apikey ApiKeyAuth
// @in header
// @name Authorization

package main

import "github.com/dolv/k8s-controller-tutorial/cmd"

func main() {
	cmd.Execute()
}
