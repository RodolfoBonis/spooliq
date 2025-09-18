# User-Specific Cache (CacheUserSpecific)

This document explains how `CacheUserSpecific` works and how it obtains the user ID.

## How It Works

`CacheUserSpecific` creates cache keys that include the user ID, ensuring each user has their own isolated cache.

### Example Generated Key
```
cache:/api/profile:user:550e8400-e29b-41d4-a716-446655440000
```

## How User ID is Obtained

The cache middleware tries to obtain the user ID in several ways, in the following order:

### 1. **Direct from Context** (Recommended)
```go
c.Set("user_id", "550e8400-e29b-41d4-a716-446655440000")
```

### 2. **From JWT Claims**
```go
c.Set("claims", userClaim) // userClaim has ID field
```

### 3. **User UUID** (alternative naming)
```go
c.Set("user_uuid", "550e8400-e29b-41d4-a716-446655440000")
```

### 4. **Sub Claim** (JWT standard)
```go
c.Set("sub", "550e8400-e29b-41d4-a716-446655440000")
```

## Required Configuration

### Authentication Middleware

The authentication middleware **must run BEFORE** cache and set the user ID:

```go
func authMiddleware() gin.HandlerFunc {
    return func(c *gin.Context) {
        // ... token validation ...
        
        // Option 1: Set user_id directly (RECOMMENDED)
        c.Set("user_id", userClaim.ID.String())
        
        // Option 2: Set complete claims (also works)
        c.Set("claims", userClaim)
        
        c.Next()
    }
}
```

### Middleware Order

```go
router.GET("/api/profile", 
    authMiddleware(),           // 1st - Authenticate and set user_id
    cacheMiddleware.CacheUserSpecific(10*time.Minute), // 2nd - Cache per user
    getProfileHandler)         // 3rd - Final handler
```

## Practical Examples

### 1. **User Profile Cache**
```go
func UserRoutes(route *gin.RouterGroup, cacheMiddleware *middlewares.CacheMiddleware) {
    user := route.Group("/user")
    user.Use(authMiddleware()) // Global middleware for the group
    
    // User-specific cache for 10 minutes
    user.GET("/profile", 
        cacheMiddleware.CacheUserSpecific(10*time.Minute), 
        getUserProfileHandler)
    
    // User preferences cache for 30 minutes
    user.GET("/preferences", 
        cacheMiddleware.CacheUserSpecific(30*time.Minute), 
        getUserPreferencesHandler)
}
```

### 2. **Personalized Dashboard Cache**
```go
func DashboardRoutes(route *gin.RouterGroup, cacheMiddleware *middlewares.CacheMiddleware) {
    dashboard := route.Group("/dashboard")
    dashboard.Use(authMiddleware())
    
    // Each user has their own metrics cache
    dashboard.GET("/metrics", 
        cacheMiddleware.CacheUserSpecific(5*time.Minute), 
        getDashboardMetricsHandler)
}
```

### 3. **Combined Cache (User + Query)**
```go
// Cache that varies by both user and query parameters
router.GET("/api/search", 
    authMiddleware(),
    cacheMiddleware.Cache(middlewares.CacheConfig{
        TTL:         5 * time.Minute,
        VaryByUser:  true,  // Include user_id in key
        VaryByQuery: true,  // Include query params in key
    }), 
    searchHandler)

// Example generated keys:
// cache:/api/search:user:123:query:q=laptop&category=tech
// cache:/api/search:user:456:query:q=laptop&category=tech
```

## Behavior Without User ID

If user ID is not found in context:

1. **Warning Log** (in development)
2. **Global cache** will be used (no user separation)
3. **No error** - application continues working

```go
// If user_id doesn't exist, the key will be:
cache:/api/profile  // WITHOUT the :user:123
```

## Debug and Troubleshooting

### 1. **Check if User ID is being set**
```go
func debugMiddleware() gin.HandlerFunc {
    return func(c *gin.Context) {
        c.Next()
        
        // Debug log
        if userID, exists := c.Get("user_id"); exists {
            fmt.Printf("User ID found: %v\n", userID)
        } else {
            fmt.Println("User ID NOT found in context")
        }
    }
}
```

### 2. **Response Headers**
Cache adds useful headers for debugging:
```
X-Cache: HIT    # Cache was used
X-Cache: MISS   # Cache was not found
```

### 3. **Structured Logs**
```json
{
  "level": "debug",
  "message": "Response cached successfully",
  "cache_key": "cache:/api/profile:user:550e8400-e29b-41d4-a716-446655440000",
  "ttl": "10m0s"
}
```

## Security

### ✅ **Best Practices**
- Cache user-specific data (profile, preferences)
- Cache personalized search results
- Cache individual dashboards

### ❌ **Don't Use For**
- Sensitive data (passwords, tokens)
- Data that must always be fresh (bank balance)
- Data shared between users

## Use Case Examples

### 1. **E-commerce**
```go
// User shopping cart cache
router.GET("/cart", cacheMiddleware.CacheUserSpecific(5*time.Minute), getCartHandler)

// Personalized recommendations cache
router.GET("/recommendations", cacheMiddleware.CacheUserSpecific(1*time.Hour), getRecommendationsHandler)
```

### 2. **Notification System**
```go
// Unread notifications cache per user
router.GET("/notifications/unread", cacheMiddleware.CacheUserSpecific(2*time.Minute), getUnreadNotificationsHandler)
```

### 3. **Personalized Settings**
```go
// User interface settings cache
router.GET("/settings/ui", cacheMiddleware.CacheUserSpecific(1*time.Hour), getUISettingsHandler)
```

## Performance

### Advantages
- **Isolation**: Each user has independent cache
- **Personalization**: User-specific data
- **Efficiency**: Avoids recalculation of personalized data

### Considerations
- **Memory Usage**: More cache keys = more memory
- **Appropriate TTL**: Adjust according to data change frequency
- **Invalidation**: Consider invalidating when user data changes 