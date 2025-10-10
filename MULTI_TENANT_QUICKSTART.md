# Multi-Tenant & PDF Generator - Quick Start

## 🚀 O que foi implementado?

✅ **Multi-tenancy completo** com Keycloak  
✅ **Company Settings** (gerenciar informações da empresa)  
✅ **CDN Integration** (upload de arquivos)  
✅ **Upload Endpoints** (logos e arquivos)  
✅ **Budget Enhancements** (campos adicionais para PDF)

## ⚙️ Configuração Rápida

### 1. Keycloak Setup

Siga o guia completo: `docs/keycloak-multi-tenant-setup.md`

**Resumo**:
1. Criar User Attribute: `organization_id`
2. Criar Client Scope: `organization`
3. Adicionar Protocol Mapper para `organization_id`
4. Atribuir scope ao client `spooliq`
5. Definir `organization_id` para cada usuário

### 2. Environment Variables

Adicione ao seu `.env`:

```bash
# CDN Configuration
CDN_BASE_URL=https://rb-cdn.rodolfodebonis.com.br
CDN_API_KEY=<sua-api-key>
CDN_BUCKET=spooliq
```

### 3. Rodar o Projeto

```bash
make run
```

## 📝 API Endpoints

### Company Settings

| Método | Endpoint | Descrição |
|--------|----------|-----------|
| POST | `/v1/company` | Criar company |
| GET | `/v1/company` | Buscar company |
| PUT | `/v1/company` | Atualizar company |

### Uploads

| Método | Endpoint | Descrição | Size Limit |
|--------|----------|-----------|-----------|
| POST | `/v1/uploads/logo` | Upload logo | 5MB |
| POST | `/v1/uploads/file` | Upload arquivo | 50MB |

### Budgets (Updated)

Agora inclui:
- `delivery_days`: Prazo de entrega
- `payment_terms`: Condições de pagamento
- `notes`: Observações
- `pdf_url`: URL do PDF (preparado para futuro)

## 🧪 Testar

### 1. Criar Company

```bash
TOKEN="seu-token-aqui"

curl -X POST http://localhost:8000/v1/company \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Artesier 3D",
    "email": "contato@artesier.com",
    "phone": "(11) 99999-9999",
    "whatsapp": "(11) 99999-9999",
    "instagram": "@artesier3d"
  }'
```

### 2. Upload Logo

```bash
curl -X POST http://localhost:8000/v1/uploads/logo \
  -H "Authorization: Bearer $TOKEN" \
  -F "file=@logo.png"
```

Guarde a URL retornada!

### 3. Atualizar Company com Logo

```bash
curl -X PUT http://localhost:8000/v1/company \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "logo_url": "https://rb-cdn.rodolfodebonis.com.br/v1/cdn/spooliq/logos/xxx.png"
  }'
```

### 4. Criar Budget com Novos Campos

```bash
curl -X POST http://localhost:8000/v1/budgets \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Orçamento Outubro Rosa",
    "description": "Chaveiros personalizados",
    "customer_id": "<customer-uuid>",
    "print_time_hours": 3,
    "print_time_minutes": 45,
    "delivery_days": 7,
    "payment_terms": "50% entrada + 50% na entrega",
    "notes": "Cliente VIP - prioridade",
    "include_energy_cost": true,
    "include_labor_cost": true,
    "include_waste_cost": true,
    "items": [
      {
        "filament_id": "<filament-uuid>",
        "quantity": 350.0,
        "order": 0
      }
    ]
  }'
```

## 📚 Documentação Completa

- **Keycloak Setup**: `docs/keycloak-multi-tenant-setup.md`
- **Upload Guide**: `docs/uploads-guide.md`
- **Implementation Summary**: `IMPLEMENTATION_SUMMARY.md`

## 🔜 Próximos Passos (Fase 2)

1. Implementar PDF Service
2. Criar template de PDF baseado no exemplo
3. Endpoint para gerar PDF: `GET /v1/budgets/:id/pdf`
4. Auto-geração de PDF ao criar/atualizar budget

## 🎯 Estrutura do Projeto

```
features/
├── company/           # ✅ Company Settings Feature
│   ├── data/
│   │   ├── models/
│   │   └── repositories/
│   ├── domain/
│   │   ├── entities/
│   │   ├── repositories/
│   │   └── usecases/
│   ├── di/
│   └── routes.go
├── uploads/           # ✅ Upload Feature
│   ├── domain/
│   │   └── usecases/
│   ├── di/
│   └── routes.go
└── budget/            # ✅ Enhanced with new fields
    └── ...

core/
├── services/
│   ├── cdn_service.go # ✅ CDN Integration
│   └── ...
├── entities/
│   └── jwt_claims_entity.go # ✅ OrganizationID added
├── middlewares/
│   └── auth_middleware.go # ✅ Organization context
└── config/
    └── environment.go # ✅ CDN env vars
```

## ✅ Checklist

- [x] Multi-tenancy infrastructure
- [x] Company Settings CRUD
- [x] CDN Service
- [x] Upload endpoints
- [x] Budget enhancements
- [x] Migrations
- [x] Documentation
- [x] Compilation OK
- [ ] PDF Service (Fase 2)
- [ ] PDF Generation (Fase 2)

## 🆘 Troubleshooting

### Token não tem organization_id
- Verifique se o user tem o atributo configurado no Keycloak
- Verifique se o client scope está assignado como "Default"
- Limpe o cache e obtenha um novo token

### Upload falha
- Verifique se o CDN_API_KEY está correto
- Verifique se o CDN está acessível
- Verifique o tamanho do arquivo

### Company já existe
- Cada organization só pode ter 1 company
- Use PUT para atualizar ao invés de criar novamente

## 📞 Suporte

Para mais informações, consulte a documentação completa em `IMPLEMENTATION_SUMMARY.md`.

