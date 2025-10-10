#!/usr/bin/env python3
"""
Fix Keycloak Roles - Update to correct role names
Roles: PlatformAdmin, OrgAdmin, Owner, User
"""

import requests
import sys

# ============================================================
# CONFIGURATION
# ============================================================
import os

KEYCLOAK_URL = os.getenv("KEYCLOAK_URL", "https://auth.rodolfodebonis.com.br")
REALM = os.getenv("KEYCLOAK_REALM", "spooliq")
ADMIN_EMAIL = os.getenv("KEYCLOAK_ADMIN_EMAIL")
ADMIN_PASSWORD = os.getenv("KEYCLOAK_ADMIN_PASSWORD")

# Validate required environment variables
if not ADMIN_EMAIL or not ADMIN_PASSWORD:
    print("❌ Error: KEYCLOAK_ADMIN_EMAIL and KEYCLOAK_ADMIN_PASSWORD environment variables are required")
    print("\nUsage:")
    print("  export KEYCLOAK_ADMIN_EMAIL=your_admin@email.com")
    print("  export KEYCLOAK_ADMIN_PASSWORD=your_password")
    print("  python3 scripts/fix_keycloak_roles.py")
    sys.exit(1)

print("=" * 60)
print("🔧 Fixing Keycloak Roles")
print("=" * 60)
print(f"Keycloak URL: {KEYCLOAK_URL}")
print(f"Realm: {REALM}")
print(f"User: {ADMIN_EMAIL}")
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
# STEP 2: Delete Old Roles (if they exist)
# ============================================================
print("\n🗑️  Removing old roles...")
roles_url = f"{KEYCLOAK_URL}/admin/realms/{REALM}/roles"

old_roles = ["user", "adm", "User"]
for role in old_roles:
    response = requests.delete(f"{roles_url}/{role}", headers=headers)
    if response.status_code == 204:
        print(f"   ✅ Deleted role '{role}'")
    elif response.status_code == 404:
        print(f"   ℹ️  Role '{role}' does not exist")
    else:
        print(f"   ⚠️  Failed to delete role '{role}': {response.status_code}")

# ============================================================
# STEP 3: Create Correct Roles
# ============================================================
print("\n👥 Creating correct realm roles...")

correct_roles = ["PlatformAdmin", "OrgAdmin", "Owner", "User"]
for role in correct_roles:
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
            print(f"       Response: {response.text}")

# ============================================================
# STEP 4: Update User Roles
# ============================================================
print(f"\n🎭 Updating user roles for {ADMIN_EMAIL}...")
users_url = f"{KEYCLOAK_URL}/admin/realms/{REALM}/users"

# Find user
response = requests.get(users_url, headers=headers, params={"email": ADMIN_EMAIL, "exact": "true"})
users = response.json()

if not users:
    print(f"❌ User not found: {ADMIN_EMAIL}")
    sys.exit(1)

user_id = users[0]["id"]
print(f"✅ User found (ID: {user_id})")

# Remove old roles
user_roles_url = f"{KEYCLOAK_URL}/admin/realms/{REALM}/users/{user_id}/role-mappings/realm"

# Get current roles
response = requests.get(user_roles_url, headers=headers)
current_roles = response.json()

if current_roles:
    print("   Removing old roles...")
    response = requests.delete(user_roles_url, headers=headers, json=current_roles)
    if response.status_code == 204:
        print("   ✅ Old roles removed")

# Assign new roles (PlatformAdmin, OrgAdmin, Owner for the main admin user)
print("   Assigning new roles...")
new_roles_to_assign = ["PlatformAdmin", "OrgAdmin", "Owner", "User"]
available_roles = []

for role in new_roles_to_assign:
    response = requests.get(f"{roles_url}/{role}", headers=headers)
    if response.status_code == 200:
        available_roles.append(response.json())

if available_roles:
    response = requests.post(user_roles_url, headers=headers, json=available_roles)
    if response.status_code in [204, 409]:
        print(f"   ✅ Roles assigned: {', '.join(new_roles_to_assign)}")
    else:
        print(f"   ⚠️  Role assignment returned: {response.status_code}")
        print(f"       Response: {response.text}")

# ============================================================
# SUCCESS SUMMARY
# ============================================================
print("\n" + "=" * 60)
print("✅ Keycloak roles fixed successfully!")
print("=" * 60)

print(f"""
📋 Updated Configuration:
   Realm: {REALM}
   User: {ADMIN_EMAIL}
   Roles: PlatformAdmin, OrgAdmin, Owner, User

📝 Next steps:

1. Update core/roles/roles.go to match:
   
   const (
       UserRole          = "User"
       OrgAdminRole      = "OrgAdmin"
       OwnerRole         = "Owner"
       PlatformAdminRole = "PlatformAdmin"
   )

2. Remove AdminRole = "adm" (no longer used)

3. Test login and verify JWT contains correct roles:
   
   curl -X POST http://localhost:8000/v1/login \\
     -H 'Content-Type: application/json' \\
     -d '{{"email":"{ADMIN_EMAIL}","password":"YOUR_PASSWORD"}}'

4. Decode the JWT token and verify:
   - "realm_access.roles" contains: ["PlatformAdmin", "OrgAdmin", "Owner", "User"]

5. Test company endpoint:
   
   curl -X GET http://localhost:8000/v1/company/ \\
     -H "Authorization: Bearer YOUR_TOKEN"

""" + "=" * 60)

