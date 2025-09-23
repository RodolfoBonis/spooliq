# SpoolIQ Migration System

O SpoolIQ agora possui um sistema de migrações moderno similar ao TypeORM, com suporte a arquivos SQL e CLI avançada.

## 🚀 Quick Start

### Comando Básicos

```bash
# Ver status das migrações
make db/status

# Executar todas as migrações pendentes
make db/migrate

# Criar nova migração
make db/create NAME="add_users_table"

# Listar todas as migrações disponíveis
make db/list

# Rollback da última migração
make db/rollback

# Rollback de múltiplas migrações
make db/rollback COUNT=3
```

## 📁 Estrutura de Arquivos

```
migrations/
├── 20250923000001_initial_schema/
│   ├── up.sql    # Script de migração
│   └── down.sql  # Script de rollback
├── 20250923000002_add_filament_fields/
│   ├── up.sql
│   └── down.sql
└── 20250923000003_add_filament_metadata_fields/
    ├── up.sql
    └── down.sql
```

## 🛠️ CLI Commands

### Usando o CLI Diretamente

```bash
# Build da ferramenta de migração
make migrate-build

# Criar nova migração
./migrate create "add_new_feature"

# Executar migrações
./migrate up

# Executar apenas a próxima migração
./migrate up:one

# Rollback
./migrate down
./migrate down 2  # Rollback 2 migrações

# Status das migrações
./migrate status

# Listar migrações
./migrate list

# Reset completo (CUIDADO!)
./migrate fresh

# Reset com rollback
./migrate reset
```

### Usando o Makefile

```bash
# Criar migração
make db/create NAME="migration_name"

# Executar migrações
make db/migrate

# Rollback
make db/rollback COUNT=1

# Status
make db/status

# Listar
make db/list

# Fresh (DROP ALL TABLES - CUIDADO!)
make db/fresh-confirm

# Reset (rollback all + rerun)
make db/reset-migrations
```

## 📝 Criando Migrações

### 1. Criar Nova Migração

```bash
make db/create NAME="add_user_preferences"
```

Isso criará:
- `migrations/20250923123456_add_user_preferences/up.sql`
- `migrations/20250923123456_add_user_preferences/down.sql`

### 2. Editar os Arquivos SQL

**up.sql** (aplicar mudanças):
```sql
-- Migration: add_user_preferences
-- Description: Add user preferences table
-- Generated: 2025-09-23

CREATE TABLE user_preferences (
    id SERIAL PRIMARY KEY,
    user_id VARCHAR(255) NOT NULL,
    theme VARCHAR(50) DEFAULT 'light',
    language VARCHAR(10) DEFAULT 'en',
    notifications BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_user_preferences_user_id ON user_preferences(user_id);
```

**down.sql** (reverter mudanças):
```sql
-- Rollback Migration: add_user_preferences
-- Description: Remove user preferences table
-- Generated: 2025-09-23

DROP INDEX IF EXISTS idx_user_preferences_user_id;
DROP TABLE IF EXISTS user_preferences;
```

### 3. Executar a Migração

```bash
make db/migrate
```

## 🔄 Workflows Comuns

### Desenvolvimento Local

```bash
# 1. Criar feature migration
make db/create NAME="add_feature_x"

# 2. Editar os arquivos SQL
# 3. Aplicar migração
make db/migrate

# 4. Testar se funcionou
make db/status
```

### Rollback de Mudanças

```bash
# Rollback da última migração
make db/rollback

# Rollback de múltiplas migrações
make db/rollback COUNT=3

# Ver status após rollback
make db/status
```

### Deploy em Produção

```bash
# 1. Verificar migrações pendentes
make db/status

# 2. Executar migrações
make db/migrate

# 3. Verificar se todas foram aplicadas
make db/status
```

### Reset Completo (Desenvolvimento)

```bash
# CUIDADO: Isso apaga TODOS os dados!
make db/fresh-confirm

# Alternativa mais segura: rollback + rerun
make db/reset-migrations
```

## 🆚 Sistema Legacy

### Executar Migrações Legacy

```bash
# Executar migrações Go antigas
make db/migrate-legacy

# Status das migrações legacy
make db/status-legacy
```

### Migração do Sistema Legacy

O sistema novo detecta automaticamente migrações legacy já aplicadas e as marca como executadas no novo sistema.

## ⚡ Features Avançadas

### Variáveis de Ambiente

```bash
# Personalizar diretório de migrações
export MIGRATIONS_PATH="custom_migrations"
make db/migrate
```

### Multiple SQL Statements

O sistema suporta múltiplos statements SQL separados por `;`:

```sql
CREATE TABLE users (id SERIAL PRIMARY KEY);
CREATE INDEX idx_users_id ON users(id);
INSERT INTO users DEFAULT VALUES;
```

### Comentários SQL

Use comentários para documentar:

```sql
-- Esta tabela armazena preferências dos usuários
CREATE TABLE user_preferences (
    -- ID único do usuário
    user_id VARCHAR(255) NOT NULL,
    /* 
     * Configurações do tema
     * Valores: 'light', 'dark', 'auto'
     */
    theme VARCHAR(50) DEFAULT 'light'
);
```

## 🚨 Cuidados Importantes

1. **Sempre teste migrações em desenvolvimento primeiro**
2. **Faça backup antes de executar em produção**
3. **Evite mudanças destrutivas sem rollback**
4. **Use transações para operações críticas**
5. **Teste os scripts de rollback**

## 🐛 Troubleshooting

### Migration Failed

```bash
# Ver logs detalhados
make db/status

# Rollback se necessário
make db/rollback

# Corrigir SQL e tentar novamente
make db/migrate
```

### Reset Database

```bash
# Reset completo (CUIDADO!)
make db/fresh-confirm

# Ou via CLI
./migrate fresh
# Digite 'yes' quando perguntado
```

### Schema Corruption

```bash
# 1. Backup do banco
pg_dump database_name > backup.sql

# 2. Reset completo
make db/fresh-confirm

# 3. Restore se necessário
psql database_name < backup.sql
```

## 📋 Examples

### Adicionar Coluna

```sql
-- up.sql
ALTER TABLE filaments ADD COLUMN description TEXT;

-- down.sql
ALTER TABLE filaments DROP COLUMN IF EXISTS description;
```

### Criar Índice

```sql
-- up.sql
CREATE INDEX IF NOT EXISTS idx_filaments_name ON filaments(name);

-- down.sql
DROP INDEX IF EXISTS idx_filaments_name;
```

### Inserir Dados

```sql
-- up.sql
INSERT INTO filament_brands (name, active) VALUES 
    ('Prusament', true),
    ('PETG', true)
ON CONFLICT (name) DO NOTHING;

-- down.sql
DELETE FROM filament_brands WHERE name IN ('Prusament', 'PETG');
```

### Modificar Tabela

```sql
-- up.sql
ALTER TABLE filaments 
    ALTER COLUMN price_per_kg TYPE DECIMAL(12,4),
    ADD CONSTRAINT check_price_positive CHECK (price_per_kg > 0);

-- down.sql
ALTER TABLE filaments 
    DROP CONSTRAINT IF EXISTS check_price_positive,
    ALTER COLUMN price_per_kg TYPE DECIMAL(10,2);
```

## 🎯 Best Practices

1. **Nomes descritivos**: `add_user_authentication` vs `migration_001`
2. **Migrations pequenas**: Uma mudança por migração
3. **Sempre reversível**: Todos as migrações devem ter rollback
4. **Testes**: Teste up + down em desenvolvimento
5. **Documentação**: Use comentários SQL explicativos
6. **Ordem**: Migrations executam em ordem cronológica
7. **Backup**: Sempre backup antes de mudanças destrutivas

---

**Happy migrating! 🚀**