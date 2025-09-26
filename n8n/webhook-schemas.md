# n8n CI/CD Webhook Payload Schemas

Este documento define os schemas dos payloads enviados pelos GitHub Actions para o webhook do n8n.

## Webhook Endpoint

```
POST /webhook/spooliq-ci-cd
Authorization: Bearer <N8N_API_TOKEN>
Content-Type: application/json
```

## Event Types

### 1. CI Success (`ci_success`)

Enviado quando o build/teste passa com sucesso.

```json
{
  "event_type": "ci_success",
  "repository": "RodolfoBonis/spooliq",
  "workflow": "CI - spooliq",
  "commit_sha": "a1b2c3d4e5f6",
  "tag": null,
  "branch": "refs/heads/main",
  "actor": "RodolfoBonis",
  "timestamp": "2024-01-15T10:30:00Z",
  "details": {
    "build_time": "2m 30s",
    "test_results": "all passed",
    "coverage": "85%"
  },
  "telegram_chat_id": "-123456789",
  "telegram_thread_id": "42"
}
```

### 2. CI Failure (`ci_failure`)

Enviado quando o build/teste falha.

```json
{
  "event_type": "ci_failure",
  "repository": "RodolfoBonis/spooliq",
  "workflow": "CI - spooliq",
  "commit_sha": "a1b2c3d4e5f6",
  "tag": null,
  "branch": "refs/heads/main", 
  "actor": "RodolfoBonis",
  "timestamp": "2024-01-15T10:30:00Z",
  "details": {
    "error_message": "Tests failed in auth module",
    "logs_url": "https://github.com/RodolfoBonis/spooliq/actions/runs/12345",
    "failed_tests": ["TestAuthMiddleware", "TestJWTValidation"]
  },
  "telegram_chat_id": "-123456789",
  "telegram_thread_id": "42"
}
```

### 3. Deploy Success (`deploy_success`)

Enviado quando o deploy é realizado com sucesso via GoReleaser.

```json
{
  "event_type": "deploy_success",
  "repository": "RodolfoBonis/spooliq",
  "workflow": "Release with GoReleaser",
  "commit_sha": "a1b2c3d4e5f6",
  "tag": "v1.2.3",
  "branch": "refs/heads/main",
  "actor": "RodolfoBonis", 
  "timestamp": "2024-01-15T10:45:00Z",
  "details": {
    "docker_image": "123456789.dkr.ecr.us-east-1.amazonaws.com/rodolfobonis/spooliq:1.2.3",
    "argocd_sync": "success",
    "build_time": "5m 12s",
    "release_url": "https://github.com/RodolfoBonis/spooliq/releases/tag/v1.2.3"
  },
  "telegram_chat_id": "-123456789",
  "telegram_thread_id": "42"
}
```

### 4. Deploy Failure (`deploy_failure`)

Enviado quando o deploy falha via GoReleaser.

```json
{
  "event_type": "deploy_failure",
  "repository": "RodolfoBonis/spooliq",
  "workflow": "Release with GoReleaser",
  "commit_sha": "a1b2c3d4e5f6",
  "tag": "v1.2.3",
  "branch": "refs/heads/main",
  "actor": "RodolfoBonis",
  "timestamp": "2024-01-15T10:45:00Z",
  "details": {
    "error_message": "GoReleaser deployment failed - check GitHub Actions logs",
    "logs_url": "https://github.com/RodolfoBonis/spooliq/actions/runs/12346",
    "job_status": "failure"
  },
  "telegram_chat_id": "-123456789",
  "telegram_thread_id": "42"
}
```

## ⚡ Workflow Simplificado

**Agora usando apenas GoReleaser:**
- ✅ Removido workflow CD separado
- ✅ GoReleaser funciona para push em `main` E tags
- ✅ Auto-incremento de versão quando push em `main`
- ✅ Release manual quando push de tag `v*`

## Campos Comuns

### Campos Obrigatórios

| Campo | Tipo | Descrição |
|-------|------|-----------|
| `event_type` | string | Tipo do evento: `ci_success`, `ci_failure`, `deploy_success`, `deploy_failure` |
| `repository` | string | Nome completo do repositório (owner/repo) |
| `workflow` | string | Nome do workflow do GitHub Actions |
| `commit_sha` | string | SHA do commit (primeiros 7 caracteres) |
| `branch` | string | Referência completa da branch/tag |
| `actor` | string | Usuário que triggou o workflow |
| `timestamp` | string | Timestamp ISO 8601 UTC |

### Campos Opcionais

| Campo | Tipo | Descrição |
|-------|------|-----------|
| `tag` | string\|null | Tag de release (apenas para deploys) |
| `details` | object | Detalhes específicos do evento |
| `telegram_chat_id` | string | ID do chat do Telegram |
| `telegram_thread_id` | string | ID da thread do Telegram |

## Detalhes por Tipo de Evento

### CI Success Details
- `build_time`: Tempo de build
- `test_results`: Resultado dos testes
- `coverage`: Cobertura de código (opcional)

### CI Failure Details  
- `error_message`: Mensagem de erro resumida
- `logs_url`: URL dos logs no GitHub Actions
- `failed_tests`: Array com nomes dos testes que falharam (opcional)

### Deploy Success Details
- `docker_image`: URL completa da imagem Docker
- `argocd_sync`: Status do sync do ArgoCD
- `build_time`: Tempo total de build/deploy
- `release_url`: URL da release no GitHub

### Deploy Failure Details
- `error_message`: Mensagem de erro resumida  
- `logs_url`: URL dos logs no GitHub Actions
- `job_status`: Status do job que falhou

## Configuração no n8n

### Secrets Necessários no GitHub

```bash
# Webhook do n8n
N8N_WEBHOOK_URL=https://your-n8n.domain.com/webhook/spooliq-ci-cd

# Token de autenticação do n8n (opcional)
N8N_API_TOKEN=your-api-token

# Configurações do Telegram (existentes)
CHAT_ID=-123456789
THREAD_ID=42
```

### Configurações de Credenciais no n8n

1. **Telegram Bot**: Configure as credenciais do bot do Telegram
2. **PostgreSQL**: Configure conexão com banco para histórico
3. **Slack** (opcional): Configure webhook do Slack para alertas críticos
4. **SigNoz** (opcional): Configure API token para anotações

## Exemplo de Resposta do Webhook

O webhook do n8n deve retornar:

```json
{
  "status": "received",
  "timestamp": "2024-01-15T10:30:00Z"
}
```