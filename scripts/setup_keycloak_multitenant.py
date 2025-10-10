#!/usr/bin/env python3
"""
Keycloak Multi-tenant Setup Script

This script automatically configures Keycloak realm 'spooliq' with:
- Client configuration
- Roles (PlatformAdmin, OrgAdmin, User)
- Client scope for organization_id
- User attributes and role assignments
"""

import sys
import uuid as uuid_lib
import requests
from typing import Dict, Optional
import json
import os

# Configuration - Load from environment variables
KEYCLOAK_URL = os.getenv("KEYCLOAK_URL", "https://auth.rodolfodebonis.com.br")
REALM_NAME = os.getenv("KEYCLOAK_REALM", "spooliq")
ADMIN_EMAIL = os.getenv("KEYCLOAK_ADMIN_EMAIL")
ADMIN_PASSWORD = os.getenv("KEYCLOAK_ADMIN_PASSWORD")
CLIENT_ID = "spooliq"

# Validate required environment variables
if not ADMIN_EMAIL or not ADMIN_PASSWORD:
    print("❌ Error: KEYCLOAK_ADMIN_EMAIL and KEYCLOAK_ADMIN_PASSWORD environment variables are required")
    print("\nUsage:")
    print("  export KEYCLOAK_ADMIN_EMAIL=your_admin@email.com")
    print("  export KEYCLOAK_ADMIN_PASSWORD=your_password")
    print("  python3 scripts/setup_keycloak_multitenant.py")
    sys.exit(1)


class KeycloakSetup:
    def __init__(self):
        self.base_url = KEYCLOAK_URL
        self.realm = REALM_NAME
        self.token = None
        self.session = requests.Session()
        self.session.verify = True  # Enable SSL verification
        self.organization_uuid = str(uuid_lib.uuid4())
        
    def get_admin_token(self) -> bool:
        """Authenticate with Keycloak master realm and get admin token."""
        print("🔐 Authenticating with Keycloak master realm...")
        
        url = f"{self.base_url}/realms/master/protocol/openid-connect/token"
        data = {
            "client_id": "admin-cli",
            "username": ADMIN_EMAIL,
            "password": ADMIN_PASSWORD,
            "grant_type": "password"
        }
        
        try:
            response = self.session.post(url, data=data)
            response.raise_for_status()
            self.token = response.json()["access_token"]
            self.session.headers.update({
                "Authorization": f"Bearer {self.token}",
                "Content-Type": "application/json"
            })
            print("✅ Authentication successful")
            return True
        except requests.exceptions.RequestException as e:
            print(f"❌ Authentication failed: {e}")
            if hasattr(e, 'response') and e.response:
                print(f"   Response: {e.response.text}")
            return False
    
    def create_client(self) -> Optional[str]:
        """Create or get the spooliq client."""
        print(f"\n📱 Setting up client '{CLIENT_ID}'...")
        
        # Check if client exists
        url = f"{self.base_url}/admin/realms/{self.realm}/clients"
        try:
            response = self.session.get(url, params={"clientId": CLIENT_ID})
            response.raise_for_status()
            clients = response.json()
            
            if clients:
                client_uuid = clients[0]["id"]
                print(f"✅ Client '{CLIENT_ID}' already exists (ID: {client_uuid})")
                return client_uuid
        except requests.exceptions.RequestException as e:
            print(f"⚠️  Error checking client: {e}")
        
        # Create new client
        client_config = {
            "clientId": CLIENT_ID,
            "name": "Spooliq Application",
            "protocol": "openid-connect",
            "publicClient": True,
            "standardFlowEnabled": True,
            "directAccessGrantsEnabled": True,
            "redirectUris": ["http://localhost:8000/*"],
            "webOrigins": ["http://localhost:8000"],
            "enabled": True
        }
        
        try:
            response = self.session.post(url, json=client_config)
            response.raise_for_status()
            
            # Get the created client's UUID
            response = self.session.get(url, params={"clientId": CLIENT_ID})
            response.raise_for_status()
            client_uuid = response.json()[0]["id"]
            
            print(f"✅ Client '{CLIENT_ID}' created successfully (ID: {client_uuid})")
            return client_uuid
        except requests.exceptions.RequestException as e:
            print(f"❌ Failed to create client: {e}")
            if hasattr(e, 'response') and e.response:
                print(f"   Response: {e.response.text}")
            return None
    
    def create_roles(self) -> bool:
        """Create realm roles."""
        print("\n👥 Creating realm roles...")
        
        roles = ["PlatformAdmin", "OrgAdmin", "User"]
        url = f"{self.base_url}/admin/realms/{self.realm}/roles"
        
        success = True
        for role_name in roles:
            # Check if role exists
            try:
                check_response = self.session.get(f"{url}/{role_name}")
                if check_response.status_code == 200:
                    print(f"   ✅ Role '{role_name}' already exists")
                    continue
            except:
                pass
            
            # Create role
            role_config = {
                "name": role_name,
                "description": f"{role_name} role for Spooliq multi-tenant system"
            }
            
            try:
                response = self.session.post(url, json=role_config)
                response.raise_for_status()
                print(f"   ✅ Role '{role_name}' created")
            except requests.exceptions.RequestException as e:
                print(f"   ❌ Failed to create role '{role_name}': {e}")
                success = False
        
        return success
    
    def create_client_scope(self) -> Optional[str]:
        """Create organization client scope."""
        print("\n🔧 Creating 'organization' client scope...")
        
        url = f"{self.base_url}/admin/realms/{self.realm}/client-scopes"
        
        # Check if scope exists
        try:
            response = self.session.get(url)
            response.raise_for_status()
            for scope in response.json():
                if scope["name"] == "organization":
                    scope_id = scope["id"]
                    print(f"✅ Scope 'organization' already exists (ID: {scope_id})")
                    return scope_id
        except:
            pass
        
        # Create scope
        scope_config = {
            "name": "organization",
            "description": "Organization multi-tenancy scope",
            "protocol": "openid-connect",
            "attributes": {
                "include.in.token.scope": "true",
                "display.on.consent.screen": "false"
            }
        }
        
        try:
            response = self.session.post(url, json=scope_config)
            response.raise_for_status()
            
            # Get the created scope's UUID
            response = self.session.get(url)
            response.raise_for_status()
            for scope in response.json():
                if scope["name"] == "organization":
                    scope_id = scope["id"]
                    print(f"✅ Scope 'organization' created (ID: {scope_id})")
                    return scope_id
        except requests.exceptions.RequestException as e:
            print(f"❌ Failed to create scope: {e}")
            return None
    
    def create_protocol_mapper(self, scope_id: str) -> bool:
        """Create organization_id protocol mapper."""
        print("\n🗺️  Creating organization_id protocol mapper...")
        
        url = f"{self.base_url}/admin/realms/{self.realm}/client-scopes/{scope_id}/protocol-mappers/models"
        
        # Check if mapper already exists
        try:
            response = self.session.get(url)
            response.raise_for_status()
            mappers = response.json()
            for mapper in mappers:
                if mapper.get("name") == "organization-id-mapper":
                    print("✅ Protocol mapper already exists")
                    return True
        except:
            pass
        
        mapper_config = {
            "name": "organization-id-mapper",
            "protocol": "openid-connect",
            "protocolMapper": "oidc-usermodel-attribute-mapper",
            "config": {
                "user.attribute": "organization_id",
                "claim.name": "organization_id",
                "jsonType.label": "String",
                "id.token.claim": "true",
                "access.token.claim": "true",
                "userinfo.token.claim": "true"
            }
        }
        
        try:
            response = self.session.post(url, json=mapper_config)
            response.raise_for_status()
            print("✅ Protocol mapper created")
            return True
        except requests.exceptions.RequestException as e:
            if e.response and e.response.status_code == 409:
                print("✅ Protocol mapper already exists")
                return True
            print(f"❌ Failed to create mapper: {e}")
            return False
    
    def assign_scope_to_client(self, client_uuid: str, scope_id: str) -> bool:
        """Assign organization scope to spooliq client."""
        print("\n🔗 Assigning 'organization' scope to client...")
        
        url = f"{self.base_url}/admin/realms/{self.realm}/clients/{client_uuid}/default-client-scopes/{scope_id}"
        
        try:
            response = self.session.put(url)
            response.raise_for_status()
            print("✅ Scope assigned to client as default")
            return True
        except requests.exceptions.RequestException as e:
            print(f"❌ Failed to assign scope: {e}")
            return False
    
    def find_user_by_email(self, email: str) -> Optional[str]:
        """Find user by email."""
        print(f"\n🔍 Finding user '{email}'...")
        
        url = f"{self.base_url}/admin/realms/{self.realm}/users"
        
        try:
            response = self.session.get(url, params={"email": email, "exact": "true"})
            response.raise_for_status()
            users = response.json()
            
            if users:
                user_id = users[0]["id"]
                print(f"✅ User found (ID: {user_id})")
                return user_id
            else:
                print(f"❌ User '{email}' not found")
                return None
        except requests.exceptions.RequestException as e:
            print(f"❌ Error finding user: {e}")
            return None
    
    def set_user_attribute(self, user_id: str) -> bool:
        """Set organization_id attribute on user."""
        print(f"\n🏢 Setting organization_id attribute to: {self.organization_uuid}")
        
        url = f"{self.base_url}/admin/realms/{self.realm}/users/{user_id}"
        
        # Get current user data
        try:
            response = self.session.get(url)
            response.raise_for_status()
            user_data = response.json()
            
            # Update attributes
            if "attributes" not in user_data:
                user_data["attributes"] = {}
            
            user_data["attributes"]["organization_id"] = [self.organization_uuid]
            
            # Update user
            response = self.session.put(url, json=user_data)
            response.raise_for_status()
            print("✅ organization_id attribute set")
            return True
        except requests.exceptions.RequestException as e:
            print(f"❌ Failed to set attribute: {e}")
            if hasattr(e, 'response') and e.response:
                print(f"   Response: {e.response.text}")
            return False
    
    def assign_roles_to_user(self, user_id: str) -> bool:
        """Assign PlatformAdmin, OrgAdmin, and User roles to user."""
        print("\n🎭 Assigning roles to user...")
        
        roles_to_assign = ["PlatformAdmin", "OrgAdmin", "User"]
        
        # Get realm roles
        url = f"{self.base_url}/admin/realms/{self.realm}/roles"
        try:
            response = self.session.get(url)
            response.raise_for_status()
            all_roles = response.json()
            
            # Find role objects
            role_objects = []
            for role_name in roles_to_assign:
                for role in all_roles:
                    if role["name"] == role_name:
                        role_objects.append({
                            "id": role["id"],
                            "name": role["name"]
                        })
                        break
            
            if len(role_objects) != len(roles_to_assign):
                print(f"❌ Not all roles found. Found {len(role_objects)} of {len(roles_to_assign)}")
                return False
            
            # Assign roles
            assign_url = f"{self.base_url}/admin/realms/{self.realm}/users/{user_id}/role-mappings/realm"
            response = self.session.post(assign_url, json=role_objects)
            response.raise_for_status()
            
            print("✅ Roles assigned: PlatformAdmin, OrgAdmin, User")
            return True
        except requests.exceptions.RequestException as e:
            print(f"❌ Failed to assign roles: {e}")
            if hasattr(e, 'response') and e.response:
                print(f"   Response: {e.response.text}")
            return False
    
    def run(self) -> bool:
        """Run the complete setup."""
        print("=" * 60)
        print("🚀 Keycloak Multi-tenant Setup")
        print("=" * 60)
        
        # Step 1: Authenticate
        if not self.get_admin_token():
            return False
        
        # Step 2: Create client
        client_uuid = self.create_client()
        if not client_uuid:
            return False
        
        # Step 3: Create roles
        if not self.create_roles():
            return False
        
        # Step 4: Create client scope
        scope_id = self.create_client_scope()
        if not scope_id:
            return False
        
        # Step 5: Create protocol mapper
        if not self.create_protocol_mapper(scope_id):
            return False
        
        # Step 6: Assign scope to client
        if not self.assign_scope_to_client(client_uuid, scope_id):
            return False
        
        # Step 7: Find user
        user_id = self.find_user_by_email(ADMIN_EMAIL)
        if not user_id:
            return False
        
        # Step 8: Set user attribute
        if not self.set_user_attribute(user_id):
            return False
        
        # Step 9: Assign roles
        if not self.assign_roles_to_user(user_id):
            return False
        
        # Success summary
        print("\n" + "=" * 60)
        print("✅ Setup completed successfully!")
        print("=" * 60)
        print(f"\n📋 Configuration Summary:")
        print(f"   Realm: {self.realm}")
        print(f"   Client: {CLIENT_ID}")
        print(f"   User: {ADMIN_EMAIL}")
        print(f"   Organization UUID: {self.organization_uuid}")
        print(f"\n📝 Next steps:")
        print(f"   1. Create company record in database:")
        print(f"      INSERT INTO companies (id, organization_id, name, email)")
        print(f"      VALUES (")
        print(f"        uuid_generate_v4(),")
        print(f"        '{self.organization_uuid}',")
        print(f"        'Spooliq Platform',")
        print(f"        'contato@spooliq.com'")
        print(f"      );")
        print(f"\n   2. Test login via API:")
        print(f"      curl -X POST http://localhost:8000/v1/login \\")
        print(f"        -H 'Content-Type: application/json' \\")
        print(f"        -d '{{\"email\":\"{ADMIN_EMAIL}\",\"password\":\"YOUR_PASSWORD\"}}'")
        print(f"\n   3. Verify JWT contains organization_id claim")
        print(f"\n   4. Test creating a company via POST /v1/company/")
        print("\n" + "=" * 60)
        
        return True


def main():
    """Main entry point."""
    setup = KeycloakSetup()
    
    try:
        success = setup.run()
        sys.exit(0 if success else 1)
    except KeyboardInterrupt:
        print("\n\n⚠️  Setup interrupted by user")
        sys.exit(1)
    except Exception as e:
        print(f"\n\n❌ Unexpected error: {e}")
        import traceback
        traceback.print_exc()
        sys.exit(1)


if __name__ == "__main__":
    main()

