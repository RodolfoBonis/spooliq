# 🔍 Validação de Endpoints - Frontend vs Backend

## ❌ Discrepâncias Encontradas

### 1. **POST /register** - CRÍTICO

**Documentado:**
```typescript
Body: {
  name: string;
  email: string;
  password: string;
  company: {
    name: string;
    trade_name?: string;
    document?: string;
    phone?: string;
  }
}
```

**Backend Real (RegisterRequest):**
```go
{
  // User data
  name: string;
  email: string;
  password: string;
  
  // Company data (FLAT, não nested)
  company_name: string;           // required
  company_trade_name: string;     // optional
  company_document: string;        // required (CNPJ)
  company_phone: string;           // required
  
  // Address (OBRIGATÓRIO)
  address: string;                 // required
  address_number: string;          // required
  complement: string;              // optional
  neighborhood: string;            // required
  city: string;                    // required
  state: string;                   // required (2 chars)
  zip_code: string;                // required
}
```

**❌ Problemas:**
- Estrutura completamente diferente (flat vs nested)
- Campos de endereço obrigatórios não documentados
- `company_document` e `company_phone` são **obrigatórios** no backend

---

### 2. **Company Fields** - Nome de campos inconsistente

**Documentado:**
```typescript
{
  whats_app?: string;  // ❌ INCORRETO
}
```

**Backend Real:**
```go
{
  whatsapp: string;  // ✅ SEM UNDERSCORE
}
```

**❌ Problema:** 
- Frontend docs usam `whats_app` (com underscore)
- Backend usa `whatsapp` (sem underscore)
- Isso causará erro de deserialização

---

### 3. **POST /company/logo** - Nome do campo FormData

**Documentado:**
```typescript
Body: FormData {
  file: File  // ✅ CORRETO (já corrigido)
}
```

**Backend Real:**
```go
fileHeader, err := c.FormFile("file")  // ✅ Correto agora
```

**✅ Status:** CORRETO (foi corrigido durante a sessão)

---

### 4. **GET /company/** - Response fields

**Documentado tem campos extras não retornados:**
```typescript
Response: {
  // ... outros campos ...
  subscription_status: 'trial' | 'active' | 'overdue' | 'cancelled';  // ✅ OK
  trial_ends_at?: string;                                               // ✅ OK
  subscription_plan: 'basic' | 'pro' | 'enterprise';                   // ✅ OK
  
  // ❌ MAS FALTAM campos que o backend RETORNA:
  is_platform_company: boolean;
  subscription_started_at?: string;
  asaas_customer_id?: string;
  asaas_subscription_id?: string;
  last_payment_check?: string;
  next_payment_due?: string;
}
```

**⚠️ Problema:** Docs não mostram todos os campos que o backend pode retornar

---

### 5. **Budget Status Values** - Possível inconsistência

**Documentado:**
```typescript
status: 'draft' | 'sent' | 'approved' | 'rejected' | 'printing' | 'completed';
```

**Backend verificar:**
- Confirmar se todos esses valores são exatamente esses no backend
- Verificar se não há outros status possíveis

---

### 6. **Filament Color Data** - Estrutura complexa

**Documentado:**
```typescript
color_data: {
  // Solid
  color: string;
  
  // Gradient
  from: string;
  to: string;
  direction?: 'horizontal' | 'vertical' | 'diagonal';
  
  // Duo
  primary: string;
  secondary: string;
  ratio?: number;
  
  // Rainbow
  colors: string[];
  pattern?: string;
}
```

**⚠️ Precisa validar:** Se esses campos exatos são os esperados pelo backend

---

## ✅ Endpoints Validados Corretos

### 1. **Branding Endpoints** ✅
- `GET /company/branding` - OK
- `PUT /company/branding` - OK
- `GET /company/branding/templates` - OK
- Estrutura de cores completa e correta

### 2. **POST /company/logo** ✅
- Campo `file` correto no FormData
- Validação de tipo e tamanho documentada

### 3. **Budget PDF Generation** ✅
- `GET /budgets/:id/pdf` - OK
- Retorna binary PDF

---

## 🔧 Correções Necessárias

### Prioridade ALTA

1. **Corrigir RegisterRequest na documentação do frontend**
2. **Corrigir `whats_app` → `whatsapp` em todos os lugares**
3. **Adicionar campos de subscription faltantes em Company response**

### Prioridade MÉDIA

4. Validar todos os status de Budget com o backend
5. Validar estrutura de color_data dos filamentos
6. Verificar se há outros campos opcionais não documentados

---

## 📝 Checklist de Validação

- [ ] Corrigir RegisterRequest structure
- [ ] Corrigir whatsapp field naming
- [ ] Adicionar subscription fields em Company
- [ ] Validar Budget status enum
- [ ] Validar Filament color_data structure
- [ ] Verificar Customer fields
- [ ] Verificar Preset fields
- [ ] Verificar User management fields

---

**Gerado em:** $(date)
**Prioridade:** Corrigir antes de iniciar desenvolvimento frontend

