# 📋 Backend Requirements - Sistema de Presets

## 🔍 Status Atual da API

### ✅ **Endpoints Funcionais**
| Endpoint | Método | Status | Resposta |
|----------|---------|--------|----------|
| `/presets/energy` | GET | ✅ Funcionando | `{ presets: [...] }` |
| `/presets/energy/locations` | GET | ✅ Funcionando | `{ locations: [...] }` |
| `/presets/machines` | GET | ✅ Funcionando | `{ machines: [...] }` |

### ❌ **Endpoints Ausentes**
| Endpoint | Método | Status | Descrição |
|----------|---------|--------|-----------|
| `/presets/cost` | GET | ❌ 404 | Lista presets de custo |
| `/presets/margin` | GET | ❌ 404 | Lista presets de margem |
| `/presets` | POST | ❌ Não Implementado | Cria presets (all types) |
| `/presets/{key}` | PUT | ❌ Não Implementado | Atualiza preset |
| `/presets/{key}` | DELETE | ❌ Não Implementado | Deleta preset |

---

## 🔴 **PRIORIDADE CRÍTICA - Implementar Imediatamente**

### 1. **Cost Presets Endpoints**

#### **GET `/presets/cost`**
```json
{
  "cost_presets": [
    {
      "key": "cost_001",
      "name": "Custo Padrão",
      "description": "Perfil padrão de custos operacionais",
      "overhead_amount": 15.50,
      "wear_percentage": 2.5,
      "is_default": true,
      "created_at": "2024-01-01T00:00:00Z",
      "updated_at": "2024-01-01T00:00:00Z"
    }
  ]
}
```

**Campos Obrigatórios:**
- `key` (string): ID único
- `name` (string): Nome do preset
- `overhead_amount` (number): Valor fixo de overhead
- `wear_percentage` (number): Percentual de desgaste (0-100)

**Campos Opcionais:**
- `description` (string): Descrição do preset
- `is_default` (boolean): Se é o preset padrão
- `created_at`, `updated_at` (timestamp): Metadados

### 2. **Margin Presets Endpoints**

#### **GET `/presets/margin`**
```json
{
  "margin_presets": [
    {
      "key": "margin_001",
      "name": "Margem Padrão",
      "description": "Perfil padrão de margens de lucro",
      "printing_only_margin": 25.0,
      "printing_plus_margin": 35.0,
      "full_service_margin": 50.0,
      "operator_rate_per_hour": 15.00,
      "modeler_rate_per_hour": 25.00,
      "is_default": true,
      "created_at": "2024-01-01T00:00:00Z",
      "updated_at": "2024-01-01T00:00:00Z"
    }
  ]
}
```

**Campos Obrigatórios:**
- `key` (string): ID único
- `name` (string): Nome do preset
- `printing_only_margin` (number): Margem só impressão (%)
- `printing_plus_margin` (number): Margem impressão+ (%)
- `full_service_margin` (number): Margem serviço completo (%)
- `operator_rate_per_hour` (number): Valor/hora operador
- `modeler_rate_per_hour` (number): Valor/hora modelador

### 3. **Campo `key` nos Endpoints Existentes**

#### **Energy Presets - Adicionar Campos:**
```json
{
  "presets": [
    {
      // ✅ Campos existentes
      "base_tariff": 0.804,
      "flag_surcharge": 0,
      "location": "Maceió-AL",
      "year": 2025,
      "description": "Tarifa energética...",

      // ❌ ADICIONAR:
      "key": "energy_maceio_2025",
      "state": "Alagoas",
      "city": "Maceió",
      "month": null,
      "flag_type": "green",
      "created_at": "2024-01-01T00:00:00Z",
      "updated_at": "2024-01-01T00:00:00Z"
    }
  ]
}
```

#### **Machine Presets - Adicionar Campos:**
```json
{
  "machines": [
    {
      // ✅ Campos existentes
      "name": "BambuLab A1 Combo",
      "brand": "BambuLab",
      "model": "A1 Combo",
      "watt": 95,
      "idle_factor": 0,
      "description": "Impressora 3D...",
      "url": "https://bambulab.com/en/a1",

      // ❌ ADICIONAR:
      "key": "bambulab_a1_combo",
      "build_volume": {
        "x": 256,
        "y": 256,
        "z": 256
      },
      "nozzle_diameter": 0.4,
      "max_temperature": 300,
      "heated_bed": true,
      "created_at": "2024-01-01T00:00:00Z",
      "updated_at": "2024-01-01T00:00:00Z"
    }
  ]
}
```

---

## 🟡 **PRIORIDADE ALTA - CRUD Operations**

### 4. **POST `/presets?type={type}`**

**Criar Energy Preset:**
```json
POST /presets?type=energy
{
  "location": "Brasília-DF",
  "state": "Distrito Federal",
  "city": "Brasília",
  "base_tariff": 0.75,
  "flag_surcharge": 0.05,
  "year": 2025,
  "month": 3,
  "flag_type": "yellow",
  "description": "Tarifa para Brasília"
}
```

**Criar Machine Preset:**
```json
POST /presets?type=machine
{
  "name": "Prusa MK4",
  "brand": "Prusa",
  "model": "i3 MK4",
  "watt": 120,
  "idle_factor": 0.05,
  "description": "Impressora Prusa",
  "url": "https://prusa3d.com",
  "build_volume": {
    "x": 250,
    "y": 210,
    "z": 220
  },
  "nozzle_diameter": 0.4,
  "max_temperature": 300,
  "heated_bed": true
}
```

**Criar Cost Preset:**
```json
POST /presets?type=cost
{
  "name": "Custo Premium",
  "description": "Custos para serviços premium",
  "overhead_amount": 25.00,
  "wear_percentage": 3.5,
  "is_default": false
}
```

**Criar Margin Preset:**
```json
POST /presets?type=margin
{
  "name": "Margem Competitiva",
  "description": "Margens para mercado competitivo",
  "printing_only_margin": 20.0,
  "printing_plus_margin": 30.0,
  "full_service_margin": 45.0,
  "operator_rate_per_hour": 12.00,
  "modeler_rate_per_hour": 20.00,
  "is_default": false
}
```

### 5. **PUT `/presets/{key}`**
```json
PUT /presets/energy_brasilia_2025
{
  "base_tariff": 0.78,
  "flag_surcharge": 0.06,
  "description": "Tarifa atualizada para Brasília"
}
```

**Resposta:**
```json
{
  "key": "energy_brasilia_2025",
  "location": "Brasília-DF",
  "base_tariff": 0.78,
  "flag_surcharge": 0.06,
  "description": "Tarifa atualizada para Brasília",
  "updated_at": "2024-01-15T10:30:00Z"
}
```

### 6. **DELETE `/presets/{key}`**
```json
DELETE /presets/energy_brasilia_2025
```

**Resposta:**
```json
{
  "message": "Preset deleted successfully",
  "deleted_key": "energy_brasilia_2025"
}
```

---

## 🔵 **PRIORIDADE MÉDIA - Funcionalidades Admin**

### 7. **Autenticação e Autorização**

#### **Middleware de Autenticação:**
- Verificar token JWT válido
- Extrair informações do usuário
- Validar role "admin" para operações CRUD

#### **Proteção de Endpoints:**
```
✅ GET endpoints: Público
❌ POST/PUT/DELETE: Apenas admins
```

#### **Headers Necessários:**
```
Authorization: Bearer {jwt_token}
Content-Type: application/json
```

### 8. **Validação de Dados**

#### **Energy Presets:**
- `base_tariff`: número > 0
- `flag_surcharge`: número >= 0
- `year`: inteiro 2020-2030
- `month`: inteiro 1-12 (opcional)
- `location`: string obrigatória

#### **Machine Presets:**
- `watt`: inteiro > 0
- `idle_factor`: decimal 0-1
- `name`: string única
- `build_volume`: objeto com x,y,z > 0

#### **Cost Presets:**
- `overhead_amount`: número >= 0
- `wear_percentage`: decimal 0-100
- `name`: string única

#### **Margin Presets:**
- Todas as margins: decimal >= 0
- Rates per hour: decimal >= 0
- `name`: string única

### 9. **Tratamento de Erros**

#### **Códigos de Status:**
```
200: Success
201: Created
400: Bad Request (dados inválidos)
401: Unauthorized (sem token)
403: Forbidden (não é admin)
404: Not Found (preset não existe)
409: Conflict (nome duplicado)
422: Unprocessable Entity (validação falhou)
500: Internal Server Error
```

#### **Formato de Erro:**
```json
{
  "error": "Validation failed",
  "message": "Invalid data provided",
  "details": {
    "base_tariff": ["must be greater than 0"],
    "location": ["is required"]
  }
}
```

---

## 🟢 **PRIORIDADE BAIXA - Funcionalidades Avançadas**

### 10. **Filtragem e Busca**

#### **Energy Presets com Filtros:**
```
GET /presets/energy?location=São Paulo&year=2025&flag_type=green
```

#### **Machine Presets com Busca:**
```
GET /presets/machines?search=bambu&min_watt=100&max_watt=200
```

### 11. **Paginação**
```
GET /presets/energy?page=1&per_page=10&sort=year&order=desc
```

**Resposta:**
```json
{
  "presets": [...],
  "pagination": {
    "current_page": 1,
    "per_page": 10,
    "total": 50,
    "total_pages": 5,
    "has_next": true,
    "has_prev": false
  }
}
```

### 12. **Funcionalidades Extras**
- **Duplicar preset:** `POST /presets/{key}/duplicate`
- **Export JSON:** `GET /presets/export`
- **Import JSON:** `POST /presets/import`
- **Histórico:** `GET /presets/{key}/history`

---

## 📊 **Cronograma Sugerido**

### **Sprint 1 (Crítico - 1 semana)**
1. ✅ Adicionar campo `key` em Energy/Machine presets
2. ✅ Implementar Cost Presets (GET)
3. ✅ Implementar Margin Presets (GET)

### **Sprint 2 (Alto - 1 semana)**
4. ✅ CRUD operations (POST/PUT/DELETE)
5. ✅ Validação básica de dados
6. ✅ Autenticação admin

### **Sprint 3 (Médio - 1 semana)**
7. ✅ Tratamento de erros robusto
8. ✅ Validação avançada
9. ✅ Timestamps e metadados

### **Sprint 4 (Baixo - Opcional)**
10. ✅ Filtragem e busca
11. ✅ Paginação
12. ✅ Funcionalidades extras

---

## 🔧 **Validação e Testes**

### **Checklist de Implementação:**
- [ ] Cost presets GET endpoint
- [ ] Margin presets GET endpoint
- [ ] Campo `key` em todos os presets
- [ ] CRUD operations funcionando
- [ ] Autenticação admin
- [ ] Validação de dados
- [ ] Tratamento de erros
- [ ] Tests unitários
- [ ] Tests de integração

### **Testes Essenciais:**
1. Listar todos os tipos de presets
2. Criar preset com dados válidos
3. Tentar criar sem autenticação (deve falhar)
4. Atualizar preset existente
5. Deletar preset
6. Validação de dados inválidos
7. Busca e filtragem

Com essa implementação, o sistema de presets ficará 100% funcional! 🚀