# Subscription Management System

Sistema de gerenciamento de assinaturas para o SpoolIQ, com histórico de pagamentos e visualização de planos.

## 📋 Funcionalidades Implementadas

### 1. Histórico de Pagamentos

Lista todos os pagamentos da organização registrados pelos webhooks do Asaas.

**Endpoint:** `GET /api/v1/subscriptions/payment-history?limit=10&offset=0`

**Headers:**
```
Authorization: Bearer {token}
```

**Response:**
```json
{
  "payments": [
    {
      "id": "uuid",
      "organization_id": "org-123",
      "asaas_payment_id": "pay_123",
      "amount": 29.90,
      "status": "received",
      "payment_date": "2025-02-01T10:00:00Z",
      "due_date": "2025-02-01T00:00:00Z",
      "invoice_url": "https://..."
    }
  ],
  "total": 100,
  "limit": 10,
  "offset": 0
}
```

### 2. Visualização de Recursos por Plano

Endpoint público para consultar todos os planos e seus recursos.

**Endpoint:** `GET /api/v1/subscriptions/plans`

**Response:**
```json
{
  "plans": [
    {
      "id": "starter",
      "name": "Starter",
      "price": 29.90,
      "description": "Ideal para pequenos negócios e freelancers",
      "popular": true,
      "recommended": false,
      "features": [
        {
          "name": "Usuários",
          "description": "Número máximo de usuários na organização",
          "value": 5,
          "available": true
        }
      ]
    }
  ]
}
```

## 🔗 Integração com Webhooks do Asaas

O sistema utiliza o módulo **`features/webhooks`** para processar eventos do Asaas.

### Webhook Endpoint

**URL:** `POST /api/v1/webhooks/asaas`

Este endpoint é gerenciado pelo módulo `webhooks` e:
1. Valida a assinatura do webhook
2. Processa eventos de pagamento
3. Atualiza o status da company
4. **Registra automaticamente os pagamentos no histórico de subscriptions**

### Eventos Processados

- `PAYMENT_RECEIVED` - Pagamento recebido → Status: `received`
- `PAYMENT_CONFIRMED` - Pagamento confirmado → Status: `confirmed`
- `PAYMENT_OVERDUE` - Pagamento em atraso → Status: `overdue`
- `PAYMENT_REFUNDED` - Pagamento estornado → Status: `failed`

### Fluxo de Dados

```
Asaas → Webhook (/api/v1/webhooks/asaas)
  ↓
AsaasWebhookUseCase
  ├─→ Valida assinatura
  ├─→ Registra pagamento (subscription_payments)
  └─→ Atualiza status da company
```

## 📊 Banco de Dados

### Tabela: subscription_payments

```sql
CREATE TABLE subscription_payments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id VARCHAR(255) NOT NULL,
    asaas_payment_id VARCHAR(255),
    asaas_invoice_id VARCHAR(255),
    amount DECIMAL(10,2),
    status VARCHAR(20), -- pending, confirmed, received, overdue, failed
    payment_date TIMESTAMP,
    due_date TIMESTAMP,
    invoice_url TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_subscription_payments_org ON subscription_payments(organization_id);
CREATE INDEX idx_subscription_payments_asaas ON subscription_payments(asaas_payment_id);
CREATE INDEX idx_subscription_payments_status ON subscription_payments(status);
```

**Obs:** A tabela é criada automaticamente via GORM AutoMigration ao iniciar a aplicação.

## 🔧 Configuração

### 1. Integração com Asaas (via módulo webhooks)

Configure as credenciais do Asaas no `.env`:
```bash
ASAAS_API_KEY=your_api_key
ASAAS_WEBHOOK_SECRET=your_webhook_secret
ASAAS_ENVIRONMENT=sandbox # ou production
```

### 2. Configure o webhook no painel do Asaas

```
URL: https://api.spooliq.com/api/v1/webhooks/asaas
Eventos: PAYMENT_RECEIVED, PAYMENT_CONFIRMED, PAYMENT_OVERDUE, PAYMENT_REFUNDED
```

### 3. Ao criar pagamentos no Asaas

**IMPORTANTE:** Use o `organization_id` no campo `externalReference`:

```json
{
  "customer": "cus_xxx",
  "billingType": "CREDIT_CARD",
  "value": 29.90,
  "dueDate": "2025-02-01",
  "externalReference": "org-123"  ← Organization ID aqui!
}
```

Isso permite que o webhook associe o pagamento à organização correta.

## 🏗️ Arquitetura

### Módulos

1. **`features/webhooks`** - Gerencia webhooks do Asaas
   - Valida assinaturas
   - Processa eventos
   - Atualiza company
   - **Registra pagamentos em subscriptions**

2. **`features/subscriptions`** - Gerencia histórico e planos
   - Histórico de pagamentos
   - Visualização de planos
   - Repositório de pagamentos

### Dependências entre Módulos

```
webhooks
  └─→ usa → subscriptions/repositories (para registrar pagamentos)

subscriptions
  └─→ fornece → repository interface
```

## 📝 Status de Pagamento

| Status do Asaas | Status Interno | Descrição |
|----------------|----------------|-----------|
| PENDING | pending | Aguardando pagamento |
| CONFIRMED | confirmed | Pagamento confirmado |
| RECEIVED | received | Pagamento recebido |
| OVERDUE | overdue | Pagamento vencido |
| REFUNDED, etc | failed | Pagamento falhou/estornado |

## 🔐 Segurança

- Webhook valida assinatura HMAC-SHA256
- Endpoints de histórico requerem autenticação
- Endpoint de planos é público (apenas leitura)
- Todas as transações são logadas para auditoria

## 🚀 Funcionalidades Futuras

As seguintes funcionalidades foram planejadas mas não implementadas devido a complexidade de integração com Keycloak:

### Planejado para Futuro

1. **Atualização de Método de Pagamento**
   - Gerar link de pagamento via Asaas
   - Atualizar cartão de crédito

2. **Cancelamento de Assinatura**
   - Cancelar com motivo
   - Manter acesso até fim do período

3. **Upgrade/Downgrade de Planos**
   - Mudança entre planos
   - Cálculo de proração
   - Atualização de atributos no Keycloak

4. **Notificações por Email**
   - Trial expirando (7, 3, 1 dias)
   - Pagamento vencido (1, 3, 7, 15, 30 dias)
   - Templates HTML prontos

### Por que não foram implementadas agora?

Essas funcionalidades requerem:
- Integração complexa com Keycloak Admin API
- Serviço de email configurado
- Cron jobs para notificações automáticas
- Lógica de negócio mais avançada

Elas podem ser adicionadas quando houver necessidade, seguindo os exemplos e templates já criados.

## 📚 Referências

- [Documentação API Asaas](https://docs.asaas.com/)
- [Webhook do Asaas (features/webhooks)](../webhooks/README.md)
- [Gin Framework](https://gin-gonic.com/docs/)

## 🧪 Testando

### 1. Verificar planos disponíveis (público)

```bash
curl http://localhost:8000/api/v1/subscriptions/plans
```

### 2. Ver histórico de pagamentos (autenticado)

```bash
curl -H "Authorization: Bearer {token}" \
  http://localhost:8000/api/v1/subscriptions/payment-history?limit=5
```

### 3. Testar webhook (sandbox Asaas)

```bash
curl -X POST http://localhost:8000/api/v1/webhooks/asaas \
  -H "Content-Type: application/json" \
  -H "Asaas-Signature: {signature}" \
  -d '{
    "event": "PAYMENT_RECEIVED",
    "payment": {
      "id": "pay_123",
      "value": 29.90,
      "status": "RECEIVED",
      "dueDate": "2025-02-01",
      "paymentDate": "2025-02-01",
      "externalReference": "org-123"
    }
  }'
```

**Nota:** A assinatura deve ser calculada usando HMAC-SHA256 com o `ASAAS_WEBHOOK_SECRET`.
