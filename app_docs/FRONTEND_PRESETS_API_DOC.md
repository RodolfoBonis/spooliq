# 📋 API de Presets - Documentação para Frontend

## 🎯 **VISÃO GERAL**

Esta documentação detalha todos os endpoints da API de Presets implementados no backend para serem consumidos pelo frontend. O sistema suporta 4 tipos de presets: **Energy**, **Machine**, **Cost** e **Margin**.

**Base URL:** `/api/v1/presets`

---

## 🔐 **AUTENTICAÇÃO**

### **Endpoints Públicos (Sem autenticação)**
- Todos os endpoints `GET` são públicos

### **Endpoints Protegidos (Requer autenticação Admin)**
- Todos os endpoints `POST`, `PUT` e `DELETE` requerem:
  - Header: `Authorization: Bearer {jwt_token}`
  - Role: `admin`

---

## 📊 **ENDPOINTS DISPONÍVEIS**

### **1. GET /presets/energy/locations**
**Descrição:** Lista todas as localizações disponíveis para presets de energia.

**Método:** `GET`
**Autenticação:** Não requerida
**Parâmetros:** Nenhum

**Response (200):**
```json
{
  "locations": [
    "Maceió-AL",
    "São Paulo-SP",
    "Rio de Janeiro-RJ",
    "Brasília-DF"
  ]
}
```

**Códigos de Erro:**
- `500` - Erro interno do servidor

---

### **2. GET /presets/energy**
**Descrição:** Lista presets de energia, com filtro opcional por localização.

**Método:** `GET`
**Autenticação:** Não requerida
**Query Parameters:**
- `location` (string, opcional) - Filtrar por localização específica

**Exemplos de chamada:**
```
GET /presets/energy
GET /presets/energy?location=Maceió-AL
```

**Response (200):**
```json
{
  "presets": [
    {
      "key": "energy_maceio_al_2025",
      "base_tariff": 0.804,
      "flag_surcharge": 0.0,
      "location": "Maceió-AL",
      "state": "Alagoas",
      "city": "Maceió",
      "year": 2025,
      "month": null,
      "flag_type": "green",
      "description": "Tarifa energética para Maceió",
      "created_at": "2024-01-01T00:00:00Z",
      "updated_at": "2024-01-01T00:00:00Z"
    }
  ]
}
```

**Campos do Energy Preset:**
- `key` (string) - Identificador único do preset
- `base_tariff` (number) - Tarifa base em R$/kWh
- `flag_surcharge` (number) - Sobretaxa da bandeira tarifária
- `location` (string) - Localização completa (ex: "Maceió-AL")
- `state` (string) - Estado completo (ex: "Alagoas")
- `city` (string) - Cidade (ex: "Maceió")
- `year` (number) - Ano de referência
- `month` (number|null) - Mês específico (1-12) ou null para ano todo
- `flag_type` (string) - Tipo da bandeira: "green", "yellow", "red"
- `description` (string) - Descrição do preset
- `created_at` (string) - Data de criação (ISO 8601)
- `updated_at` (string) - Data da última atualização (ISO 8601)

---

### **3. GET /presets/machines**
**Descrição:** Lista todos os presets de máquinas/impressoras 3D.

**Método:** `GET`
**Autenticação:** Não requerida
**Parâmetros:** Nenhum

**Response (200):**
```json
{
  "machines": [
    {
      "key": "machine_bambulab_a1_combo_1234567890",
      "name": "BambuLab A1 Combo",
      "brand": "BambuLab",
      "model": "A1 Combo",
      "watt": 95,
      "idle_factor": 0.0,
      "description": "Impressora 3D BambuLab A1 Combo",
      "url": "https://bambulab.com/en/a1",
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

**Campos do Machine Preset:**
- `key` (string) - Identificador único do preset
- `name` (string) - Nome da máquina
- `brand` (string) - Marca da máquina
- `model` (string) - Modelo da máquina
- `watt` (number) - Potência em watts
- `idle_factor` (number) - Fator de consumo em idle (0.0-1.0)
- `description` (string) - Descrição da máquina
- `url` (string) - URL do produto (opcional)
- `build_volume` (object) - Volume de impressão
  - `x` (number) - Largura em mm
  - `y` (number) - Profundidade em mm
  - `z` (number) - Altura em mm
- `nozzle_diameter` (number) - Diâmetro do bico em mm
- `max_temperature` (number) - Temperatura máxima em °C
- `heated_bed` (boolean) - Possui mesa aquecida
- `created_at` (string) - Data de criação (ISO 8601)
- `updated_at` (string) - Data da última atualização (ISO 8601)

---

### **4. GET /presets/cost** ⭐ **NOVO**
**Descrição:** Lista todos os presets de custos operacionais.

**Método:** `GET`
**Autenticação:** Não requerida
**Parâmetros:** Nenhum

**Response (200):**
```json
{
  "cost_presets": [
    {
      "key": "cost_padrao_1234567890",
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

**Campos do Cost Preset:**
- `key` (string) - Identificador único do preset
- `name` (string) - Nome do preset de custo
- `description` (string) - Descrição do preset
- `overhead_amount` (number) - Valor fixo de overhead em R$
- `wear_percentage` (number) - Porcentagem de desgaste (0-100)
- `is_default` (boolean) - Se é o preset padrão
- `created_at` (string) - Data de criação (ISO 8601)
- `updated_at` (string) - Data da última atualização (ISO 8601)

---

### **5. GET /presets/margin** ⭐ **NOVO**
**Descrição:** Lista todos os presets de margens de lucro.

**Método:** `GET`
**Autenticação:** Não requerida
**Parâmetros:** Nenhum

**Response (200):**
```json
{
  "margin_presets": [
    {
      "key": "margin_padrao_1234567890",
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

**Campos do Margin Preset:**
- `key` (string) - Identificador único do preset
- `name` (string) - Nome do preset de margem
- `description` (string) - Descrição do preset
- `printing_only_margin` (number) - Margem para "Só Impressão" em %
- `printing_plus_margin` (number) - Margem para "Impressão+" em %
- `full_service_margin` (number) - Margem para "Serviço Completo" em %
- `operator_rate_per_hour` (number) - Taxa do operador por hora em R$
- `modeler_rate_per_hour` (number) - Taxa do modelador por hora em R$
- `is_default` (boolean) - Se é o preset padrão
- `created_at` (string) - Data de criação (ISO 8601)
- `updated_at` (string) - Data da última atualização (ISO 8601)

---

## 🔧 **ENDPOINTS DE MODIFICAÇÃO (Admin Only)**

### **6. POST /presets?type={type}**
**Descrição:** Cria um novo preset. Suporta todos os 4 tipos.

**Método:** `POST`
**Autenticação:** **REQUERIDA** (Admin)
**Query Parameters:**
- `type` (string, obrigatório) - Tipo do preset: `energy`, `machine`, `cost`, `margin`

#### **6.1. Criar Energy Preset**
**Endpoint:** `POST /presets?type=energy`

**Request Body:**
```json
{
  "location": "Brasília-DF",
  "state": "Distrito Federal",
  "city": "Brasília",
  "base_tariff": 0.75,
  "flag_surcharge": 0.05,
  "year": 2025,
  "month": null,
  "flag_type": "yellow",
  "description": "Tarifa energética para Brasília"
}
```

**Campos obrigatórios:**
- `location`, `state`, `city`, `base_tariff`, `year`, `flag_type`

**Valores válidos para `flag_type`:** `"green"`, `"yellow"`, `"red"`

#### **6.2. Criar Machine Preset**
**Endpoint:** `POST /presets?type=machine`

**Request Body:**
```json
{
  "name": "Ender 3 V2",
  "brand": "Creality",
  "model": "Ender 3 V2",
  "watt": 270,
  "idle_factor": 0.1,
  "description": "Impressora 3D Creality Ender 3 V2",
  "url": "https://www.creality.com/products/ender-3-v2-3d-printer",
  "build_volume": {
    "x": 220,
    "y": 220,
    "z": 250
  },
  "nozzle_diameter": 0.4,
  "max_temperature": 260,
  "heated_bed": true
}
```

**Campos obrigatórios:**
- `name`, `brand`, `model`, `watt`

#### **6.3. Criar Cost Preset** ⭐ **NOVO**
**Endpoint:** `POST /presets?type=cost`

**Request Body:**
```json
{
  "name": "Custo Premium",
  "description": "Perfil de custos para serviços premium",
  "overhead_amount": 25.00,
  "wear_percentage": 3.5,
  "is_default": false
}
```

**Campos obrigatórios:**
- `name`, `overhead_amount`, `wear_percentage`

#### **6.4. Criar Margin Preset** ⭐ **NOVO**
**Endpoint:** `POST /presets?type=margin`

**Request Body:**
```json
{
  "name": "Margem Premium",
  "description": "Perfil de margens para clientes premium",
  "printing_only_margin": 30.0,
  "printing_plus_margin": 40.0,
  "full_service_margin": 60.0,
  "operator_rate_per_hour": 20.00,
  "modeler_rate_per_hour": 35.00,
  "is_default": false
}
```

**Campos obrigatórios:**
- `name`, `printing_only_margin`, `printing_plus_margin`, `full_service_margin`

**Response (201):** Sem conteúdo (preset criado com sucesso)

**Códigos de Erro:**
- `400` - Dados inválidos ou tipo não suportado
- `401` - Token não fornecido ou inválido
- `403` - Usuário não tem permissão de admin
- `409` - Preset com chave duplicada já existe
- `500` - Erro interno do servidor

---

### **7. PUT /presets/{key}**
**Descrição:** Atualiza um preset existente por sua chave.

**Método:** `PUT`
**Autenticação:** **REQUERIDA** (Admin)
**Path Parameters:**
- `key` (string) - Chave única do preset

**Request Body:**
```json
{
  "data": {
    "base_tariff": 0.78,
    "flag_surcharge": 0.06
  }
}
```

**Exemplo de atualização de Cost Preset:**
```json
{
  "data": {
    "overhead_amount": 20.00,
    "wear_percentage": 3.0
  }
}
```

**Response (200):** Sem conteúdo (preset atualizado com sucesso)

**Códigos de Erro:**
- `400` - Dados inválidos
- `401` - Token não fornecido ou inválido
- `403` - Usuário não tem permissão de admin
- `404` - Preset não encontrado
- `500` - Erro interno do servidor

---

### **8. DELETE /presets/{key}**
**Descrição:** Deleta um preset por sua chave.

**Método:** `DELETE`
**Autenticação:** **REQUERIDA** (Admin)
**Path Parameters:**
- `key` (string) - Chave única do preset

**Response (204):** Sem conteúdo (preset deletado com sucesso)

**Códigos de Erro:**
- `401` - Token não fornecido ou inválido
- `403` - Usuário não tem permissão de admin
- `404` - Preset não encontrado
- `500` - Erro interno do servidor

---

## 🚨 **TRATAMENTO DE ERROS**

Todos os endpoints retornam erros no formato padrão:

```json
{
  "error": "Mensagem de erro detalhada",
  "code": "CODIGO_ERRO"
}
```

### **Códigos de Status HTTP:**
- `200` - Sucesso
- `201` - Criado com sucesso
- `204` - Deletado com sucesso
- `400` - Requisição inválida
- `401` - Não autenticado
- `403` - Não autorizado (sem permissão admin)
- `404` - Recurso não encontrado
- `409` - Conflito (preset duplicado)
- `500` - Erro interno do servidor

---

## 💡 **EXEMPLOS DE USO NO FRONTEND**

### **Listar Cost Presets**
```javascript
// GET request para listar cost presets
const response = await fetch('/api/v1/presets/cost');
const data = await response.json();
console.log(data.cost_presets);
```

### **Criar Margin Preset (Admin)**
```javascript
// POST request para criar margin preset
const response = await fetch('/api/v1/presets?type=margin', {
  method: 'POST',
  headers: {
    'Content-Type': 'application/json',
    'Authorization': `Bearer ${adminToken}`
  },
  body: JSON.stringify({
    name: "Margem VIP",
    description: "Margem para clientes VIP",
    printing_only_margin: 35.0,
    printing_plus_margin: 45.0,
    full_service_margin: 65.0,
    operator_rate_per_hour: 25.00,
    modeler_rate_per_hour: 40.00,
    is_default: false
  })
});
```

### **Atualizar Energy Preset (Admin)**
```javascript
// PUT request para atualizar preset
const response = await fetch('/api/v1/presets/energy_brasilia_df_2025', {
  method: 'PUT',
  headers: {
    'Content-Type': 'application/json',
    'Authorization': `Bearer ${adminToken}`
  },
  body: JSON.stringify({
    data: {
      base_tariff: 0.82,
      flag_surcharge: 0.07
    }
  })
});
```

---

## 📝 **NOTAS IMPORTANTES**

### **Chaves dos Presets**
- São geradas automaticamente pelo backend
- Formato padrão: `{type}_{identifier}_{timestamp}`
- Exemplos:
  - `energy_maceio_al_2025`
  - `machine_ender3_v2_creality_1234567890`
  - `cost_padrao_1234567890`
  - `margin_premium_1234567890`

### **Timestamps**
- Todos os timestamps estão no formato ISO 8601
- `created_at` é definido automaticamente na criação
- `updated_at` é atualizado automaticamente nas modificações

### **Validações**
- Todos os campos obrigatórios são validados no backend
- Valores numéricos têm validação de range apropriado
- Strings têm validação de tamanho mínimo/máximo

### **Performance**
- Endpoints GET são otimizados e podem ser chamados frequentemente
- Use cache no frontend quando apropriado
- Endpoints de modificação são mais pesados - use com moderação

---

## 🔄 **INTEGRAÇÃO COM SISTEMA EXISTENTE**

### **Compatibilidade**
- Todos os endpoints existentes continuam funcionando
- Novos campos foram adicionados aos presets existentes
- Backward compatibility mantida

### **Migração**
- Presets existentes receberam automaticamente as novas chaves
- Timestamps foram populados retroativamente
- Nenhuma ação manual necessária

---

**Desenvolvido:** Setembro 2025
**Versão da API:** v1
**Tecnologias:** Go + Gin + GORM + PostgreSQL
**Documentação Swagger:** Disponível em `/swagger/index.html`