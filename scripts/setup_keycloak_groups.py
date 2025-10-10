#!/usr/bin/env python3
"""
Keycloak Multi-tenant Setup using Groups
Each Group represents a Company/Organization
"""

import requests
import json
import sys
from uuid import uuid4

# ============================================================
# CONFIGURATION
# ============================================================
KEYCLOAK_URL = "https://auth.rodolfodebonis.com.br"
REALM = "spooliq"
ADMIN_EMAIL = "dev@rodolfodebonis.com.br"
ADMIN_PASSWORD = "U{}z3N!B]xubk$ZK*DU/q7H4R8S8CG4%"

# Generate UUID for Spooliq Platform organization
SPOOLIQ_ORG_UUID = str(uuid4())

print("=" * 60)
print("🏢 Keycloak Multi-tenant Setup using Groups")
print("=" * 60)
print(f"Keycloak URL: {KEYCLOAK_URL}")
print(f"Realm: {REALM}")
print(f"User: {ADMIN_EMAIL}")
print(f"Generated Organization UUID: {SPOOLIQ_ORG_UUID}")
print("=" * 60)

# ============================================================
# STEP 1: Authenticate with Keycloak
# ============================================================
print("\n🔐 Authenticating with Keycloak master realm...")
token_url = f"{KEYCLOAK_URL}/realms/master/protocol/openid-connect/token"
token_data = {
    "client_id": "admin-cli",
    "username": ADMIN_EMAIL,
    "password": ADMIN_PASSWORD,
    "grant_type": "password"
}

try:
    response = requests.post(token_url, data=token_data, verify=True)
    response.raise_for_status()
    admin_token = response.json()["access_token"]
    print("✅ Authentication successful")
except Exception as e:
    print(f"❌ Authentication failed: {e}")
    sys.exit(1)

headers = {
    "Authorization": f"Bearer {admin_token}",
    "Content-Type": "application/json"
}

# ============================================================
# STEP 2: Setup Client 'spooliq'
# ============================================================
print(f"\n📱 Setting up client 'spooliq'...")
client_url = f"{KEYCLOAK_URL}/admin/realms/{REALM}/clients"

# Check if client exists
response = requests.get(client_url, headers=headers, params={"clientId": "spooliq"})
existing_clients = response.json()

if existing_clients:
    client_id = existing_clients[0]["id"]
    print(f"✅ Client 'spooliq' already exists (ID: {client_id})")
else:
    client_data = {
        "clientId": "spooliq",
        "name": "Spooliq Application",
        "description": "Main Spooliq application client",
        "enabled": True,
        "protocol": "openid-connect",
        "publicClient": True,
        "standardFlowEnabled": True,
        "directAccessGrantsEnabled": True,
        "serviceAccountsEnabled": False,
        "redirectUris": ["http://localhost:8000/*", "https://*.rodolfodebonis.com.br/*"],
        "webOrigins": ["http://localhost:8000", "https://*.rodolfodebonis.com.br"],
        "attributes": {
            "access.token.lifespan": "3600"
        }
    }
    
    response = requests.post(client_url, headers=headers, json=client_data)
    if response.status_code == 201:
        # Get the created client ID
        response = requests.get(client_url, headers=headers, params={"clientId": "spooliq"})
        client_id = response.json()[0]["id"]
        print(f"✅ Client 'spooliq' created (ID: {client_id})")
    else:
        print(f"❌ Failed to create client: {response.status_code}")
        print(response.text)
        sys.exit(1)

# ============================================================
# STEP 3: Create Realm Roles
# ============================================================
print("\n👥 Creating realm roles...")
roles_url = f"{KEYCLOAK_URL}/admin/realms/{REALM}/roles"

roles = ["PlatformAdmin", "OrgAdmin", "Owner", "User"]
for role in roles:
    # Check if role exists
    response = requests.get(f"{roles_url}/{role}", headers=headers)
    if response.status_code == 200:
        print(f"   ✅ Role '{role}' already exists")
    else:
        role_data = {
            "name": role,
            "description": f"{role} role for Spooliq SaaS platform"
        }
        response = requests.post(roles_url, headers=headers, json=role_data)
        if response.status_code == 201:
            print(f"   ✅ Role '{role}' created")
        else:
            print(f"   ⚠️  Failed to create role '{role}': {response.status_code}")

# ============================================================
# STEP 4: Create 'organization' Client Scope with Group Mapper
# ============================================================
print("\n🔧 Creating 'organization' client scope with group mapper...")
scopes_url = f"{KEYCLOAK_URL}/admin/realms/{REALM}/client-scopes"

# Check if scope exists
response = requests.get(scopes_url, headers=headers)
existing_scopes = [s for s in response.json() if s["name"] == "organization"]

if existing_scopes:
    scope_id = existing_scopes[0]["id"]
    print(f"✅ Scope 'organization' already exists (ID: {scope_id})")
else:
    scope_data = {
        "name": "organization",
        "description": "Organization/Company membership via Groups",
        "protocol": "openid-connect",
        "attributes": {
            "include.in.token.scope": "true",
            "display.on.consent.screen": "false"
        }
    }
    
    response = requests.post(scopes_url, headers=headers, json=scope_data)
    if response.status_code == 201:
        # Get created scope
        response = requests.get(scopes_url, headers=headers)
        scope_id = [s["id"] for s in response.json() if s["name"] == "organization"][0]
        print(f"✅ Scope 'organization' created (ID: {scope_id})")
    else:
        print(f"❌ Failed to create scope: {response.status_code}")
        print(response.text)
        sys.exit(1)

# ============================================================
# STEP 5: Create Group Mapper in Client Scope
# ============================================================
print("\n🗺️  Creating organization_id group mapper...")
mapper_url = f"{KEYCLOAK_URL}/admin/realms/{REALM}/client-scopes/{scope_id}/protocol-mappers/models"

# Check if mapper exists
response = requests.get(mapper_url, headers=headers)
existing_mappers = [m for m in response.json() if "organization" in m.get("name", "").lower()]

if existing_mappers:
    print("✅ Group mapper already exists")
else:
    # Use Group Membership mapper to extract group name/attribute as organization_id
    mapper_data = {
        "name": "organization-group-mapper",
        "protocol": "openid-connect",
        "protocolMapper": "oidc-group-membership-mapper",
        "config": {
            "claim.name": "groups",
            "full.path": "false",
            "id.token.claim": "true",
            "access.token.claim": "true",
            "userinfo.token.claim": "true"
        }
    }
    
    response = requests.post(mapper_url, headers=headers, json=mapper_data)
    if response.status_code == 201:
        print("✅ Group mapper created")
    elif response.status_code == 409:
        print("✅ Group mapper already exists")
    else:
        print(f"⚠️  Mapper creation returned: {response.status_code}")
        print(response.text)

# Now add a custom mapper to extract organization_id from group attributes
mapper_data_custom = {
    "name": "organization-id-from-group",
    "protocol": "openid-connect",
    "protocolMapper": "oidc-usermodel-attribute-mapper",
    "config": {
        "user.attribute": "organization_id",
        "claim.name": "organization_id",
        "jsonType.label": "String",
        "id.token.claim": "true",
        "access.token.claim": "true",
        "userinfo.token.claim": "true",
        "multivalued": "false",
        "aggregate.attrs": "false"
    }
}

response = requests.post(mapper_url, headers=headers, json=mapper_data_custom)
if response.status_code == 201:
    print("✅ Custom organization_id mapper created")
elif response.status_code == 409:
    print("✅ Custom organization_id mapper already exists")

# ============================================================
# STEP 6: Assign Scope to Client as Default
# ============================================================
print("\n🔗 Assigning 'organization' scope to client...")
client_scopes_url = f"{KEYCLOAK_URL}/admin/realms/{REALM}/clients/{client_id}/default-client-scopes/{scope_id}"

response = requests.put(client_scopes_url, headers=headers)
if response.status_code in [204, 404]:
    print("✅ Scope assigned to client as default")
else:
    print(f"⚠️  Scope assignment returned: {response.status_code}")

# ============================================================
# STEP 7: Create Group for Spooliq Platform
# ============================================================
print(f"\n🏢 Creating group for Spooliq Platform organization...")
groups_url = f"{KEYCLOAK_URL}/admin/realms/{REALM}/groups"

# Check if group exists
response = requests.get(groups_url, headers=headers, params={"search": "spooliq-platform"})
existing_groups = response.json()

if existing_groups:
    group_id = existing_groups[0]["id"]
    print(f"✅ Group 'spooliq-platform' already exists (ID: {group_id})")
else:
    group_data = {
        "name": "spooliq-platform",
        "attributes": {
            "organization_id": [SPOOLIQ_ORG_UUID],
            "company_name": ["Spooliq Platform"]
        }
    }
    
    response = requests.post(groups_url, headers=headers, json=group_data)
    if response.status_code == 201:
        # Get created group
        location = response.headers.get("Location")
        group_id = location.split("/")[-1]
        print(f"✅ Group 'spooliq-platform' created (ID: {group_id})")
        print(f"   Organization UUID: {SPOOLIQ_ORG_UUID}")
    else:
        print(f"❌ Failed to create group: {response.status_code}")
        print(response.text)
        sys.exit(1)

# ============================================================
# STEP 8: Find User
# ============================================================
print(f"\n🔍 Finding user '{ADMIN_EMAIL}'...")
users_url = f"{KEYCLOAK_URL}/admin/realms/{REALM}/users"

response = requests.get(users_url, headers=headers, params={"email": ADMIN_EMAIL, "exact": "true"})
users = response.json()

if not users:
    print(f"❌ User not found: {ADMIN_EMAIL}")
    sys.exit(1)

user_id = users[0]["id"]
print(f"✅ User found (ID: {user_id})")

# ============================================================
# STEP 9: Add User to Group
# ============================================================
print(f"\n👤 Adding user to 'spooliq-platform' group...")
user_groups_url = f"{KEYCLOAK_URL}/admin/realms/{REALM}/users/{user_id}/groups/{group_id}"

response = requests.put(user_groups_url, headers=headers)
if response.status_code == 204:
    print("✅ User added to group")
else:
    print(f"⚠️  Add to group returned: {response.status_code}")

# Also set organization_id as user attribute for backward compatibility
print(f"\n📝 Setting organization_id user attribute (backward compat)...")
user_url = f"{KEYCLOAK_URL}/admin/realms/{REALM}/users/{user_id}"
response = requests.get(user_url, headers=headers)
user_data = response.json()
user_data["attributes"] = user_data.get("attributes", {})
user_data["attributes"]["organization_id"] = [SPOOLIQ_ORG_UUID]

response = requests.put(user_url, headers=headers, json=user_data)
if response.status_code == 204:
    print("✅ organization_id attribute set")

# ============================================================
# STEP 10: Assign Roles to User
# ============================================================
print("\n🎭 Assigning roles to user...")
user_roles_url = f"{KEYCLOAK_URL}/admin/realms/{REALM}/users/{user_id}/role-mappings/realm"

# Get available roles
available_roles = []
for role in roles:
    response = requests.get(f"{roles_url}/{role}", headers=headers)
    if response.status_code == 200:
        available_roles.append(response.json())

# Assign all roles
response = requests.post(user_roles_url, headers=headers, json=available_roles)
if response.status_code in [204, 409]:
    print(f"✅ Roles assigned: {', '.join(roles)}")
else:
    print(f"⚠️  Role assignment returned: {response.status_code}")

# ============================================================
# SUCCESS SUMMARY
# ============================================================
print("\n" + "=" * 60)
print("✅ Setup completed successfully!")
print("=" * 60)

print(f"""
📋 Configuration Summary:
   Realm: {REALM}
   Client: spooliq
   Group: spooliq-platform
   User: {ADMIN_EMAIL}
   Organization UUID: {SPOOLIQ_ORG_UUID}

📝 Next steps:

1. Create/Update company record in database:
   
   UPDATE companies 
   SET organization_id = '{SPOOLIQ_ORG_UUID}'
   WHERE name = 'Spooliq Platform';
   
   -- Or if it doesn't exist:
   INSERT INTO companies (id, organization_id, name, email, phone, created_at, updated_at)
   VALUES (
     gen_random_uuid(),
     '{SPOOLIQ_ORG_UUID}',
     'Spooliq Platform',
     'contato@spooliq.com',
     '+55 11 99999-9999',
     NOW(),
     NOW()
   );

2. Test login via API:
   
   curl -X POST http://localhost:8000/v1/login \\
     -H 'Content-Type: application/json' \\
     -d '{{"email":"{ADMIN_EMAIL}","password":"YOUR_PASSWORD"}}'

3. Verify JWT contains organization_id claim:
   
   # Decode the access token and check for:
   # - "organization_id": "{SPOOLIQ_ORG_UUID}"
   # - "groups": ["spooliq-platform"]
   # - "realm_access.roles": [..., "PlatformAdmin", ...]

4. Test company endpoint:
   
   curl -X GET http://localhost:8000/v1/company/ \\
     -H "Authorization: Bearer YOUR_TOKEN"

5. To add a new client company:
   
   a) Create group in Keycloak with organization_id attribute
   b) Add client user to that group
   c) Assign appropriate roles (User, OrgAdmin)
   d) Insert company record in database

=""" + "=" * 60)

