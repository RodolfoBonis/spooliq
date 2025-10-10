# Multi-Tenant UUID-Based Organization ID - Complete Checklist

## Overview

This document provides a comprehensive checklist for implementing and verifying UUID-based multi-tenancy in the Spooliq application using Keycloak.

## Why UUID for organization_id?

### Advantages

✅ **Security**: UUIDs are not sequential or predictable  
✅ **Scalability**: Can be generated in distributed systems without conflicts  
✅ **Standard**: UUID is a native type in PostgreSQL  
✅ **Privacy**: Doesn't expose information about the organization  
✅ **International**: Works in any language/region  
✅ **Uniqueness**: Virtually guaranteed to be globally unique

### Example

```
Good: 550e8400-e29b-41d4-a716-446655440000
Bad:  org-spooliq-brasil
```

## Architecture Overview

```
┌─────────────────────────────────────────────────────────┐
│                       Keycloak                           │
│  Realm: spooliq                                          │
│  ├── Users                                               │
│  │   └── dev@rodolfodebonis.com.br                      │
│  │       ├── Attribute: organization_id (UUID)          │
│  │       └── Roles: [PlatformAdmin, OrgAdmin, User]    │
│  ├── Client: spooliq                                     │
│  └── Client Scope: organization                         │
│      └── Mapper: organization-id-mapper                 │
└─────────────────────────────────────────────────────────┘
                          ↓
                      JWT Token
        {
          "organization_id": "550e8400-...",
          "realm_access": {
            "roles": ["PlatformAdmin", "OrgAdmin", "User"]
          }
        }
                          ↓
┌─────────────────────────────────────────────────────────┐
│                  Spooliq API (Go)                        │
│  ├── Middleware extracts organization_id                │
│  ├── Use cases validate and filter by org_id            │
│  └── Database stores org_id as VARCHAR/UUID             │
└─────────────────────────────────────────────────────────┘
                          ↓
┌─────────────────────────────────────────────────────────┐
│                   PostgreSQL                             │
│  companies (organization_id UUID UNIQUE)                 │
│  customers (organization_id UUID, index)                 │
│  budgets   (organization_id UUID, index)                 │
│  brands    (organization_id UUID, index)                 │
│  materials (organization_id UUID, index)                 │
│  filaments (organization_id UUID, index)                 │
│  presets   (organization_id UUID, index)                 │
└─────────────────────────────────────────────────────────┘
```

## Implementation Checklist

### Phase 1: Keycloak Configuration

- [ ] **Install Python dependencies**
  ```bash
  pip install -r scripts/requirements.txt
  ```

- [ ] **Run Keycloak setup script**
  ```bash
  python3 scripts/setup_keycloak_multitenant.py
  ```

- [ ] **Verify script output**
  - Client "spooliq" created
  - Roles created: PlatformAdmin, OrgAdmin, User
  - Client scope "organization" created
  - Protocol mapper "organization-id-mapper" created
  - User found and updated
  - organization_id attribute set (UUID)
  - Roles assigned to user

- [ ] **Copy generated UUID**
  - Note: The script outputs the UUID at the end
  - Example: `550e8400-e29b-41d4-a716-446655440000`

### Phase 2: Database Setup

- [ ] **Create company record**
  ```sql
  INSERT INTO companies (id, organization_id, name, email, created_at, updated_at)
  VALUES (
    uuid_generate_v4(),
    '550e8400-e29b-41d4-a716-446655440000', -- UUID from script
    'Spooliq Platform',
    'contato@spooliq.com',
    NOW(),
    NOW()
  );
  ```

- [ ] **Verify company inserted**
  ```sql
  SELECT * FROM companies WHERE organization_id = '550e8400-e29b-41d4-a716-446655440000';
  ```

### Phase 3: Application Code Verification

- [ ] **Code changes applied**
  - `core/helpers/context_helpers.go` - IsPlatformAdmin() added
  - `core/roles/roles.go` - PlatformAdminRole constant added
  - `features/company/domain/usecases/create_company_uc.go` - Platform Admin logic

- [ ] **Compile application**
  ```bash
  go build -o spooliq .
  ```

- [ ] **Run application**
  ```bash
  ./spooliq
  ```

### Phase 4: Testing

#### 4.1 Test Login

- [ ] **Login via API**
  ```bash
  curl -X POST http://localhost:8000/v1/login \
    -H 'Content-Type: application/json' \
    -d '{
      "email": "dev@rodolfodebonis.com.br",
      "password": "YOUR_PASSWORD"
    }'
  ```

- [ ] **Save access token**
  ```bash
  TOKEN=$(curl -s -X POST http://localhost:8000/v1/login \
    -H 'Content-Type: application/json' \
    -d '{...}' | jq -r '.accessToken')
  ```

#### 4.2 Verify JWT Token

- [ ] **Decode JWT token** (online tool or command line)
  ```bash
  echo $TOKEN | cut -d'.' -f2 | base64 -d 2>/dev/null | jq '.'
  ```

- [ ] **Verify claims present**
  - `organization_id`: UUID format
  - `realm_access.roles`: includes "PlatformAdmin", "OrgAdmin", "User"
  - `email`: dev@rodolfodebonis.com.br

#### 4.3 Test Company Endpoint

- [ ] **Get existing company**
  ```bash
  curl -X GET http://localhost:8000/v1/company/ \
    -H "Authorization: Bearer $TOKEN"
  ```

- [ ] **Verify response**
  - Should return the company you created
  - `organization_id` should match the UUID

#### 4.4 Test Platform Admin Capabilities

- [ ] **Verify IsPlatformAdmin returns true**
  - Check application logs for "Platform Admin" messages
  - Should see logs indicating Platform Admin role detected

#### 4.5 Test Multi-Tenancy Isolation

- [ ] **Create test data (brands, materials, etc.)**
  ```bash
  curl -X POST http://localhost:8000/v1/brands \
    -H "Authorization: Bearer $TOKEN" \
    -H 'Content-Type: application/json' \
    -d '{"name": "Test Brand"}'
  ```

- [ ] **Verify organization_id is set automatically**
  - Query database directly
  - Check that all created records have your organization_id

## Adding New Client Organizations

### For Platform Admin

1. **Create company via API**
   ```bash
   curl -X POST http://localhost:8000/v1/company/ \
     -H "Authorization: Bearer $TOKEN" \
     -H 'Content-Type: application/json' \
     -d '{
       "name": "Cliente XYZ",
       "email": "admin@cliente-xyz.com"
     }'
   ```

2. **Note the generated organization_id from response**

3. **Create user in Keycloak**
   - Go to Keycloak Admin Console
   - Select realm "spooliq"
   - Users → Add user
   - Email: admin@cliente-xyz.com
   - Email Verified: ON
   - Save

4. **Set user password**
   - Credentials tab
   - Set Password
   - Temporary: OFF (optional)

5. **Add organization_id attribute**
   - Attributes tab
   - Key: `organization_id`
   - Value: `[UUID from step 2]`
   - Save

6. **Assign roles**
   - Role Mappings tab
   - Assign: OrgAdmin, User
   - (Do NOT assign PlatformAdmin to regular clients)

7. **Test client login**
   ```bash
   curl -X POST http://localhost:8000/v1/login \
     -H 'Content-Type: application/json' \
     -d '{
       "email": "admin@cliente-xyz.com",
       "password": "CLIENT_PASSWORD"
     }'
   ```

## Troubleshooting

### Issue: organization_id not in JWT

**Symptom**: "Organization ID not found in context" error

**Solution**:
1. Check user has `organization_id` attribute in Keycloak
2. Verify client scope "organization" is assigned to client as **Default**
3. Check protocol mapper is correctly configured
4. Re-login to get fresh token

### Issue: PlatformAdmin detection not working

**Symptom**: User treated as regular user despite having role

**Solution**:
1. Verify user has "PlatformAdmin" role (exact spelling, case-sensitive)
2. Check `user_roles` is being set in middleware
3. Verify `helpers.IsPlatformAdmin()` is being called
4. Check application logs for role detection messages

### Issue: Company already exists error

**Symptom**: 409 Conflict when creating company

**Solution**:
- Each organization_id can have only ONE company
- Use GET /v1/company/ to retrieve existing
- Use PUT /v1/company/ to update existing

### Issue: Multi-tenancy not working

**Symptom**: Users see data from other organizations

**Solution**:
1. Verify all tables have `organization_id` column
2. Check all repository methods filter by `organization_id`
3. Ensure all use cases extract `organization_id` from context
4. Verify migrations were applied

## Database Verification Queries

```sql
-- Check company exists
SELECT * FROM companies WHERE organization_id = 'YOUR-UUID';

-- Verify all tables have organization_id
SELECT table_name, column_name 
FROM information_schema.columns 
WHERE column_name = 'organization_id';

-- Count records per organization
SELECT organization_id, COUNT(*) 
FROM customers 
GROUP BY organization_id;

-- Check for orphaned records (without org_id - should be empty)
SELECT 'customers' as table_name, COUNT(*) as count FROM customers WHERE organization_id IS NULL OR organization_id = ''
UNION ALL
SELECT 'budgets', COUNT(*) FROM budgets WHERE organization_id IS NULL OR organization_id = ''
UNION ALL
SELECT 'brands', COUNT(*) FROM brands WHERE organization_id IS NULL OR organization_id = '';
```

## Security Best Practices

1. **Never expose organization_id in URLs**
   - ❌ Bad: `/v1/admin/companies/{org_id}`
   - ✅ Good: Extract from JWT context

2. **Always validate organization_id in use cases**
   ```go
   organizationID := helpers.GetOrganizationID(c)
   if organizationID == "" && !helpers.IsPlatformAdmin(c) {
       return error
   }
   ```

3. **Filter all database queries by organization_id**
   ```go
   db.Where("organization_id = ?", organizationID).Find(&records)
   ```

4. **Log organization_id in all operations**
   ```go
   logger.Info("Operation performed", map[string]interface{}{
       "organization_id": organizationID,
       "user_id": userID,
   })
   ```

5. **Platform Admin audit trail**
   - Always log when Platform Admin acts on behalf of another org
   - Include original org_id and target org_id

## Future Enhancements

### Admin Endpoints (Planned)

```
GET    /v1/admin/organizations           - List all organizations
GET    /v1/admin/organizations/:id       - Get organization details
POST   /v1/admin/organizations           - Create new organization
GET    /v1/admin/budgets?org_id=X        - View any org's budgets
GET    /v1/admin/users?org_id=X          - View any org's users
GET    /v1/admin/analytics                - Global analytics dashboard
```

### Additional Features

- Organization switching for Platform Admin (UI feature)
- Organization usage metrics and quotas
- Automated client onboarding workflow
- Organization settings and preferences
- Billing and subscription management per organization

## Support

For issues or questions:
- Check application logs: `tail -f logs/spooliq.log`
- Keycloak Admin Console: https://auth.rodolfodebonis.com.br/admin
- Database queries: Connect to PostgreSQL and run verification queries

---

**Last Updated**: 2025-10-10  
**Version**: 1.0.0

