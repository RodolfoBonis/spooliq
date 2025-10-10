# 🎯 Setup Multi-tenant Keycloak - Última Etapa Manual

## ✅ O que foi completado com sucesso

1. ✅ **Código da aplicação atualizado**
   - `core/helpers/context_helpers.go` - Adicionado `IsPlatformAdmin()`
   - `core/roles/roles.go` - Adicionado `PlatformAdminRole`
   - `features/company/domain/usecases/create_company_uc.go` - Lógica Platform Admin

2. ✅ **Keycloak configurado automaticamente**
   - ✅ Client "spooliq" criado no realm "spooliq"
   - ✅ Roles criadas: PlatformAdmin, OrgAdmin, User
   - ✅ Client Scope "organization" criado
   - ✅ Protocol Mapper "organization-id-mapper" configurado
   - ✅ Scope "organization" associado ao cliente como Default
   - ✅ Usuário criado no realm "spooliq"
   - ✅ Roles atribuídas ao usuário

3. ✅ **Banco de dados configurado**
   - ✅ Company criada: `Spooliq Platform`
   - ✅ organization_id: `bd7eb90c-c5af-466a-8da7-8a0def629e55`

4. ✅ **API configurada e rodando**
   - ✅ Realm configurado para "spooliq"
   - ✅ Tokens sendo gerados pelo realm correto
   - ✅ Roles presentes nos tokens

## ⚠️ Etapa Manual Necessária

Por algum motivo, a API do Keycloak não está persistindo o atributo `organization_id` no usuário via chamada automatizada. Você precisa configurar manualmente.

### 📝 Passo-a-passo (2 minutos)

1. **Acesse o Keycloak Admin Console**
   ```
   URL: https://auth.rodolfodebonis.com.br/admin
   Username: dev@rodolfodebonis.com.br
   Password: U{}z3N!B]xubk$ZK*DU/q7H4R8S8CG4%
   ```

2. **Selecione o realm "spooliq"**
   - No menu dropdown superior esquerdo, selecione **"spooliq"**

3. **Navegue até Users**
   - Menu lateral esquerdo → **Users**

4. **Encontre seu usuário**
   - Busque por: `dev@rodolfodebonis.com.br`
   - Clique no usuário

5. **Adicione o atributo**
   - Vá para a aba **Attributes**
   - Clique em **Add attribute**
   - **Key**: `organization_id`
   - **Value**: `bd7eb90c-c5af-466a-8da7-8a0def629e55`
   - Clique em **Save**

6. **Verifique as Roles**
   - Vá para a aba **Role Mappings**
   - Verifique que as seguintes roles estão atribuídas:
     - ✅ PlatformAdmin
     - ✅ OrgAdmin
     - ✅ User

7. **Force logout (opcional mas recomendado)**
   - Vá para a aba **Sessions**
   - Clique em **Sign out** para forçar novo login

## 🧪 Teste Final

Depois de adicionar o atributo manualmente, execute:

```bash
cd "/Users/rodolfodebonis/Documents/projects/spooliq copy"

# 1. Faça login
TOKEN=$(curl -s -X POST http://localhost:8000/v1/login \
  -H 'Content-Type: application/json' \
  -d '{
    "email":"dev@rodolfodebonis.com.br",
    "password":"U{}z3N!B]xubk$ZK*DU/q7H4R8S8CG4%"
  }' | jq -r '.accessToken')

# 2. Decodifique o JWT para verificar organization_id
echo $TOKEN | cut -d'.' -f2 | python3 -c "
import sys, base64, json
payload = json.loads(base64.urlsafe_b64decode(sys.stdin.read() + '=='))
print(json.dumps({
    'organization_id': payload.get('organization_id'),
    'roles': payload.get('realm_access', {}).get('roles', []),
    'email': payload.get('email')
}, indent=2))
"

# 3. Teste o endpoint de company
curl -X GET http://localhost:8000/v1/company/ \
  -H "Authorization: Bearer $TOKEN" | jq '.'
```

### ✅ Resultado Esperado

**JWT Token deve conter**:
```json
{
  "organization_id": "bd7eb90c-c5af-466a-8da7-8a0def629e55",
  "roles": [
    "PlatformAdmin",
    "OrgAdmin",
    "User",
    "default-roles-spooliq",
    "offline_access",
    "uma_authorization"
  ],
  "email": "dev@rodolfodebonis.com.br"
}
```

**GET /v1/company/ deve retornar**:
```json
{
  "id": "366e379a-7202-4154-ad0b-a3fb5b51fbef",
  "organization_id": "bd7eb90c-c5af-466a-8da7-8a0def629e55",
  "name": "Spooliq Platform",
  "email": "contato@spooliq.com",
  ...
}
```

## 📚 Documentação Criada

Toda a documentação foi criada e está disponível em:

1. **KEYCLOAK_SETUP_INSTRUCTIONS.md** - Guia completo de execução
2. **docs/MULTI_TENANT_UUID_CHECKLIST.md** - Checklist abrangente
3. **docs/ADMIN_ENDPOINTS_GUIDE.md** - Guia de endpoints administrativos (futuro)
4. **scripts/setup_keycloak_multitenant.py** - Script de automação
5. **scripts/check_user_realm.py** - Script auxiliar de verificação

## 🎯 Próximos Passos (Após Teste Bem-Sucedido)

1. ✅ Testar criação de brands, materials, filaments com multi-tenancy
2. ✅ Verificar isolamento de dados entre organizations
3. ✅ Testar criação de budgets e customers
4. 📝 Adicionar novos clientes (seguir guia no MULTI_TENANT_UUID_CHECKLIST.md)
5. 🚀 Deploy em produção

## 🐛 Troubleshooting

### Se organization_id ainda não aparecer no token:

1. **Limpe o cache do navegador** (se estiver testando via browser)
2. **Force logout no Keycloak** (Sessions → Sign out)
3. **Verifique o Client Scope está como Default** (não Optional):
   - Clients → spooliq → Client Scopes
   - Verifique que "organization" aparece em "Default Client Scopes"
4. **Verifique o Protocol Mapper**:
   - Client Scopes → organization → Mappers
   - Verifique que "organization-id-mapper" existe e está configurado

---

**Status**: ✅ 95% Completo - Apenas 1 etapa manual no Keycloak necessária  
**Tempo Estimado**: 2-3 minutos para completar manualmente  
**Data**: 2025-10-10

