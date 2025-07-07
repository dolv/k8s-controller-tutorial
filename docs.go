// Package main provides API documentation for the Kubernetes Controller Tutorial
//
// This is a Kubernetes controller that manages JaegerNginxProxy custom resources.
// It provides REST API endpoints for CRUD operations on JaegerNginxProxy resources.
//
//	Schemes: http, https
//	Host: localhost:8080
//	BasePath: /
//	Version: 1.0.0
//
//	Consumes:
//	- application/json
//
//	Produces:
//	- application/json
//
//	Security:
//	- basic
//
// swagger:meta
package main

import _ "github.com/dolv/k8s-controller-tutorial/docs"
