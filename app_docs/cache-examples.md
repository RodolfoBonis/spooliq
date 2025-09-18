# Redis Cache Practical Examples

This file contains practical examples of how to use the Redis cache system in the application.

## 1. Basic Example - Product List

```go
func ProductRoutes(route *gin.RouterGroup, cacheMiddleware *middlewares.CacheMiddleware) {
    products := route.Group("/products")
    
    // Simple 15-minute cache for product list
    products.GET("", cacheMiddleware.Cache15Min(), getProductsHandler)
    
    // Specific cache per product with 1 hour
    products.GET("/:id", cacheMiddleware.Cache1Hour(), getProductByIdHandler)
}
```

## 2. User-Specific Cache

```go
func UserRoutes(route *gin.RouterGroup, cacheMiddleware *middlewares.CacheMiddleware) {
    user := route.Group("/user")
    
    // User profile cache for 10 minutes
    user.GET("/profile", 
        cacheMiddleware.CacheUserSpecific(10*time.Minute), 
        getUserProfileHandler)
    
    // User preferences cache for 30 minutes
    user.GET("/preferences", 
        cacheMiddleware.CacheUserSpecific(30*time.Minute), 
        getUserPreferencesHandler)
}
```

## 3. Cache with Query Parameters

```go
func SearchRoutes(route *gin.RouterGroup, cacheMiddleware *middlewares.CacheMiddleware) {
    search := route.Group("/search")
    
    // Search results cache considering query params
    search.GET("/products", 
        cacheMiddleware.CacheWithQuery(5*time.Minute), 
        searchProductsHandler)
    
    // Example: GET /search/products?q=laptop&category=electronics&sort=price
    // Each different parameter combination will have its own cache
}
```

## 4. Advanced Conditional Cache

```go
func AdminRoutes(route *gin.RouterGroup, cacheMiddleware *middlewares.CacheMiddleware) {
    admin := route.Group("/admin")
    
    // Cache only for non-admin users
    admin.GET("/stats", cacheMiddleware.CacheConditional(
        15*time.Minute, 
        func(c *gin.Context) bool {
            role, _ := c.Get("user_role")
            return role != "admin" // Admins always see fresh data
        }), 
        getStatsHandler)
}
```

## 5. Cache with Custom Headers

```go
func ApiRoutes(route *gin.RouterGroup, cacheMiddleware *middlewares.CacheMiddleware) {
    api := route.Group("/api")
    
    // Cache that varies by language and timezone
    api.GET("/content", cacheMiddleware.Cache(middlewares.CacheConfig{
        TTL: 1 * time.Hour,
        KeyPrefix: "content",
        VaryByQuery: true,
        VaryByHeader: []string{"Accept-Language", "X-Timezone"},
    }), getContentHandler)
}
```

## 6. Direct Redis Service Usage

```go
type ProductService struct {
    redisService *services.RedisService
    database     *gorm.DB
    logger       logger.Logger
}

func (s *ProductService) GetProduct(ctx context.Context, id string) (*Product, error) {
    cacheKey := fmt.Sprintf("product:%s", id)
    
    // Try to fetch from cache
    var product Product
    if err := s.redisService.GetWithJSON(ctx, cacheKey, &product); err == nil {
        s.logger.Debug(ctx, "Product found in cache", map[string]interface{}{
            "product_id": id,
        })
        return &product, nil
    }
    
    // Fetch from database
    if err := s.database.First(&product, "id = ?", id).Error; err != nil {
        return nil, err
    }
    
    // Save to cache for 1 hour
    if err := s.redisService.SetWithJSON(ctx, cacheKey, product, 1*time.Hour); err != nil {
        s.logger.Error(ctx, "Failed to cache product", map[string]interface{}{
            "product_id": id,
            "error": err.Error(),
        })
    }
    
    return &product, nil
}

func (s *ProductService) InvalidateProduct(ctx context.Context, id string) error {
    cacheKey := fmt.Sprintf("product:%s", id)
    return s.redisService.Delete(ctx, cacheKey)
}
```

## 7. Cache with Automatic Invalidation

```go
type CategoryService struct {
    redisService *services.RedisService
    database     *gorm.DB
}

func (s *CategoryService) GetCategories(ctx context.Context) ([]Category, error) {
    cacheKey := "categories:all"
    
    // Fetch from cache
    var categories []Category
    if err := s.redisService.GetWithJSON(ctx, cacheKey, &categories); err == nil {
        return categories, nil
    }
    
    // Fetch from database
    if err := s.database.Find(&categories).Error; err != nil {
        return nil, err
    }
    
    // Cache for 2 hours
    s.redisService.SetWithJSON(ctx, cacheKey, categories, 2*time.Hour)
    
    return categories, nil
}

func (s *CategoryService) CreateCategory(ctx context.Context, category *Category) error {
    // Create in database
    if err := s.database.Create(category).Error; err != nil {
        return err
    }
    
    // Invalidate cache
    s.redisService.Delete(ctx, "categories:all")
    
    return nil
}
```

## 8. Cache-Aside Pattern

```go
type UserService struct {
    redisService *services.RedisService
    database     *gorm.DB
}

func (s *UserService) GetUserWithCache(ctx context.Context, id string) (*User, error) {
    // 1. Check cache
    user, err := s.getUserFromCache(ctx, id)
    if err == nil && user != nil {
        return user, nil
    }
    
    // 2. Fetch from database
    user, err = s.getUserFromDatabase(ctx, id)
    if err != nil {
        return nil, err
    }
    
    // 3. Update cache
    go s.setUserInCache(ctx, id, user) // Async to not affect performance
    
    return user, nil
}

func (s *UserService) getUserFromCache(ctx context.Context, id string) (*User, error) {
    var user User
    err := s.redisService.GetWithJSON(ctx, fmt.Sprintf("user:%s", id), &user)
    if err != nil {
        return nil, err
    }
    return &user, nil
}

func (s *UserService) setUserInCache(ctx context.Context, id string, user *User) {
    s.redisService.SetWithJSON(ctx, fmt.Sprintf("user:%s", id), user, 30*time.Minute)
}
```

## 9. Cache for Aggregated Data

```go
func DashboardRoutes(route *gin.RouterGroup, cacheMiddleware *middlewares.CacheMiddleware) {
    dashboard := route.Group("/dashboard")
    
    // Metrics cache for 5 minutes (frequently changing data)
    dashboard.GET("/metrics", 
        cacheMiddleware.Cache(middlewares.CacheConfig{
            TTL: 5 * time.Minute,
            KeyPrefix: "dashboard-metrics",
            VaryByUser: true, // Each user may have different metrics
        }), 
        getDashboardMetricsHandler)
    
    // Reports cache for 1 hour (less volatile data)
    dashboard.GET("/reports", 
        cacheMiddleware.Cache1Hour(), 
        getReportsHandler)
}
```

## 10. Cache with Warmup

```go
func (s *ProductService) WarmupCache(ctx context.Context) error {
    // Fetch most popular products
    var popularProducts []Product
    err := s.database.
        Order("view_count DESC").
        Limit(100).
        Find(&popularProducts).Error
    if err != nil {
        return err
    }
    
    // Pre-populate cache
    for _, product := range popularProducts {
        cacheKey := fmt.Sprintf("product:%s", product.ID)
        s.redisService.SetWithJSON(ctx, cacheKey, product, 1*time.Hour)
    }
    
    return nil
}
```

## Performance Tips

### 1. Appropriate Cache Sizes
- Small objects: Direct JSON
- Large objects: Consider compression
- Very large lists: Paginate and cache pages separately

### 2. TTL Strategies
```go
// TTL based on data type
const (
    UserProfileTTL    = 15 * time.Minute  // Data that changes moderately
    ProductCatalogTTL = 1 * time.Hour     // Relatively static data
    RealtimeDataTTL   = 30 * time.Second  // Real-time data
    ConfigDataTTL     = 24 * time.Hour    // Configuration data
)
```

### 3. Monitoring
```go
func (s *CacheService) LogCacheHit(ctx context.Context, key string) {
    s.logger.Debug(ctx, "Cache hit", map[string]interface{}{
        "cache_key": key,
        "cache_status": "hit",
    })
}

func (s *CacheService) LogCacheMiss(ctx context.Context, key string) {
    s.logger.Debug(ctx, "Cache miss", map[string]interface{}{
        "cache_key": key,
        "cache_status": "miss",
    })
}
``` 