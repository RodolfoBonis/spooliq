# 🚨 CRITICAL: Multi-Tenancy Fix Required

## Problema Identificado

O sistema atualmente tem multi-tenancy implementado **APENAS** na tabela `companies`, mas **TODAS as outras tabelas** não têm `organization_id`, o que significa:

❌ **ISOLAMENTO DE DADOS QUEBRADO**
- Todas as organizações veem os mesmos filaments
- Todas as organizações veem os mesmos customers  
- Todas as organizações veem os mesmos budgets
- Todas as organizações veem os mesmos brands/materials/presets

Isso **invalida completamente o multi-tenancy**!

---

## Solução: Adicionar `organization_id` em TODAS as tabelas

### Tabelas que PRECISAM de `organization_id`:

1. ✅ `companies` (já tem)
2. 🔄 `customers` (EM PROGRESSO)
3. ⏳ `budgets`
4. ⏳ `budget_items`
5. ⏳ `filaments`
6. ⏳ `brands`
7. ⏳ `materials`
8. ⏳ `machine_presets`
9. ⏳ `energy_presets`
10. ⏳ `cost_presets`

---

## Implementação Sistemática

### 1. Por Feature

Para cada feature, preciso:

#### A. Entity Update
- Adicionar campo `OrganizationID string` na entity
- Manter `OwnerUserID` para audit trail
- `OrganizationID` será o filtro principal

#### B. Model Update  
- Adicionar campo no GORM model
- Criar índice: `index:idx_{table}_org`
- Se tiver unique constraint em campos como email, tornar composto: `uniqueIndex:idx_{table}_org_email`
- Atualizar `ToEntity()` e `FromEntity()`

#### C. Repository Update
- Adicionar filtro `.Where("organization_id = ?", organizationID)` em TODOS os queries
- Remover lógica de `isAdmin` vs `owner_user_id` 
- Usar apenas `organization_id` como filtro principal
- Admins veem toda a organização, não todas as organizações

#### D. Use Case Update
- Extrair `organization_id` do contexto usando helper `getOrganizationID(c)`
- Passar `organization_id` para o repository ao invés de `userID` e `isAdmin`
- Validar que `organization_id` existe no contexto

---

## Mudanças no Comportamento

### Antes (ERRADO):
```go
// Repository
func FindAll(ctx, userID string, isAdmin bool) ([]*Entity, error) {
    query := db.WithContext(ctx)
    if !isAdmin {
        query = query.Where("owner_user_id = ?", userID)
    }
    // Admin vê TUDO de TODAS as organizações ❌
}
```

### Depois (CORRETO):
```go
// Repository  
func FindAll(ctx, organizationID string) ([]*Entity, error) {
    // SEMPRE filtra por organization_id
    return db.WithContext(ctx).
        Where("organization_id = ?", organizationID).
        Find(&entities)
    // Todos da mesma organização veem os mesmos dados ✅
    // Isolamento total entre organizações ✅
}
```

---

## Status por Feature

### 1. Customer ✅ CONCLUÍDO
- [x] Entity atualizada
- [x] Model atualizado
- [x] ToEntity/FromEntity atualizados
- [ ] Repository atualizado
- [ ] Use Cases atualizados

### 2. Budget ⏳ PENDENTE
- [ ] Entity
- [ ] Model
- [ ] Repository
- [ ] Use Cases

### 3. Filament ⏳ PENDENTE
- [ ] Entity
- [ ] Model
- [ ] Repository
- [ ] Use Cases

### 4. Brand ⏳ PENDENTE
- [ ] Entity
- [ ] Model
- [ ] Repository
- [ ] Use Cases

### 5. Material ⏳ PENDENTE
- [ ] Entity
- [ ] Model
- [ ] Repository
- [ ] Use Cases

### 6. Presets ⏳ PENDENTE
- [ ] Machine Preset Entity
- [ ] Energy Preset Entity
- [ ] Cost Preset Entity
- [ ] Models
- [ ] Repository
- [ ] Use Cases

---

## Checklist de Validação

Após implementação completa:

- [ ] Compilação OK
- [ ] Migrations executadas
- [ ] Testes com 2 organizações diferentes
- [ ] Verificar isolamento de dados
- [ ] Verificar que Admin não vê outras organizações
- [ ] Atualizar documentação do Keycloak
- [ ] Criar script de migração de dados existentes

---

## Migration de Dados Existentes

Para dados já existentes no banco:

```sql
-- Opção 1: Atribuir todos os dados a uma org default
UPDATE customers SET organization_id = 'default_org' WHERE organization_id IS NULL;
UPDATE budgets SET organization_id = 'default_org' WHERE organization_id IS NULL;
-- ... etc

-- Opção 2: Associar com base no owner_user_id
-- Requer tabela de mapeamento user -> organization
```

---

## Impacto

### Alto Impacto
- **TODAS** as queries precisam ser atualizadas
- **TODOS** os repositories precisam mudar
- **TODOS** os use cases precisam ser atualizados
- Migrations em TODAS as tabelas

### Benefícios
- ✅ Isolamento real de dados
- ✅ True multi-tenancy
- ✅ Segurança aprimorada
- ✅ Conformidade com arquitetura multi-tenant

---

## Estimativa

- Customer: ~30 min ✅ DONE (entities e models)
- Budget: ~45 min (mais complexo)
- Filament: ~30 min
- Brand: ~20 min  
- Material: ~20 min
- Presets: ~40 min (3 entidades)
- Testing: ~30 min

**Total: ~4 horas de trabalho**

---

## Ordem de Implementação

1. ✅ Customer (entity e model done)
2. Customer (repository e use cases)
3. Budget (critical - tem PDF e relacionamentos)
4. Brand e Material (usado por Filament)
5. Filament
6. Presets (machine, energy, cost)
7. Testing completo
8. Migration de dados
9. Documentação

---

## Próximos Passos IMEDIATOS

1. Finalizar Customer (repository + use cases)
2. Fazer commit incremental
3. Aplicar pattern para Budget
4. Aplicar pattern para outras features
5. Testar isolamento
6. Documentar processo

**ESTA É UMA CORREÇÃO CRÍTICA DE SEGURANÇA E ARQUITETURA!**

