# API Key Authentication Middleware

This project includes an API Key authentication middleware as an alternative to JWT/Keycloak authentication for internal systems and services.

## Overview

The API Key middleware provides a simple and secure way to authenticate requests using API keys managed by the `go_key_guardian` library. This is ideal for:

- Service-to-service communication
- Internal API access
- Third-party integrations
- Simple authentication without OAuth2 complexity

## Configuration

### Environment Variables

Add the following variable to your `.env` file:

```bash
SERVICE_ID=spooliq-service
```

This SERVICE_ID is used to validate API keys against the key guardian service.

### Dependencies

The middleware uses the `go_key_guardian` library:

```bash
go get github.com/RodolfoBonis/go_key_guardian
```

## Usage

### 1. Basic Usage with Dependency Injection

```go
func Routes(route *gin.RouterGroup, apiKeyMiddleware *middlewares.APIKeyMiddleware) {
    api := route.Group("/api")
    
    // Protected route with API Key
    api.GET("/protected", 
        apiKeyMiddleware.ProtectWithAPIKey(), 
        protectedHandler)
}
```

### 2. Standalone Function Usage

```go
func Routes(route *gin.RouterGroup, logger logger.Logger, cfg *config.AppConfig) {
    api := route.Group("/api")
    
    // Using standalone function
    api.GET("/protected", 
        middlewares.ProtectWithAPIKeyFunc(logger, cfg), 
        protectedHandler)
}
```

### 3. Mixed Authentication Routes

```go
func Routes(
    route *gin.RouterGroup, 
    protectMiddleware func(gin.HandlerFunc, string) gin.HandlerFunc,
    apiKeyMiddleware *middlewares.APIKeyMiddleware,
) {
    api := route.Group("/api")
    
    // JWT Authentication
    api.GET("/jwt-protected", 
        protectMiddleware(jwtProtectedHandler, "user"))
    
    // API Key Authentication
    api.GET("/apikey-protected", 
        apiKeyMiddleware.ProtectWithAPIKey(), 
        apikeyProtectedHandler)
    
    // Public endpoint
    api.GET("/public", publicHandler)
}
```

## API Key Headers

The middleware accepts API keys in multiple header formats:

### 1. X-Api-Key Header (Recommended)
```bash
curl -H "X-Api-Key: your-api-key-here" \
  http://localhost:8080/api/protected
```

### 2. X-API-Key Header (Alternative)
```bash
curl -H "X-API-Key: your-api-key-here" \
  http://localhost:8080/api/protected
```

### 3. Authorization Bearer (Compatibility)
```bash
curl -H "Authorization: Bearer your-api-key-here" \
  http://localhost:8080/api/protected
```

## Context Variables

After successful authentication, the middleware sets the following context variables:

```go
func protectedHandler(c *gin.Context) {
    // Get API key configuration
    configs, exists := c.Get("api_key_configs")
    if !exists {
        // Handle error
        return
    }
    
    // Get application ID
    appID, exists := c.Get("application_id")
    if exists {
        fmt.Printf("Request from application: %s\n", appID)
    }
    
    // Your handler logic here
    c.JSON(200, gin.H{"message": "Access granted"})
}
```

## Error Responses

The middleware returns standardized error responses:

### Missing API Key
```json
{
  "error": "API Key is required",
  "status_code": 400,
  "timestamp": "2024-01-15T10:30:00Z"
}
```

### Invalid API Key
```json
{
  "error": "Invalid API key",
  "status_code": 401,
  "timestamp": "2024-01-15T10:30:00Z"
}
```

## Logging

The middleware provides structured logging for authentication events:

### Successful Authentication
```json
{
  "level": "info",
  "message": "API Key authentication successful",
  "application_id": "550e8400-e29b-41d4-a716-446655440000",
  "service_id": "my-service",
  "timestamp": "2024-01-15T10:30:00Z"
}
```

### Failed Authentication
```json
{
  "level": "error",
  "message": "API Key authentication failed: invalid key",
  "error": "Invalid API key provided",
  "timestamp": "2024-01-15T10:30:00Z"
}
```

## Complete Example

### Route Setup
```go
package myfeature

import (
    "github.com/gin-gonic/gin"
    "github.com/RodolfoBonis/spooliq/core/middlewares"
)

func Routes(route *gin.RouterGroup, apiKeyMiddleware *middlewares.APIKeyMiddleware) {
    api := route.Group("/api/v1")
    
    // Public endpoints
    api.GET("/health", healthHandler)
    api.GET("/version", versionHandler)
    
    // API Key protected endpoints
    protected := api.Group("")
    protected.Use(apiKeyMiddleware.ProtectWithAPIKey())
    {
        protected.GET("/users", getUsersHandler)
        protected.POST("/users", createUserHandler)
        protected.GET("/analytics", getAnalyticsHandler)
    }
}
```

### Handler Implementation
```go
func getUsersHandler(c *gin.Context) {
    // Get application info from context
    appID, _ := c.Get("application_id")
    
    // Log the request
    fmt.Printf("Users requested by application: %s\n", appID)
    
    // Your business logic here
    users := []User{
        {ID: 1, Name: "John Doe"},
        {ID: 2, Name: "Jane Smith"},
    }
    
    c.JSON(200, gin.H{
        "users": users,
        "requested_by": appID,
    })
}
```

## Best Practices

### 1. Service Identification
- Use descriptive SERVICE_ID values
- Include environment in SERVICE_ID for multiple deployments
- Example: `user-service-prod`, `payment-service-dev`

### 2. API Key Management
- Generate strong, unique API keys
- Rotate API keys regularly
- Store API keys securely (environment variables, secret managers)
- Never log API keys in plain text

### 3. Error Handling
- Always check for context variables
- Provide meaningful error messages
- Log authentication failures for monitoring

### 4. Security Considerations
- Use HTTPS in production
- Implement rate limiting
- Monitor for suspicious API key usage
- Validate API key permissions for specific operations

## Integration with go_key_guardian

The middleware integrates with the `go_key_guardian` library for API key validation. Make sure to:

1. Set up your key guardian service
2. Register your service with appropriate SERVICE_ID
3. Generate and distribute API keys to authorized applications
4. Monitor API key usage through the guardian dashboard

## Troubleshooting

### API Key Not Working
1. Verify the API key is valid in key guardian
2. Check the SERVICE_ID configuration
3. Ensure the key guardian service is accessible
4. Review application logs for detailed error messages

### Performance Issues
1. Consider caching valid API keys temporarily
2. Implement connection pooling for key guardian requests
3. Monitor response times and optimize as needed

### Debugging
Enable debug logging to see detailed authentication flow:

```go
// In your configuration
logger.SetLevel("debug")
```

This will show API key validation attempts and results. 