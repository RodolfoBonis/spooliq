# Configuração Rápida do Keycloak para Multi-tenancy

## 🔧 PASSOS NECESSÁRIOS

### 1. Adicionar Atributo ao Usuário

1. Acesse o Keycloak Admin Console
2. Vá em **Users** → Selecione o usuário `dev@rodolfodebonis.com.br`
3. Aba **Attributes**
4. Adicione:
   - **Key**: `organization_id`
   - **Value**: `org-spooliq-001` (ou qualquer ID único)
5. Clique em **Save**

### 2. Criar Client Scope

1. Vá em **Client Scopes** → **Create**
2. Nome: `organization`
3. Type: `Default`
4. Protocol: `openid-connect`
5. Clique em **Save**

### 3. Adicionar Mapper ao Scope

1. No scope `organization` criado, vá na aba **Mappers**
2. Clique em **Add mapper** → **By configuration**
3. Selecione **User Attribute**
4. Configure:
   - **Name**: `organization-id-mapper`
   - **User Attribute**: `organization_id`
   - **Token Claim Name**: `organization_id`
   - **Claim JSON Type**: `String`
   - **Add to ID token**: ✅ ON
   - **Add to access token**: ✅ ON
   - **Add to userinfo**: ✅ ON
5. Clique em **Save**

### 4. Associar Scope ao Cliente

1. Vá em **Clients** → Selecione o cliente `spooliq`
2. Aba **Client Scopes**
3. Clique em **Add client scope**
4. Selecione `organization` 
5. Adicione como **Default** (não Optional)
6. Clique em **Add**

### 5. Testar

Após a configuração, faça logout e login novamente:

```bash
curl -X POST http://localhost:8000/v1/login \
  -H "Content-Type: application/json" \
  -d '{
    "email": "dev@rodolfodebonis.com.br",
    "password": "U{}z3N!B]xubk$ZK*DU/q7H4R8S8CG4%"
  }'
```

O token agora deve conter o claim `organization_id`.

---

## 🧪 Criar Segundo Usuário para Testar Isolamento

Para testar o isolamento de multi-tenancy:

1. Crie um novo usuário no Keycloak: `teste@empresa2.com.br`
2. Adicione o atributo `organization_id` = `org-empresa2-002`
3. Faça login com esse usuário
4. Crie dados (brands, materials, customers, etc.)
5. Verifique que os dois usuários NÃO veem os dados um do outro! ✅

---

## 📝 IMPORTANTE

- Cada organização (empresa) deve ter um `organization_id` único
- Todos os usuários da mesma organização devem ter o MESMO `organization_id`
- O sistema isolará automaticamente os dados por organização
- Admins veem apenas dados da sua própria organização (não há super-admin)

---

## 🎯 Status Atual

⚠️ **Ação Necessária**: Configure o Keycloak conforme os passos acima antes de continuar testando a API.

Após a configuração, o endpoint de company funcionará corretamente!

