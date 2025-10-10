# Keycloak Multi-Tenant Setup - Execution Instructions

## 📋 Overview

This guide provides step-by-step instructions for configuring Keycloak for UUID-based multi-tenancy in the Spooliq application.

**Time Required**: ~10-15 minutes  
**Difficulty**: Easy (automated via Python script)

## ✅ Prerequisites

Before starting, ensure you have:

- [x] Python 3.7+ installed
- [x] `pip` package manager
- [x] Access to Keycloak admin console (https://auth.rodolfodebonis.com.br)
- [x] PostgreSQL database access
- [x] Spooliq API running locally (or ready to start)

## 🚀 Quick Start

### Option A: Fully Automated (Recommended)

```bash
# 1. Install dependencies
cd /Users/rodolfodebonis/Documents/projects/spooliq\ copy
pip install -r scripts/requirements.txt

# 2. Run setup script
python3 scripts/setup_keycloak_multitenant.py

# 3. Copy the organization UUID from output
# Example: 550e8400-e29b-41d4-a716-446655440000

# 4. Insert company record in database
psql -h localhost -U user -d spooliq_db -c "
INSERT INTO companies (id, organization_id, name, email, created_at, updated_at)
VALUES (
  uuid_generate_v4(),
  '550e8400-e29b-41d4-a716-446655440000',
  'Spooliq Platform',
  'contato@spooliq.com',
  NOW(),
  NOW()
);"

# 5. Start/Restart the API
./spooliq

# 6. Test login
curl -X POST http://localhost:8000/v1/login \
  -H 'Content-Type: application/json' \
  -d '{
    "email": "dev@rodolfodebonis.com.br",
    "password": "U{}z3N!B]xubk$ZK*DU/q7H4R8S8CG4%"
  }' | jq '.'
```

### Option B: Step-by-Step with Verification

Follow the detailed instructions below.

---

## 📝 Detailed Step-by-Step Instructions

### Step 1: Prepare Environment

```bash
# Navigate to project directory
cd /Users/rodolfodebonis/Documents/projects/spooliq\ copy

# Verify Python version (3.7+)
python3 --version

# Install required packages
pip install -r scripts/requirements.txt

# Verify installation
python3 -c "import requests; print('✅ requests installed')"
```

**Expected Output**:
```
✅ requests installed
```

---

### Step 2: Run Keycloak Setup Script

```bash
python3 scripts/setup_keycloak_multitenant.py
```

**What the script does**:
1. ✅ Authenticates with Keycloak master realm
2. ✅ Creates/verifies client "spooliq" in realm "spooliq"
3. ✅ Creates realm roles: PlatformAdmin, OrgAdmin, User
4. ✅ Creates client scope "organization"
5. ✅ Creates protocol mapper "organization-id-mapper"
6. ✅ Assigns scope to client
7. ✅ Generates UUID for your organization
8. ✅ Finds user dev@rodolfodebonis.com.br
9. ✅ Sets organization_id attribute (UUID)
10. ✅ Assigns all three roles to user

**Expected Output**:
```
============================================================
🚀 Keycloak Multi-tenant Setup
============================================================
🔐 Authenticating with Keycloak master realm...
✅ Authentication successful

📱 Setting up client 'spooliq'...
✅ Client 'spooliq' created successfully (ID: abc123...)

👥 Creating realm roles...
   ✅ Role 'PlatformAdmin' created
   ✅ Role 'OrgAdmin' created
   ✅ Role 'User' created

🔧 Creating 'organization' client scope...
✅ Scope 'organization' created (ID: def456...)

🗺️  Creating organization_id protocol mapper...
✅ Protocol mapper created

🔗 Assigning 'organization' scope to client...
✅ Scope assigned to client as default

🔍 Finding user 'dev@rodolfodebonis.com.br'...
✅ User found (ID: ghi789...)

🏢 Setting organization_id attribute to: 550e8400-e29b-41d4-a716-446655440000
✅ organization_id attribute set

🎭 Assigning roles to user...
✅ Roles assigned: PlatformAdmin, OrgAdmin, User

============================================================
✅ Setup completed successfully!
============================================================

📋 Configuration Summary:
   Realm: spooliq
   Client: spooliq
   User: dev@rodolfodebonis.com.br
   Organization UUID: 550e8400-e29b-41d4-a716-446655440000

📝 Next steps:
   1. Create company record in database:
      INSERT INTO companies (id, organization_id, name, email)
      VALUES (
        uuid_generate_v4(),
        '550e8400-e29b-41d4-a716-446655440000',
        'Spooliq Platform',
        'contato@spooliq.com'
      );

   2. Test login via API:
      curl -X POST http://localhost:8000/v1/login \
        -H 'Content-Type: application/json' \
        -d '{"email":"dev@rodolfodebonis.com.br","password":"YOUR_PASSWORD"}'

   3. Verify JWT contains organization_id claim

   4. Test creating a company via POST /v1/company/

============================================================
```

**⚠️ IMPORTANT**: Copy the **Organization UUID** from the output! You'll need it in the next step.

---

### Step 3: Create Company Record in Database

Using the UUID from Step 2:

```bash
# Option A: Using psql
psql -h localhost -U user -d spooliq_db

# Then run this SQL (replace UUID):
INSERT INTO companies (id, organization_id, name, email, phone, created_at, updated_at)
VALUES (
  uuid_generate_v4(),
  '550e8400-e29b-41d4-a716-446655440000',  -- UUID from script output
  'Spooliq Platform',
  'contato@spooliq.com',
  '+55 11 99999-9999',
  NOW(),
  NOW()
);

# Verify insertion
SELECT id, organization_id, name, email FROM companies;
```

**Expected Output**:
```
                  id                  |           organization_id            |       name        |         email
--------------------------------------+--------------------------------------+-------------------+----------------------
 a1b2c3d4-e5f6-7890-abcd-ef1234567890 | 550e8400-e29b-41d4-a716-446655440000 | Spooliq Platform  | contato@spooliq.com
```

---

### Step 4: Start/Restart the API

```bash
# If API is not running
./spooliq

# If API is already running, restart it
# Find and kill the process
lsof -ti:8000 | xargs kill -9

# Start again
./spooliq
```

**Expected Output**:
```
[Fx] RUNNING
[GIN-debug] Listening and serving HTTP on :8000
2025-10-10T14:00:00.000-0300    INFO    app/init.go:51  Migrations done
```

---

### Step 5: Test Login

```bash
# Test login and save token
curl -X POST http://localhost:8000/v1/login \
  -H 'Content-Type: application/json' \
  -d '{
    "email": "dev@rodolfodebonis.com.br",
    "password": "U{}z3N!B]xubk$ZK*DU/q7H4R8S8CG4%"
  }' > response.json

# Extract and save token
TOKEN=$(cat response.json | jq -r '.accessToken')
echo "Token saved: $TOKEN"
```

**Expected Output**:
```json
{
  "accessToken": "eyJhbGciOiJSUzI1NiIsInR5cCIgO...",
  "expiresIn": 3600
}
```

---

### Step 6: Verify JWT Token Contains organization_id

```bash
# Decode JWT token (payload only)
echo $TOKEN | cut -d'.' -f2 | base64 -d 2>/dev/null | jq '.'
```

**Expected Claims** (verify these are present):
```json
{
  "exp": 1728577200,
  "iat": 1728573600,
  "email": "dev@rodolfodebonis.com.br",
  "email_verified": true,
  "organization_id": "550e8400-e29b-41d4-a716-446655440000",
  "realm_access": {
    "roles": [
      "PlatformAdmin",
      "OrgAdmin",
      "User",
      "uma_protection"
    ]
  }
}
```

**✅ Verify**:
- [ ] `organization_id` is present and is a valid UUID
- [ ] `realm_access.roles` includes "PlatformAdmin", "OrgAdmin", "User"
- [ ] `email` matches your user

---

### Step 7: Test Company Endpoint

```bash
# Get company information
curl -X GET http://localhost:8000/v1/company/ \
  -H "Authorization: Bearer $TOKEN" \
  | jq '.'
```

**Expected Response**:
```json
{
  "id": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
  "organization_id": "550e8400-e29b-41d4-a716-446655440000",
  "name": "Spooliq Platform",
  "email": "contato@spooliq.com",
  "phone": "+55 11 99999-9999",
  "created_at": "2025-10-10T14:00:00Z",
  "updated_at": "2025-10-10T14:00:00Z"
}
```

---

### Step 8: Verify Platform Admin Capabilities

Check the API logs for Platform Admin detection:

```bash
# In another terminal, tail the logs
tail -f /path/to/logs/spooliq.log

# Make a request and watch logs
curl -X GET http://localhost:8000/v1/company/ \
  -H "Authorization: Bearer $TOKEN"
```

**Expected Log Entries**:
```
2025-10-10T14:00:00.000-0300    INFO    middlewares/auth_middleware.go:120     Auth success
  {"role": "user", "user_roles": ["PlatformAdmin","OrgAdmin","User"], ...}
```

---

## ✅ Verification Checklist

After completing all steps, verify:

- [ ] Keycloak client "spooliq" exists in realm "spooliq"
- [ ] Realm roles created: PlatformAdmin, OrgAdmin, User
- [ ] Client scope "organization" created with mapper
- [ ] User has organization_id attribute (UUID format)
- [ ] User has all three roles assigned
- [ ] Company record exists in database with matching UUID
- [ ] Login returns valid JWT token
- [ ] JWT token contains organization_id claim
- [ ] Company endpoint returns your company data
- [ ] API logs show Platform Admin role detection

---

## 🔧 Troubleshooting

### Issue 1: Script fails with authentication error

**Error**: `❌ Authentication failed: 401 Unauthorized`

**Solution**:
1. Verify Keycloak URL is correct: https://auth.rodolfodebonis.com.br
2. Check username/password in script are correct
3. Ensure user exists in master realm with admin privileges

### Issue 2: User not found

**Error**: `❌ User 'dev@rodolfodebonis.com.br' not found`

**Solution**:
1. Verify user exists in Keycloak realm "spooliq" (not master)
2. Check email spelling is exact
3. Ensure realm name in script matches (spooliq)

### Issue 3: Company insertion fails

**Error**: `ERROR:  duplicate key value violates unique constraint "companies_organization_id_key"`

**Solution**:
- Company already exists for this organization_id
- Use GET /v1/company/ to retrieve it
- Or delete existing: `DELETE FROM companies WHERE organization_id = '...'`

### Issue 4: organization_id not in JWT token

**Error**: Login works but JWT doesn't contain organization_id

**Solution**:
1. Verify client scope "organization" is assigned as **Default** (not Optional)
2. Check protocol mapper exists and is correctly configured
3. Log out and log in again to get fresh token
4. Verify in Keycloak: Client → spooliq → Client Scopes tab

### Issue 5: Platform Admin detection not working

**Error**: Treated as regular user despite having role

**Solution**:
1. Verify role name is exactly "PlatformAdmin" (case-sensitive)
2. Check `user_roles` array in JWT includes "PlatformAdmin"
3. Restart API after role changes
4. Clear any token caches

---

## 🎯 Next Steps

After successful setup:

1. **Create test data**:
   - Brands: `POST /v1/brands`
   - Materials: `POST /v1/materials`
   - Filaments: `POST /v1/filaments`
   - Customers: `POST /v1/customers`

2. **Verify multi-tenancy**:
   - Check all created records have your organization_id
   - Query database to confirm

3. **Read documentation**:
   - [Multi-Tenant UUID Checklist](docs/MULTI_TENANT_UUID_CHECKLIST.md)
   - [Admin Endpoints Guide](docs/ADMIN_ENDPOINTS_GUIDE.md)

4. **Add client organizations** (when needed):
   - Follow guide in MULTI_TENANT_UUID_CHECKLIST.md
   - Section: "Adding New Client Organizations"

---

## 📞 Support

If you encounter issues not covered here:

1. Check application logs: `tail -f logs/spooliq.log`
2. Verify Keycloak configuration: https://auth.rodolfodebonis.com.br/admin
3. Check database state: Run verification SQL queries
4. Review [MULTI_TENANT_UUID_CHECKLIST.md](docs/MULTI_TENANT_UUID_CHECKLIST.md)

---

**Last Updated**: 2025-10-10  
**Script Version**: 1.0.0  
**Tested On**: macOS, Python 3.11

