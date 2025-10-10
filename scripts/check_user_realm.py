#!/usr/bin/env python3
"""
Check which realm the user exists in and provide guidance.
"""

import requests
import sys

KEYCLOAK_URL = "https://auth.rodolfodebonis.com.br"
ADMIN_EMAIL = "dev@rodolfodebonis.com.br"
ADMIN_PASSWORD = "U{}z3N!B]xubk$ZK*DU/q7H4R8S8CG4%"

def get_admin_token():
    """Get admin token from master realm."""
    url = f"{KEYCLOAK_URL}/realms/master/protocol/openid-connect/token"
    data = {
        "client_id": "admin-cli",
        "username": ADMIN_EMAIL,
        "password": ADMIN_PASSWORD,
        "grant_type": "password"
    }
    
    response = requests.post(url, data=data, verify=True)
    response.raise_for_status()
    return response.json()["access_token"]

def check_user_in_realm(token, realm, email):
    """Check if user exists in a specific realm."""
    url = f"{KEYCLOAK_URL}/admin/realms/{realm}/users"
    headers = {
        "Authorization": f"Bearer {token}",
        "Content-Type": "application/json"
    }
    
    response = requests.get(url, headers=headers, params={"email": email, "exact": "true"})
    
    if response.status_code == 200:
        users = response.json()
        return users[0] if users else None
    return None

def create_user_in_spooliq(token, email):
    """Create user in spooliq realm."""
    url = f"{KEYCLOAK_URL}/admin/realms/spooliq/users"
    headers = {
        "Authorization": f"Bearer {token}",
        "Content-Type": "application/json"
    }
    
    user_data = {
        "username": email,
        "email": email,
        "emailVerified": True,
        "enabled": True,
        "firstName": "Admin",
        "lastName": "Platform"
    }
    
    response = requests.post(url, headers=headers, json=user_data)
    
    if response.status_code == 201:
        # Get user ID from location header
        location = response.headers.get("Location")
        if location:
            user_id = location.split("/")[-1]
            return user_id
        
        # Fallback: search for user
        user = check_user_in_realm(token, "spooliq", email)
        return user["id"] if user else None
    
    return None

def set_user_password(token, user_id, password):
    """Set password for user."""
    url = f"{KEYCLOAK_URL}/admin/realms/spooliq/users/{user_id}/reset-password"
    headers = {
        "Authorization": f"Bearer {token}",
        "Content-Type": "application/json"
    }
    
    password_data = {
        "type": "password",
        "value": password,
        "temporary": False
    }
    
    response = requests.put(url, headers=headers, json=password_data)
    return response.status_code == 204

def main():
    print("🔍 Verificando usuário em diferentes realms...")
    
    try:
        token = get_admin_token()
        print("✅ Autenticado com sucesso\n")
        
        # Check master realm
        master_user = check_user_in_realm(token, "master", ADMIN_EMAIL)
        if master_user:
            print(f"✅ Usuário encontrado no realm 'master'")
            print(f"   ID: {master_user['id']}")
            print(f"   Username: {master_user.get('username', 'N/A')}")
        else:
            print("❌ Usuário NÃO encontrado no realm 'master'")
        
        # Check spooliq realm
        spooliq_user = check_user_in_realm(token, "spooliq", ADMIN_EMAIL)
        if spooliq_user:
            print(f"\n✅ Usuário encontrado no realm 'spooliq'")
            print(f"   ID: {spooliq_user['id']}")
            print(f"   Username: {spooliq_user.get('username', 'N/A')}")
            return 0
        else:
            print(f"\n❌ Usuário NÃO encontrado no realm 'spooliq'")
            print(f"\n🔧 Criando usuário no realm 'spooliq'...")
            
            user_id = create_user_in_spooliq(token, ADMIN_EMAIL)
            if user_id:
                print(f"✅ Usuário criado com sucesso!")
                print(f"   ID: {user_id}")
                
                # Set password
                print(f"\n🔑 Configurando senha...")
                if set_user_password(token, user_id, ADMIN_PASSWORD):
                    print("✅ Senha configurada com sucesso!")
                else:
                    print("⚠️  Falha ao configurar senha. Configure manualmente no Keycloak.")
                
                print(f"\n✅ Agora execute novamente o script de setup:")
                print(f"   python3 scripts/setup_keycloak_multitenant.py")
                return 0
            else:
                print("❌ Falha ao criar usuário")
                return 1
        
    except Exception as e:
        print(f"\n❌ Erro: {e}")
        import traceback
        traceback.print_exc()
        return 1

if __name__ == "__main__":
    sys.exit(main())

