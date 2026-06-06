package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func (h *SwaggerHandler) Index(c *gin.Context) {
	c.Header("Content-Type", "text/html; charset=utf-8")
	c.String(http.StatusOK, swaggerIndexHTML)
}

func (h *SwaggerHandler) OpenAPI(c *gin.Context) {
	c.Header("Content-Type", "application/json")
	c.String(http.StatusOK, openAPISpec)
}

type SwaggerHandler struct{}

func NewSwaggerHandler() *SwaggerHandler {
	return &SwaggerHandler{}
}

const swaggerIndexHTML = `<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8" />
  <title>InduGate API Docs</title>
  <link rel="stylesheet" href="https://unpkg.com/swagger-ui-dist@5/swagger-ui.css" />
</head>
<body>
  <div id="swagger-ui"></div>
  <script src="https://unpkg.com/swagger-ui-dist@5/swagger-ui-bundle.js"></script>
  <script>
    SwaggerUIBundle({ url: '/swagger/openapi.json', dom_id: '#swagger-ui' });
  </script>
</body>
</html>`

const openAPISpec = `{
  "openapi": "3.0.3",
  "info": {
    "title": "InduGate API",
    "description": "Industrial Agent Protocol Gateway REST API",
    "version": "0.7.0"
  },
  "servers": [{"url": "/api/v1"}],
  "tags": [
    {"name": "Auth"},
    {"name": "Users"},
    {"name": "Devices"},
    {"name": "Data"},
    {"name": "Alerts"},
    {"name": "Dashboard"},
    {"name": "Audit"},
    {"name": "Simulators"}
  ],
  "paths": {
    "/auth/config": {
      "get": {"summary": "Auth configuration", "tags": ["Auth"], "responses": {"200": {"description": "OK"}}}
    },
    "/auth/login": {
      "post": {"summary": "Login with username/password", "tags": ["Auth"], "responses": {"200": {"description": "OK"}}}
    },
    "/auth/me": {
      "get": {"summary": "Current user profile", "tags": ["Auth"], "responses": {"200": {"description": "OK"}}}
    },
    "/users": {
      "get": {"summary": "List users (admin)", "tags": ["Users"], "responses": {"200": {"description": "OK"}}},
      "post": {"summary": "Create user (admin)", "tags": ["Users"], "responses": {"201": {"description": "Created"}}}
    },
    "/users/{id}": {
      "put": {"summary": "Update user (admin)", "tags": ["Users"], "parameters": [{"name": "id", "in": "path", "required": true, "schema": {"type": "integer"}}], "responses": {"200": {"description": "OK"}}},
      "delete": {"summary": "Delete user (admin)", "tags": ["Users"], "parameters": [{"name": "id", "in": "path", "required": true, "schema": {"type": "integer"}}], "responses": {"200": {"description": "OK"}}}
    },
    "/devices": {
      "get": {"summary": "List devices", "tags": ["Devices"], "responses": {"200": {"description": "OK"}}},
      "post": {"summary": "Create device", "tags": ["Devices"], "responses": {"201": {"description": "Created"}}}
    },
    "/devices/{id}": {
      "get": {"summary": "Get device", "tags": ["Devices"], "parameters": [{"name": "id", "in": "path", "required": true, "schema": {"type": "integer"}}], "responses": {"200": {"description": "OK"}}},
      "put": {"summary": "Update device", "tags": ["Devices"], "parameters": [{"name": "id", "in": "path", "required": true, "schema": {"type": "integer"}}], "responses": {"200": {"description": "OK"}}},
      "delete": {"summary": "Delete device", "tags": ["Devices"], "parameters": [{"name": "id", "in": "path", "required": true, "schema": {"type": "integer"}}], "responses": {"200": {"description": "OK"}}}
    },
    "/devices/{id}/connect": {
      "post": {"summary": "Connect device", "tags": ["Devices"], "parameters": [{"name": "id", "in": "path", "required": true, "schema": {"type": "integer"}}], "responses": {"200": {"description": "OK"}}}
    },
    "/devices/{id}/disconnect": {
      "post": {"summary": "Disconnect device", "tags": ["Devices"], "parameters": [{"name": "id", "in": "path", "required": true, "schema": {"type": "integer"}}], "responses": {"200": {"description": "OK"}}}
    },
    "/devices/{id}/nodes": {
      "get": {"summary": "Browse nodes", "tags": ["Data"], "parameters": [
        {"name": "id", "in": "path", "required": true, "schema": {"type": "integer"}},
        {"name": "node", "in": "query", "schema": {"type": "string"}},
        {"name": "depth", "in": "query", "schema": {"type": "integer"}}
      ], "responses": {"200": {"description": "OK"}}}
    },
    "/devices/{id}/data/history": {
      "get": {"summary": "Query data history", "tags": ["Data"], "parameters": [
        {"name": "id", "in": "path", "required": true, "schema": {"type": "integer"}},
        {"name": "node_id", "in": "query", "schema": {"type": "string"}},
        {"name": "limit", "in": "query", "schema": {"type": "integer"}},
        {"name": "since", "in": "query", "schema": {"type": "string", "format": "date-time"}}
      ], "responses": {"200": {"description": "OK"}}}
    },
    "/devices/{id}/subscribe": {
      "post": {"summary": "Subscribe data changes", "tags": ["Data"], "parameters": [{"name": "id", "in": "path", "required": true, "schema": {"type": "integer"}}], "responses": {"201": {"description": "Created"}}}
    },
    "/simulators": {
      "get": {"summary": "List simulators", "tags": ["Simulators"], "responses": {"200": {"description": "OK"}}}
    },
    "/alerts/rules": {
      "get": {"summary": "List alert rules", "tags": ["Alerts"], "responses": {"200": {"description": "OK"}}},
      "post": {"summary": "Create alert rule", "tags": ["Alerts"], "responses": {"201": {"description": "Created"}}}
    },
    "/alerts/events": {
      "get": {"summary": "List alert events", "tags": ["Alerts"], "responses": {"200": {"description": "OK"}}}
    },
    "/dashboard/stats": {
      "get": {"summary": "Dashboard statistics", "tags": ["Dashboard"], "responses": {"200": {"description": "OK"}}}
    },
    "/audit/logs": {
      "get": {"summary": "Query audit logs (admin)", "tags": ["Audit"], "responses": {"200": {"description": "OK"}}}
    },
    "/simulators/{type}/start": {
      "post": {"summary": "Start simulator", "tags": ["Simulators"], "parameters": [{"name": "type", "in": "path", "required": true, "schema": {"type": "string", "enum": ["opcua", "modbus", "mqtt"]}}], "responses": {"200": {"description": "OK"}}}
    },
    "/simulators/{type}/stop": {
      "post": {"summary": "Stop simulator", "tags": ["Simulators"], "parameters": [{"name": "type", "in": "path", "required": true, "schema": {"type": "string"}}], "responses": {"200": {"description": "OK"}}}
    },
    "/simulators/{type}/config": {
      "put": {"summary": "Update simulator config", "tags": ["Simulators"], "parameters": [{"name": "type", "in": "path", "required": true, "schema": {"type": "string"}}], "responses": {"200": {"description": "OK"}}}
    }
  }
}`
