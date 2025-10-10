# Spooliq Scripts

## Keycloak Configuration Scripts

This directory contains Python scripts for automating Keycloak configuration for multi-tenancy support.

### Prerequisites

1. Python 3.x installed
2. Install required Python packages:

```bash
pip3 install -r requirements.txt
```

### Environment Variables

All scripts require the following environment variables to be set:

```bash
export KEYCLOAK_URL="https://auth.rodolfodebonis.com.br"
export KEYCLOAK_REALM="spooliq"
export KEYCLOAK_ADMIN_EMAIL="your_admin@email.com"
export KEYCLOAK_ADMIN_PASSWORD="your_password"
```

**Optional** (for initial setup scripts):
```bash
export SPOOLIQ_ORG_UUID="your-organization-uuid"  # If not provided, will be auto-generated
```

### Available Scripts

#### 1. `setup_keycloak_groups.py`

Sets up Keycloak for multi-tenancy using Groups approach. This is the **recommended** script for initial setup.

**Features:**
- Creates `spooliq` client (confidential)
- Creates realm roles: `PlatformAdmin`, `OrgAdmin`, `Owner`, `User`
- Creates `organization` client scope with group mapper
- Creates organization group with `organization_id` attribute
- Assigns user to group and roles

**Usage:**
```bash
export KEYCLOAK_ADMIN_EMAIL="admin@example.com"
export KEYCLOAK_ADMIN_PASSWORD="secure_password"
python3 scripts/setup_keycloak_groups.py
```

#### 2. `fix_keycloak_roles.py`

Fixes and updates Keycloak roles to match the SaaS requirements. Use this if you need to correct existing role names.

**Features:**
- Removes old/incorrect roles
- Creates correct roles: `PlatformAdmin`, `OrgAdmin`, `Owner`, `User`
- Updates user role assignments

**Usage:**
```bash
export KEYCLOAK_ADMIN_EMAIL="admin@example.com"
export KEYCLOAK_ADMIN_PASSWORD="secure_password"
python3 scripts/fix_keycloak_roles.py
```

#### 3. `setup_keycloak_multitenant.py`

Alternative setup script with more detailed configuration options.

**Usage:**
```bash
export KEYCLOAK_ADMIN_EMAIL="admin@example.com"
export KEYCLOAK_ADMIN_PASSWORD="secure_password"
python3 scripts/setup_keycloak_multitenant.py
```

#### 4. `check_user_realm.py`

Utility script to check which realm a user exists in and verify their configuration.

**Usage:**
```bash
export KEYCLOAK_ADMIN_EMAIL="admin@example.com"
export KEYCLOAK_ADMIN_PASSWORD="secure_password"
python3 scripts/check_user_realm.py
```

### Security Notes

⚠️ **IMPORTANT**: Never commit credentials to git!

- All scripts now use environment variables for sensitive data
- No hardcoded credentials in the codebase
- Use `.env` files locally (but never commit them)
- In production, use secure secret management (AWS Secrets Manager, Vault, etc.)

### Typical Setup Workflow

1. **First Time Setup:**
   ```bash
   # Set environment variables
   export KEYCLOAK_ADMIN_EMAIL="your_admin@email.com"
   export KEYCLOAK_ADMIN_PASSWORD="your_secure_password"
   
   # Run the main setup script
   python3 scripts/setup_keycloak_groups.py
   ```

2. **Save the Organization UUID:**
   The script will output an `organization_id` UUID. Save this value and insert it into your database:
   
   ```sql
   INSERT INTO companies (id, organization_id, name, email, phone, is_platform_company, subscription_status, created_at, updated_at)
   VALUES (
     gen_random_uuid(),
     'YOUR_ORGANIZATION_UUID_HERE',
     'Your Company Name',
     'contact@yourcompany.com',
     '+55 11 99999-9999',
     true,
     'permanent',
     NOW(),
     NOW()
   );
   ```

3. **Test the Setup:**
   ```bash
   # Login to get a token
   curl -X POST http://localhost:8000/v1/login \
     -H 'Content-Type: application/json' \
     -d '{"email":"your_admin@email.com","password":"your_password"}'
   
   # Decode the JWT at https://jwt.io and verify:
   # - organization_id is present
   # - roles include PlatformAdmin, OrgAdmin, Owner, User
   ```

4. **Access Protected Endpoints:**
   ```bash
   curl -X GET http://localhost:8000/v1/company/ \
     -H "Authorization: Bearer YOUR_TOKEN"
   ```

### Troubleshooting

**Problem:** "KEYCLOAK_ADMIN_EMAIL and KEYCLOAK_ADMIN_PASSWORD environment variables are required"

**Solution:** Make sure you've exported the environment variables in your current shell session.

---

**Problem:** "Authentication failed"

**Solution:** 
- Verify your Keycloak URL is correct
- Ensure your admin credentials are correct
- Check that the admin user has permissions in the master realm

---

**Problem:** "User not found"

**Solution:** 
- Make sure the user exists in the target realm (use `check_user_realm.py`)
- If user is in master realm, you need to create them in the target realm

---

**Problem:** "organization_id is null in JWT"

**Solution:**
- Verify the organization client scope is set as "Default" (not "Optional") in Keycloak
- Check that the user is added to the correct group
- Ensure the group has the organization_id attribute set

### Development Tips

**Using .env files (locally only):**

Create a `.env.local` file in the scripts directory (already in .gitignore):

```bash
export KEYCLOAK_URL="https://auth.rodolfodebonis.com.br"
export KEYCLOAK_REALM="spooliq"
export KEYCLOAK_ADMIN_EMAIL="your_admin@email.com"
export KEYCLOAK_ADMIN_PASSWORD="your_password"
export SPOOLIQ_ORG_UUID="550e8400-e29b-41d4-a716-446655440000"
```

Then source it before running scripts:
```bash
source scripts/.env.local
python3 scripts/setup_keycloak_groups.py
```

### Support

For more information, see:
- `docs/MULTI_TENANT_UUID_CHECKLIST.md` - Complete multi-tenancy guide
- `docs/ADMIN_ENDPOINTS_GUIDE.md` - Admin endpoints documentation

For issues, please open a GitHub issue or contact the development team.

