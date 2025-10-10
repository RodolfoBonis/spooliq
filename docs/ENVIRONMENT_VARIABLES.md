# Environment Variables Documentation

This document lists all environment variables required for the Spooliq API.

## Application

| Variable | Description | Default | Required |
|----------|-------------|---------|----------|
| `APP_PORT` | HTTP server port | `8080` | No |
| `APP_ENV` | Environment (development/staging/production) | `development` | No |

## Database (PostgreSQL)

| Variable | Description | Default | Required |
|----------|-------------|---------|----------|
| `DB_HOST` | PostgreSQL host | `localhost` | Yes |
| `DB_PORT` | PostgreSQL port | `5432` | Yes |
| `DB_USER` | Database username | `postgres` | Yes |
| `DB_PASSWORD` | Database password | `postgres` | Yes |
| `DB_NAME` | Database name | `spooliq` | Yes |
| `DB_SSLMODE` | SSL mode (disable/require/verify-ca/verify-full) | `disable` | No |

## Keycloak Authentication

| Variable | Description | Default | Required |
|----------|-------------|---------|----------|
| `KEYCLOAK_HOST` | Keycloak server URL | `http://localhost:8181` | Yes |
| `REALM` | Keycloak realm name | `spooliq` | Yes |
| `CLIENT_ID` | Keycloak client ID | `spooliq` | Yes |
| `CLIENT_SECRET` | Keycloak client secret (confidential client) | - | Yes |
| `KEYCLOAK_ADMIN_USER` | Keycloak admin username | `admin` | Yes |
| `KEYCLOAK_ADMIN_PASSWORD` | Keycloak admin password | `admin` | Yes |

**Notes:**
- `CLIENT_SECRET` must be generated in Keycloak admin console
- Required for user creation during company registration
- Admin credentials are for Keycloak Admin API access

## Redis Cache

| Variable | Description | Default | Required |
|----------|-------------|---------|----------|
| `REDIS_HOST` | Redis server host | `localhost` | Yes |
| `REDIS_PORT` | Redis server port | `6379` | Yes |
| `REDIS_PASSWORD` | Redis password (leave empty if none) | - | No |
| `REDIS_DB` | Redis database number | `0` | No |

## Message Queue (RabbitMQ)

| Variable | Description | Default | Required |
|----------|-------------|---------|----------|
| `AMQP_CONNECTION` | RabbitMQ connection string | `amqp://guest:guest@localhost:5672/` | Yes |

## Sentry (Error Tracking)

| Variable | Description | Default | Required |
|----------|-------------|---------|----------|
| `SENTRY_DSN` | Sentry DSN for error tracking | - | No |
| `SENTRY_ENVIRONMENT` | Environment name for Sentry | `development` | No |
| `SENTRY_SAMPLE_RATE` | Performance monitoring sample rate (0.0-1.0) | `0.1` | No |

## Observability (OpenTelemetry)

| Variable | Description | Default | Required |
|----------|-------------|---------|----------|
| `OBSERVABILITY_ENABLED` | Enable OpenTelemetry tracing | `false` | No |
| `OTEL_SERVICE_NAME` | Service name for traces | `spooliq-api` | No |
| `OTEL_SERVICE_VERSION` | Service version | `1.0.0` | No |
| `OTEL_EXPORTER_OTLP_ENDPOINT` | OTLP endpoint (Jaeger/Tempo/etc) | `http://localhost:4318` | No |
| `OTEL_TRACE_SAMPLE_RATE` | Trace sampling rate (0.0-1.0) | `1.0` | No |

## CDN / File Storage

| Variable | Description | Default | Required |
|----------|-------------|---------|----------|
| `CDN_BASE_URL` | CDN base URL | `https://rb-cdn.rodolfodebonis.com.br` | Yes |
| `CDN_API_KEY` | CDN API authentication key | - | Yes |
| `CDN_BUCKET` | CDN bucket name | `spooliq` | Yes |

**Notes:**
- Used for uploading company logos, PDFs, and 3MF files
- API documentation: https://rb-cdn.rodolfodebonis.com.br/v1/docs

## Asaas Payment Gateway

| Variable | Description | Default | Required |
|----------|-------------|---------|----------|
| `ASAAS_API_KEY` | Asaas API authentication key | - | Yes |
| `ASAAS_BASE_URL` | Asaas API base URL | `https://sandbox.asaas.com/api/v3` | Yes |
| `ASAAS_WEBHOOK_SECRET` | Asaas webhook signature secret | - | **Critical** |

**Notes:**
- For production, use `https://api.asaas.com/v3`
- `ASAAS_WEBHOOK_SECRET` is required for webhook signature validation (HMAC-SHA256)
- Generate webhook secret in Asaas dashboard webhook settings
- **Security Critical:** Never commit this secret or use API key as webhook secret

## Email Service

| Variable | Description | Default | Required |
|----------|-------------|---------|----------|
| `SMTP_HOST` | SMTP server host | - | No (TODO) |
| `SMTP_PORT` | SMTP server port | - | No (TODO) |
| `SMTP_USERNAME` | SMTP authentication username | - | No (TODO) |
| `SMTP_PASSWORD` | SMTP authentication password | - | No (TODO) |
| `SMTP_FROM_EMAIL` | Default "from" email address | - | No (TODO) |
| `SMTP_FROM_NAME` | Default "from" name | - | No (TODO) |

**Notes:**
- Email service implementation is pending
- Can be configured with SendGrid, AWS SES, or standard SMTP

## Logging

| Variable | Description | Default | Required |
|----------|-------------|---------|----------|
| `LOG_LEVEL` | Logging level (debug/info/warn/error) | `info` | No |
| `LOG_FORMAT` | Log format (json/text) | `json` | No |

## CORS

| Variable | Description | Default | Required |
|----------|-------------|---------|----------|
| `CORS_ALLOWED_ORIGINS` | Comma-separated list of allowed origins | `http://localhost:3000` | No |

## SaaS Configuration

| Variable | Description | Default | Required |
|----------|-------------|---------|----------|
| `TRIAL_PERIOD_DAYS` | Trial period duration in days | `14` | No |
| `SUBSCRIPTION_MONTHLY_PRICE` | Monthly subscription price (BRL) | `99.90` | No |
| `PLATFORM_COMPANY_ORG_ID` | Platform company organization UUID | - | Yes |

**Notes:**
- `PLATFORM_COMPANY_ORG_ID` should be set to your own company's organization UUID
- This company is exempt from subscription checks (`IsPlatformCompany = true`)

## Feature Flags

| Variable | Description | Default | Required |
|----------|-------------|---------|----------|
| `FEATURE_WEBHOOKS_ENABLED` | Enable Asaas webhooks | `true` | No |
| `FEATURE_SUBSCRIPTION_CHECK_ENABLED` | Enable subscription middleware | `true` | No |
| `FEATURE_CDN_UPLOADS_ENABLED` | Enable CDN file uploads | `true` | No |

---

## Setup Instructions

### 1. Create .env file

```bash
cp .env.example .env
```

### 2. Configure Keycloak

1. Access Keycloak admin console: http://localhost:8181
2. Create realm: `spooliq`
3. Create client: `spooliq`
   - Access Type: `confidential`
   - Copy Client Secret
4. Create roles: `user`, `adm`, `PlatformAdmin`
5. Create client scope: `organization`
6. Add protocol mapper: `organization_id` (group attribute mapper)

### 3. Configure Asaas

1. Sign up at: https://www.asaas.com/
2. Get API Key from dashboard (sandbox or production)
3. Configure webhook URL: `https://your-domain.com/v1/webhooks/asaas`
4. Generate and configure webhook secret
5. Enable webhook events:
   - PAYMENT_RECEIVED
   - PAYMENT_CONFIRMED
   - PAYMENT_OVERDUE
   - PAYMENT_REFUNDED

### 4. Configure CDN

1. Access: https://rb-cdn.rodolfodebonis.com.br/v1/docs
2. Get API Key
3. Use bucket: `spooliq`

### 5. Initialize Database

```bash
# Run migrations automatically on startup
go run cmd/main.go
```

---

## Security Checklist

- [ ] Never commit `.env` file
- [ ] Use strong passwords in production
- [ ] Rotate secrets regularly
- [ ] Use different keys for sandbox and production
- [ ] Enable HTTPS in production
- [ ] Configure proper CORS origins
- [ ] Use secret managers in production (AWS Secrets Manager, Azure Key Vault, etc.)
- [ ] Validate all webhook signatures
- [ ] Monitor Sentry for security events
- [ ] Enable observability in production

---

## Production Deployment

### Recommended Changes for Production:

1. **Database:**
   - Use managed PostgreSQL (AWS RDS, Azure Database, etc.)
   - Enable SSL: `DB_SSLMODE=require`
   - Use strong passwords

2. **Keycloak:**
   - Use managed Keycloak or hosted solution
   - Enable HTTPS
   - Configure proper realm settings

3. **Asaas:**
   - Use production API: `ASAAS_BASE_URL=https://api.asaas.com/v3`
   - Use production API keys
   - Use separate webhook secret (never use API key)

4. **CDN:**
   - Use production CDN
   - Configure CDN cache settings

5. **Observability:**
   - Enable OpenTelemetry: `OBSERVABILITY_ENABLED=true`
   - Configure proper OTLP endpoint (Jaeger, Tempo, Grafana Cloud, etc.)
   - Enable Sentry

6. **Email:**
   - Configure production email service (SendGrid, AWS SES, etc.)
   - Use verified domain

7. **Secrets:**
   - Use environment variables or secret managers
   - Never hardcode secrets
   - Rotate secrets regularly

---

## Troubleshooting

### Common Issues:

1. **Keycloak connection failed:**
   - Check `KEYCLOAK_HOST` is correct
   - Verify Keycloak is running
   - Check `CLIENT_ID` and `CLIENT_SECRET`

2. **Database connection failed:**
   - Verify PostgreSQL is running
   - Check `DB_HOST`, `DB_PORT`, `DB_USER`, `DB_PASSWORD`
   - Verify database exists

3. **Webhook signature validation failed:**
   - Verify `ASAAS_WEBHOOK_SECRET` is correct
   - Check webhook configuration in Asaas dashboard
   - Test with Asaas webhook simulator

4. **CDN upload failed:**
   - Verify `CDN_API_KEY` is correct
   - Check `CDN_BUCKET` exists
   - Verify network connectivity

---

## Support

For issues or questions:
- GitHub Issues: https://github.com/RodolfoBonis/spooliq/issues
- Documentation: `/docs`
- API Documentation: `http://localhost:8080/v1/docs`

