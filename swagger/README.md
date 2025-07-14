# Swagger API Documentation

This directory contains the Swagger UI for the Kubernetes Controller Tutorial API.

## Accessing the API Documentation

Once the server is running, you can access the Swagger UI at:

- **Swagger UI**: http://localhost:8080/swagger
- **Swagger JSON**: http://localhost:8080/docs/swagger.json

## Available Endpoints

The API provides the following endpoints for managing JaegerNginxProxy resources:

### REST API Endpoints (Port 8080)
- `GET /api/jaegernginxproxies` - List all JaegerNginxProxy resources
- `GET /api/jaegernginxproxies/{name}` - Get a JaegerNginxProxy by name
- `POST /api/jaegernginxproxies` - Create a new JaegerNginxProxy
- `PUT /api/jaegernginxproxies/{name}` - Update an existing JaegerNginxProxy (full update)
- `PATCH /api/jaegernginxproxies/{name}` - Patch an existing JaegerNginxProxy (partial update)
- `DELETE /api/jaegernginxproxies/{name}` - Delete a JaegerNginxProxy

### MCP (Model Context Protocol) Endpoints (Port 9090)
- `GET /sse` - Server-Sent Events stream for tool capabilities
- `POST /message` - JSON-RPC endpoint for tool invocation

### Other Endpoints
- `GET /deployments` - List deployment names from informer cache
- `GET /docs/swagger.json` - Get Swagger JSON specification
- `GET /swagger` - Get Swagger UI

## Partial Updates with PATCH

The PATCH endpoint allows you to update only specific fields of a JaegerNginxProxy resource without sending the complete object. This is more efficient and reduces the risk of accidentally overwriting other fields.

### Example PATCH Requests

#### Update replica count only:
```bash
curl -X PATCH http://localhost:8080/api/jaegernginxproxies/test-proxy \
  -H 'Content-Type: application/json' \
  -d '{
    "spec": {
      "replicaCount": 3
    }
  }'
```

#### Update image tag only:
```bash
curl -X PATCH http://localhost:8080/api/jaegernginxproxies/test-proxy \
  -H 'Content-Type: application/json' \
  -d '{
    "spec": {
      "image": {
        "tag": "1.22"
      }
    }
  }'
```

#### Update resources only:
```bash
curl -X PATCH http://localhost:8080/api/jaegernginxproxies/test-proxy \
  -H 'Content-Type: application/json' \
  -d '{
    "spec": {
      "resources": {
        "limits": {
          "cpu": "1000m",
          "memory": "1Gi"
        }
      }
    }
  }'
```

#### Update multiple fields:
```bash
curl -X PATCH http://localhost:8080/api/jaegernginxproxies/test-proxy \
  -H 'Content-Type: application/json' \
  -d '{
    "spec": {
      "replicaCount": 2,
      "containerPort": 9090,
      "image": {
        "tag": "1.23"
      },
      "resources": {
        "requests": {
          "cpu": "200m",
          "memory": "256Mi"
        }
      }
    }
  }'
```

### Supported Fields for PATCH

You can update any of these fields individually or in combination:

- `spec.replicaCount` - Number of replicas (minimum 1, webhook validation will reject 0)
- `spec.containerPort` - Container port
- `spec.image.repository` - Image repository
- `spec.image.tag` - Image tag
- `spec.image.pullPolicy` - Image pull policy
- `spec.upstream.collectorHost` - Upstream collector host
- `spec.service.type` - Service type
- `spec.resources.limits.cpu` - CPU limits
- `spec.resources.limits.memory` - Memory limits
- `spec.resources.requests.cpu` - CPU requests
- `spec.resources.requests.memory` - Memory requests
- `spec.ports` - Ports array (replaces entire array)

### Important Notes

**Replica Count Validation**: The webhook validates that `replicaCount` must be greater than 0. If you try to set it to 0, the request will be rejected with a validation error.

**Status Behavior**: 
- When `replicaCount > 0` and all pods are running: `ready: true, message: "All X pods are running"`
- When `replicaCount > 0` but not all pods are ready: `ready: false, message: "Available replicas: X/Y..."`
- When `replicaCount = 0` and no pods running: `ready: true, message: "Deployment scaled to 0 replicas"`
- When `replicaCount = 0` but pods still running: `ready: false, message: "Scaling down: X pods still running, desired: 0"`

## MCP (Model Context Protocol) Integration

The API includes MCP server support for AI/LLM integration. When enabled with `--enable-mcp`, the server provides:

### Available MCP Tools
- `list_jaegernginxproxies` - List all JaegerNginxProxy resources
- `create_jaegernginxproxy` - Create a new JaegerNginxProxy resource

### MCP Usage
1. **Connect to SSE stream**: `GET http://localhost:9090/sse`
2. **Send JSON-RPC messages**: `POST http://localhost:9090/message`
3. **Example tool invocation**:
   ```json
   {
     "jsonrpc": "2.0",
     "id": 1,
     "method": "tools/call",
     "params": {
       "name": "list_jaegernginxproxies"
     }
   }
   ```

### When to Use MCP vs REST API
- **Use MCP for**: AI/LLM integrations, chatbots, intelligent DevOps tools
- **Use REST API for**: Traditional CLI tools, scripts, direct programmatic access

## Regenerating Documentation

To regenerate the Swagger documentation after making changes to the API:

```bash
swag init -g main.go
```

This will update the files in the `docs/` directory:
- `docs/docs.go` - Generated Go code
- `docs/swagger.json` - OpenAPI specification in JSON format
- `docs/swagger.yaml` - OpenAPI specification in YAML format

## Features

- Interactive API documentation
- Try-it-out functionality for testing endpoints
- Request/response examples
- Schema validation
- Modern, responsive UI 