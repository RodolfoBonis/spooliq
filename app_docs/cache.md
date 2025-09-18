# Redis Cache System

This project includes a complete Redis cache system integrated with easy-to-use middlewares, similar to TypeScript decorators.

## Configuration

### Environment Variables

```bash
REDIS_HOST=localhost
REDIS_PORT=6379
REDIS_PASSWORD=redis123
REDIS_DB=0
```

### Docker Compose

Redis is included in the default infrastructure:

```bash
make infrastructure/raise
```

## How to Use

### 1. Simple Cache (Decorators)

```go
// 5-minute cache
router.GET("/api/data", cacheMiddleware.Cache5Min(), handler)

// 15-minute cache
router.GET("/api/data", cacheMiddleware.Cache15Min(), handler)

// 1-hour cache
router.GET("/api/data", cacheMiddleware.Cache1Hour(), handler)
```

### 2. Cache with Custom Configuration

```go
router.GET("/api/data", cacheMiddleware.Cache(middlewares.CacheConfig{
    TTL:       30 * time.Minute,
    KeyPrefix: "api-data",
    VaryByQuery: true,
    VaryByUser: true,
}), handler)
```

### 3. User-Specific Cache

```go
// Cache that varies by user (includes user_id in the key)
router.GET("/api/profile", cacheMiddleware.CacheUserSpecific(10*time.Minute), handler)
```

### 4. Cache with Query Parameters

```go
// Cache that considers query parameters in the key
router.GET("/api/search", cacheMiddleware.CacheWithQuery(5*time.Minute), handler)
```

### 5. Conditional Cache

```go
// Cache that is only active under certain conditions
router.GET("/api/data", cacheMiddleware.CacheConditional(5*time.Minute, func(c *gin.Context) bool {
    // Only cache if there's no "no-cache" header
    return c.GetHeader("Cache-Control") != "no-cache"
}), handler)
```

## Advanced Configurations

### CacheConfig Options

```go
type CacheConfig struct {
    TTL          time.Duration         // Cache time-to-live
    KeyPrefix    string               // Prefix for keys
    VaryByUser   bool                 // Include user_id in key
    VaryByQuery  bool                 // Include query parameters in key
    VaryByHeader []string             // Specific headers to include in key
    Condition    func(*gin.Context) bool // Condition to activate cache
}
```

### Complete Example

```go
router.GET("/api/products", cacheMiddleware.Cache(middlewares.CacheConfig{
    TTL:       1 * time.Hour,
    KeyPrefix: "products",
    VaryByQuery: true,
    VaryByHeader: []string{"Accept-Language"},
    Condition: func(c *gin.Context) bool {
        // Don't cache for admins
        role, _ := c.Get("user_role")
        return role != "admin"
    },
}), getProductsHandler)
```

## Response Headers

The system adds informative headers:

- `X-Cache: HIT` - Response came from cache
- `X-Cache: MISS` - Response was processed and cached

## Direct Redis Service

Besides middlewares, you can use Redis directly:

```go
func (s *MyService) GetData(ctx context.Context, key string) (string, error) {
    // Try to fetch from cache
    cachedData, err := s.redisService.Get(ctx, key)
    if err == nil && cachedData != "" {
        return cachedData, nil
    }
    
    // Fetch from real data source
    data := s.fetchFromDatabase(key)
    
    // Save to cache
    s.redisService.Set(ctx, key, data, 10*time.Minute)
    
    return data, nil
}
```

### Available Methods

```go
// Basic operations
redisService.Set(ctx, key, value, expiration)
redisService.Get(ctx, key)
redisService.Delete(ctx, key)
redisService.Exists(ctx, key)

// JSON helpers
redisService.SetWithJSON(ctx, key, object, expiration)
redisService.GetWithJSON(ctx, key, &destObject)
```

## Best Practices

### 1. Appropriate TTL
- **Static data**: 1 hour or more
- **Dynamic data**: 5-15 minutes
- **Critical data**: 1-5 minutes

### 2. Cache Keys
- Use descriptive prefixes
- Include API version if necessary
- Consider invalidation when data changes

### 3. Cache Conditions
- Don't cache sensitive data
- Consider user role
- Respect client cache headers

### 4. Monitoring
- Monitor hit rate
- Observe memory usage
- Check response times

## Cache Invalidation

```go
// Clear specific cache
cacheMiddleware.ClearCache(c, "cache:key:pattern")

// Or using the service directly
redisService.Delete(ctx, "specific-key")
```

## Troubleshooting

### Cache is not working
1. Check if Redis is running
2. Confirm credentials
3. Check application logs

### Performance
1. Monitor hit rate
2. Adjust TTL as needed
3. Use organized prefixes

### Development
- Use `no-cache` query parameter to skip cache during tests
- Monitor cache logs for debugging 