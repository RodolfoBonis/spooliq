# Admin Endpoints Guide - Future Enhancement

## Overview

This document outlines the planned admin endpoints for Platform Admin users to manage multi-tenant organizations, view cross-organization data, and perform administrative tasks.

**Status**: 📋 **Planned** - Not yet implemented  
**Priority**: Medium  
**Estimated Effort**: 2-3 days

## Architecture

### Access Control

```go
// Middleware protection
adminRoutes := v1.Group("/admin")
adminRoutes.Use(middlewares.RequirePlatformAdmin()) // New middleware
{
    // All admin endpoints here
}
```

### Platform Admin Detection

```go
func RequirePlatformAdmin() gin.HandlerFunc {
    return func(c *gin.Context) {
        if !helpers.IsPlatformAdmin(c) {
            c.JSON(http.StatusForbidden, gin.H{
                "error": "Platform Admin role required"
            })
            c.Abort()
            return
        }
        c.Next()
    }
}
```

## Planned Endpoints

### 1. Organization Management

#### GET /v1/admin/organizations

List all organizations in the system.

**Request**:
```bash
curl -X GET http://localhost:8000/v1/admin/organizations \
  -H "Authorization: Bearer $PLATFORM_ADMIN_TOKEN"
```

**Response**:
```json
{
  "organizations": [
    {
      "organization_id": "550e8400-e29b-41d4-a716-446655440000",
      "company_name": "Spooliq Platform",
      "created_at": "2025-10-10T10:00:00Z",
      "user_count": 1,
      "budget_count": 15,
      "active": true
    },
    {
      "organization_id": "650e8400-e29b-41d4-a716-446655440001",
      "company_name": "Cliente ABC",
      "created_at": "2025-10-11T14:30:00Z",
      "user_count": 5,
      "budget_count": 42,
      "active": true
    }
  ],
  "total": 2
}
```

#### GET /v1/admin/organizations/:organization_id

Get detailed information about a specific organization.

**Request**:
```bash
curl -X GET http://localhost:8000/v1/admin/organizations/650e8400-... \
  -H "Authorization: Bearer $PLATFORM_ADMIN_TOKEN"
```

**Response**:
```json
{
  "organization_id": "650e8400-e29b-41d4-a716-446655440001",
  "company": {
    "id": "750e8400-...",
    "name": "Cliente ABC",
    "email": "contato@cliente-abc.com",
    "phone": "+55 11 98765-4321",
    "created_at": "2025-10-11T14:30:00Z"
  },
  "stats": {
    "users": 5,
    "customers": 120,
    "budgets": 42,
    "brands": 8,
    "materials": 15,
    "filaments": 35
  },
  "activity": {
    "last_login": "2025-10-12T09:15:00Z",
    "last_budget_created": "2025-10-12T08:45:00Z"
  }
}
```

#### POST /v1/admin/organizations

Create a new organization (generates UUID automatically).

**Request**:
```bash
curl -X POST http://localhost:8000/v1/admin/organizations \
  -H "Authorization: Bearer $PLATFORM_ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "company_name": "Novo Cliente XYZ",
    "admin_email": "admin@novo-cliente.com",
    "admin_name": "João Silva"
  }'
```

**Response**:
```json
{
  "organization_id": "850e8400-e29b-41d4-a716-446655440002",
  "company_id": "950e8400-e29b-41d4-a716-446655440003",
  "message": "Organization created successfully",
  "next_steps": [
    "1. User created in Keycloak with email: admin@novo-cliente.com",
    "2. Temporary password sent to admin email",
    "3. Organization ID assigned: 850e8400-...",
    "4. User must login and change password"
  ]
}
```

### 2. Cross-Organization Data Access

#### GET /v1/admin/budgets

View budgets from any organization.

**Query Parameters**:
- `organization_id` (optional): Filter by specific organization
- `status` (optional): Filter by budget status
- `page` (optional): Page number (default: 1)
- `page_size` (optional): Items per page (default: 20)

**Request**:
```bash
# All budgets
curl -X GET http://localhost:8000/v1/admin/budgets \
  -H "Authorization: Bearer $PLATFORM_ADMIN_TOKEN"

# Specific organization
curl -X GET "http://localhost:8000/v1/admin/budgets?organization_id=650e8400..." \
  -H "Authorization: Bearer $PLATFORM_ADMIN_TOKEN"

# Filter by status
curl -X GET "http://localhost:8000/v1/admin/budgets?status=approved" \
  -H "Authorization: Bearer $PLATFORM_ADMIN_TOKEN"
```

**Response**:
```json
{
  "budgets": [
    {
      "id": "a50e8400-...",
      "organization_id": "650e8400-...",
      "organization_name": "Cliente ABC",
      "customer_name": "Maria Santos",
      "total_cost": 1250.50,
      "status": "approved",
      "created_at": "2025-10-12T08:45:00Z"
    }
  ],
  "pagination": {
    "page": 1,
    "page_size": 20,
    "total": 157
  }
}
```

#### GET /v1/admin/customers

View customers from any organization.

**Query Parameters**:
- `organization_id` (optional): Filter by specific organization
- `search` (optional): Search by name or email
- `page`, `page_size`: Pagination

**Request**:
```bash
curl -X GET "http://localhost:8000/v1/admin/customers?organization_id=650e8400..." \
  -H "Authorization: Bearer $PLATFORM_ADMIN_TOKEN"
```

#### GET /v1/admin/companies

List all companies across organizations.

**Request**:
```bash
curl -X GET http://localhost:8000/v1/admin/companies \
  -H "Authorization: Bearer $PLATFORM_ADMIN_TOKEN"
```

### 3. Analytics and Reports

#### GET /v1/admin/analytics/overview

Global system analytics.

**Request**:
```bash
curl -X GET http://localhost:8000/v1/admin/analytics/overview \
  -H "Authorization: Bearer $PLATFORM_ADMIN_TOKEN"
```

**Response**:
```json
{
  "totals": {
    "organizations": 15,
    "users": 87,
    "customers": 1543,
    "budgets": 892,
    "total_revenue": 458932.50
  },
  "monthly_stats": {
    "budgets_created": 156,
    "budgets_approved": 98,
    "new_customers": 45,
    "active_organizations": 12
  },
  "top_organizations": [
    {
      "organization_id": "...",
      "name": "Cliente ABC",
      "budget_count": 156,
      "revenue": 125000.00
    }
  ]
}
```

#### GET /v1/admin/analytics/organization/:organization_id

Analytics for specific organization.

**Request**:
```bash
curl -X GET http://localhost:8000/v1/admin/analytics/organization/650e8400... \
  -H "Authorization: Bearer $PLATFORM_ADMIN_TOKEN"
```

### 4. User Management

#### GET /v1/admin/users

List users across all organizations.

**Query Parameters**:
- `organization_id` (optional): Filter by organization
- `role` (optional): Filter by role (User, OrgAdmin)
- `active` (optional): Filter by active status

**Request**:
```bash
curl -X GET "http://localhost:8000/v1/admin/users?organization_id=650e8400..." \
  -H "Authorization: Bearer $PLATFORM_ADMIN_TOKEN"
```

**Response**:
```json
{
  "users": [
    {
      "user_id": "a50e8400-...",
      "email": "admin@cliente-abc.com",
      "name": "João Silva",
      "organization_id": "650e8400-...",
      "organization_name": "Cliente ABC",
      "roles": ["OrgAdmin", "User"],
      "last_login": "2025-10-12T09:15:00Z",
      "active": true
    }
  ],
  "total": 87
}
```

### 5. System Operations

#### POST /v1/admin/organizations/:organization_id/disable

Disable an organization (soft delete).

**Request**:
```bash
curl -X POST http://localhost:8000/v1/admin/organizations/650e8400.../disable \
  -H "Authorization: Bearer $PLATFORM_ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "reason": "Payment overdue",
    "notify_users": true
  }'
```

#### POST /v1/admin/organizations/:organization_id/enable

Re-enable a disabled organization.

**Request**:
```bash
curl -X POST http://localhost:8000/v1/admin/organizations/650e8400.../enable \
  -H "Authorization: Bearer $PLATFORM_ADMIN_TOKEN"
```

## Implementation Plan

### Phase 1: Core Admin Endpoints

**Priority**: High  
**Effort**: 2 days

- [ ] Create admin middleware (RequirePlatformAdmin)
- [ ] Implement GET /v1/admin/organizations
- [ ] Implement GET /v1/admin/organizations/:id
- [ ] Implement GET /v1/admin/budgets with filters
- [ ] Add audit logging for all admin actions

### Phase 2: Analytics

**Priority**: Medium  
**Effort**: 1 day

- [ ] Implement GET /v1/admin/analytics/overview
- [ ] Implement GET /v1/admin/analytics/organization/:id
- [ ] Create dashboard-ready data aggregations

### Phase 3: Advanced Management

**Priority**: Low  
**Effort**: 2 days

- [ ] Implement POST /v1/admin/organizations (with Keycloak integration)
- [ ] Implement organization disable/enable
- [ ] Implement cross-organization user management
- [ ] Add bulk operations support

## Security Considerations

### Audit Logging

**All admin actions must be logged**:

```go
adminLogger.LogAdminAction(ctx, AdminAction{
    AdminUserID:        helpers.GetUserID(c),
    AdminEmail:         getUserEmail(c),
    Action:             "VIEW_ORGANIZATION_DATA",
    TargetOrgID:        organizationID,
    TargetResourceType: "budgets",
    TargetResourceID:   budgetID,
    IPAddress:          c.ClientIP(),
    Timestamp:          time.Now(),
    Success:            true,
})
```

### Rate Limiting

Admin endpoints should have separate, stricter rate limits:

```go
adminRoutes.Use(middlewares.RateLimit(100, time.Hour)) // 100 requests per hour
```

### IP Whitelisting (Optional)

```go
adminRoutes.Use(middlewares.IPWhitelist([]string{
    "203.0.113.0/24",  // Office network
    "198.51.100.45",    // VPN
}))
```

## Testing

### Integration Tests

```go
func TestAdminEndpoints(t *testing.T) {
    // Test Platform Admin can access
    // Test regular user cannot access (403)
    // Test organization isolation
    // Test filtering and pagination
}
```

### Manual Testing Checklist

- [ ] Platform Admin can access all admin endpoints
- [ ] Regular user (User role) gets 403 on admin endpoints
- [ ] OrgAdmin (without PlatformAdmin) gets 403 on admin endpoints
- [ ] Organization filtering works correctly
- [ ] Audit logs are created for all actions
- [ ] Pagination works correctly
- [ ] Data from different organizations is properly isolated

## Documentation Updates

When implementing, update:

1. **Swagger/OpenAPI**: Add admin endpoints to docs
2. **README**: Add admin features section
3. **API Documentation**: Create admin API guide
4. **Postman Collection**: Add admin endpoint examples

## Future Enhancements

- **Organization Settings**: Quotas, limits, feature flags per organization
- **Billing Integration**: Usage tracking, invoicing per organization
- **Automated Reports**: Email reports to Platform Admin
- **Organization Templates**: Pre-configured settings for new organizations
- **Data Migration Tools**: Move data between organizations
- **Backup/Restore**: Per-organization backup and restore capabilities

---

**Status**: 📋 Planned  
**Next Review**: When Phase 1 (Multi-tenancy) is complete and stable  
**Contact**: Platform Team

