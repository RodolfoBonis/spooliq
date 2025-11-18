# 🎯 SpoolIQ Release Workflow - Resumo Executivo

## ✅ O QUE FOI IMPLEMENTADO

### Modelo: Quick Release (develop = next release)
- **Ciclo:** 2-8 horas do início ao fim
- **Filosofia:** Releases pequenos e frequentes
- **Feature Cutoff:** Quando release branch é criada, define o que vai ou não para produção

---

## 🔄 FLUXO COMPLETO EM 10 PASSOS

### 1. **PREPARAR RELEASE** (Manual - 5min)
```
Ação: Developer/PM vai em Actions → "Prepare Release" → Run workflow

O que acontece:
✅ Cria branch release/v2.2.0 de develop
✅ Atualiza version.txt (2.1.1 → 2.2.0)
✅ Gera CHANGELOG.md
✅ CRIA TAG v2.2.0 (ANTES do PR!)
✅ Push branch + tag
✅ Cria PR para main
✅ Dispara staging deploy
✅ 🚨 FEATURE CUTOFF anunciado!

Resultado:
- PR criado: "🚀 Release v2.2.0"
- Tag v2.2.0 existe no GitHub
- Staging deployando
- Time notificado via Telegram
```

### 2. **FEATURE CUTOFF** (Automático - Instantâneo)
```
O que significa:
- Features em develop AGORA → vão para v2.2.0
- Features merged DEPOIS → vão para v2.3.0

Exemplo Timeline:
10:00 AM - Feature A merged ✅ (vai para v2.2.0)
11:00 AM - Release criada 🚨 CUTOFF!
12:00 PM - Feature B merged ❌ (vai para v2.3.0)

Por quê: QA testa um conjunto fixo de features. Sem mudanças durante QA!
```

### 3. **STAGING DEPLOY** (Automático - 5-10min)
```
O que acontece:
✅ release-staging.yaml detecta push para release/v2.2.0
✅ Build Docker image
✅ Push para ECR com tag: 2.2.0-rc.1737204000
✅ Update K8s manifests (staging)
✅ ArgoCD sync staging cluster

Resultado:
- Staging rodando versão 2.2.0-rc.xxx
- QA pode começar validação
- URL: staging.example.com
```

### 4. **QA VALIDAÇÃO** (Manual - 2-4h)
```
QA Team testa em staging:
☐ Novas features funcionam
☐ Sem regressões
☐ Performance OK
☐ UI correta (desktop + mobile)
☐ Integrações funcionando

Quando pronto:
✅ QA aprova o PR no GitHub
```

### 5. **MERGE PR** (Manual - 1min)
```
⚠️ IMPORTANTE: Merge é MANUAL (não auto-merge!)

Developer/PM:
1. Vai no PR "🚀 Release v2.2.0"
2. Verifica:
   ✅ QA aprovado
   ✅ CI passou
   ✅ Sem conflitos
3. Click "Merge pull request"
4. Escolhe "Merge commit" (não squash!)
5. Confirma

Resultado:
- main branch atualizado com release
- Trigger automático do próximo passo
```

### 6. **ORQUESTRAÇÃO POST-MERGE** (Automático - 1min)
```
post-merge-release.yaml detecta merge:

O que faz:
✅ Detecta PR release/* merged para main
✅ Extrai versão do nome da branch (v2.2.0)
✅ VALIDA que tag v2.2.0 existe
✅ Cria labels se necessário
✅ Dispara release.yaml com tag v2.2.0

Resultado:
- release.yaml vai executar
- Time notificado: "Production deployment iniciado"
```

### 7. **VALIDAÇÃO DE VERSÃO** (Automático - 1min)
```
🆕 NOVO: Validação rigorosa antes de deploy!

release.yaml job "validate":

Checks:
✅ Tag formato válido? (v2.2.0 ✓)
✅ Input version = tag version? (2.2.0 = 2.2.0 ✓)
✅ version.txt = tag version? (lê arquivo no tag)

Se QUALQUER check falhar:
❌ Deployment PARA
❌ Team notificado do erro
❌ Produção protegida!

Se todos passarem:
✅ Continua para deploy
```

### 8. **PRODUCTION DEPLOY** (Automático - 15-30min)
```
release.yaml job "release":

Passos:
1. Checkout tag v2.2.0
2. Setup Go
3. Configure AWS/ECR
4. Run GoReleaser
   ├─ Build binaries
   ├─ Create archives
   ├─ Build Docker image
   └─ Push to ECR: spooliq:2.2.0
5. Update K8s manifests (production)
6. Sync ArgoCD production cluster
7. Calculate build time
8. Notify success

Resultado:
✅ Production rodando v2.2.0
✅ GitHub Release criado
✅ Docker image: spooliq:2.2.0
✅ Team notificado: "Deploy SUCCESS!"
```

### 9. **BACKPORT PARA DEVELOP** (Automático - 5-10min)
```
release.yaml job "backport":

O que faz:
1. Checkout main
2. Cria branch: backport/v2.2.0-to-develop
3. Merge main → backport branch
4. Push backport branch
5. Cria PR para develop
   ├─ Labels: backport, automated, auto-merge
   └─ Auto-merge: HABILITADO

auto-merge.yaml detecta:
✅ PR tem label "auto-merge"
✅ CI passou
✅ Sem conflitos
→ Auto-aprova e merge!

Resultado:
- develop atualizado com código de produção
- develop pronto para próximas features (v2.3.0)
```

### 10. **STAGING NÃO RE-DEPLOYA** (Automático - Bloqueado)
```
🆕 NOVO: Staging bloqueia backports!

Quando backport é merged para develop:
- Commit message contém "backport"
- release-staging.yaml tem condição:
  if: !contains(commit.message, 'backport')
- Staging deploy É BLOQUEADO ✅

Por quê:
- Código já foi testado em staging antes
- Não precisa re-deployar o que já está em produção
- Economiza recursos e tempo
```

---

## ⏱️ TIMELINE TÍPICA

```
T+0h:00m  → Release preparada (manual)
T+0h:05m  → Staging deployed (auto)
T+0h:10m  → QA inicia testes
T+4h:00m  → QA aprova (manual)
T+4h:01m  → PR merged (manual)
T+4h:02m  → Post-merge orchestration (auto)
T+4h:03m  → Validation passes (auto)
T+4h:05m  → Production deploy starts (auto)
T+4h:25m  → Production deployed ✅ (auto)
T+4h:30m  → Backport PR created (auto)
T+4h:35m  → Backport auto-merged (auto)
───────────────────────────────────────────
T+5h:00m  → 🎉 RELEASE COMPLETO!
```

**Ações manuais:** 3 (prepare, QA approve, merge PR)
**Ações automáticas:** 15+ passos

---

## 🚨 HOTFIX FLOW (1-2h)

Mesmo fluxo, mas:
- Parte de `main` (não develop)
- Auto-increment patch (2.2.0 → 2.2.1)
- Label `priority:critical`
- QA expedida (30min-1h)
- Timeline comprimido

---

## 🎯 PRINCIPAIS DIFERENÇAS DO FLUXO ANTIGO

### ❌ ANTES (Problemático)

1. **Double Version Increment**
   - prepare-release: 2.0.4 → 2.1.0
   - release.yaml: 2.1.0 → 2.1.1 (INCREMENTA DE NOVO!)
   - Production: v2.1.1 (errado!)

2. **3 Triggers Conflitantes**
   - PR merge → trigger
   - CI complete → trigger
   - Tag push → trigger
   - Mesma release roda 3x! Race conditions!

3. **Tag Criada Tarde**
   - PR criado (sem tag)
   - QA valida
   - PR merged
   - release.yaml cria tag (tarde demais!)

4. **Sem Validação**
   - Nenhum check de consistência
   - version.txt desincronizado
   - Deployments cegos

5. **Staging Duplicado**
   - Backport merge → staging re-deploya
   - Desperdício de recursos

### ✅ AGORA (Corrigido)

1. **Single Version Increment**
   - prepare-release: 2.1.1 → 2.2.0 (uma vez!)
   - Tag v2.2.0 criada
   - release.yaml: USA o tag (não incrementa)
   - Production: v2.2.0 (correto!)

2. **Single Trigger**
   - Apenas: workflow_dispatch via post-merge-release
   - Executa 1x exatamente
   - Sem race conditions

3. **Tag Criada Cedo**
   - Tag criada ANTES do PR
   - QA pode validar o tag diretamente
   - Validação antes de deploy

4. **Validação Rigorosa**
   - Job "validate" antes de qualquer deploy
   - Verifica: tag = version.txt
   - Falha rápido se inconsistente

5. **Staging Inteligente**
   - Backports bloqueados
   - Sem deploys duplicados

---

## 📊 COMPONENTES E RESPONSABILIDADES

### prepare-release.yaml (O Iniciador)
```
Responsável por:
✓ Determinar versão
✓ Criar release branch
✓ Atualizar version.txt
✓ Gerar CHANGELOG
✓ CRIAR TAG (cedo!)
✓ Criar PR
✓ Notificar feature cutoff
```

### post-merge-release.yaml (O Orquestrador)
```
Responsável por:
✓ Detectar merge de release/hotfix
✓ Validar tag existe
✓ Disparar release.yaml
✓ Single point of entry!
```

### release.yaml (O Deployer)
```
Responsável por:
✓ VALIDAR consistência
✓ Build com GoReleaser
✓ Deploy para ECR
✓ Sync ArgoCD production
✓ Criar backport PR
✗ NÃO incrementa versão
✗ NÃO cria tags
```

### release-staging.yaml (O Validador)
```
Responsável por:
✓ Deploy em staging
✓ BLOQUEAR backports
✓ Ambiente de QA
```

### auto-merge.yaml (O Automatizador)
```
Responsável por:
✓ Auto-merge backports
✓ Auto-merge dependabot
✗ NÃO auto-merge releases!
```

### notify-release-cutoff.yaml (O Comunicador)
```
Responsável por:
✓ Notificar feature cutoff
✓ Avisar time do snapshot
```

---

## 🎓 FEATURE CUTOFF - EXEMPLO PRÁTICO

```
Segunda-feira:
─────────────────────────────────────────────────────────────
9:00  → Dev A merge Feature Login Social para develop ✅
10:00 → Dev B merge Feature Dark Mode para develop ✅

11:00 → PM cria release/v2.3.0
        🚨 FEATURE CUTOFF AGORA!

        Snapshot contém:
        ✅ Login Social (está em develop)
        ✅ Dark Mode (está em develop)

12:00 → Dev C merge Feature Chat Real-time para develop

        ⚠️ Chat vai para v2.4.0!

        Por quê? Release v2.3.0 já foi criada às 11:00.
        O snapshot não inclui Chat.

16:00 → QA valida v2.3.0 em staging
        Testa: Login Social + Dark Mode
        NÃO testa Chat (não está nessa release)

17:00 → v2.3.0 vai para produção
        Contém: Login Social + Dark Mode
        Chat vai na próxima (v2.4.0)
```

**Por que esse modelo funciona:**
- QA sabe exatamente o que testar (scope fixo)
- Sem "só mais uma feature" que atrasa releases
- Ciclos rápidos (Chat vai em 1-2 dias na próxima!)
- Comunicação clara

---

## 🔍 VALIDAÇÃO - EXEMPLO

```
Input para release.yaml:
- tag: v2.2.0
- version: 2.2.0

Validação executa:

CHECK 1: Tag formato válido?
━━━━━━━━━━━━━━━━━━━━━━━━━━━━
Pattern: ^v[0-9]+\.[0-9]+\.[0-9]+$
v2.2.0 matches? ✅ YES

CHECK 2: Input version = tag version?
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
Tag: v2.2.0 → remove v → 2.2.0
Input: 2.2.0
2.2.0 == 2.2.0? ✅ YES

CHECK 3: File version = tag version?
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
git checkout v2.2.0
cat version.txt → "2.2.0"
2.2.0 == 2.2.0? ✅ YES

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
✅ ALL CHECKS PASSED!
Safe to deploy to production!
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

Se qualquer check falhar:
❌ Workflow PARA imediatamente
❌ Nenhum deploy acontece
❌ Team é notificado do erro
```

---

## 📱 NOTIFICAÇÕES TELEGRAM

```
1. Release preparada (prepare-release.yaml)
   🚀 Release v2.2.0 preparada!
   ⚠️ FEATURE CUTOFF ativo
   🧪 Staging: deployando
   📋 PR: [link]

2. Feature cutoff (notify-release-cutoff.yaml)
   📸 Snapshot de develop capturado
   ✅ Features IN: Tudo em develop AGORA
   ❌ Features OUT: Merges futuros

3. Deploy iniciado (post-merge-release.yaml)
   🚀 Deploy production iniciado para v2.2.0
   PR #123 merged por @developer

4. Deploy sucesso (release.yaml)
   ✅ Production deployed!
   Version: v2.2.0
   Build time: 5m 23s
   Release: [GitHub link]

5. Backport criado (release.yaml)
   🔄 Backport PR criado
   Target: develop
   Auto-merge: enabled
```

---

## 🛠️ PRÓXIMOS PASSOS

### 1. Push para GitHub ✅
```bash
git push origin develop
```

### 2. Sync Labels (Opcional)
```bash
gh label sync -f .github/labels.yaml
```

### 3. Comunicar Time
```markdown
📢 NOVO FLUXO DE RELEASE!

Quick releases (2-8h)
develop = next release
Feature cutoff model

📚 Docs completas em:
- .github/README.md
- .github/RELEASE_PROCESS.md
- .github/FLOW_DIAGRAM.md

🧪 Test release será feita antes de usar
```

### 4. Test Release (RECOMENDADO!)
```
Fazer uma release v2.2.0-test ou v2.1.2-test
para validar todo o fluxo antes de usar em
releases reais de produção.
```

---

## ❓ FAQ

**P: E se eu quiser que uma feature vá na release atual?**
R: Merge ANTES de criar a release branch. Se já criou, cherry-pick (não recomendado) ou espera próxima release (quick cycle!).

**P: Como sei se feature cutoff está ativo?**
R: Checa se existe PR de release aberto para main. Se sim, cutoff ativo.

**P: Posso fazer rollback?**
R: Sim! Trigger release.yaml manualmente com tag antiga (ex: v2.1.1).

**P: E se validação falhar?**
R: Workflow para. Corrige version.txt ou recria tag. Tenta de novo.

**P: Backports sempre funcionam?**
R: 99% sim (auto-merge). Se houver conflito, resolve manualmente.

**P: Posso pular staging?**
R: NÃO! Staging validation é obrigatória. Sem atalhos.

---

## 📚 DOCUMENTAÇÃO COMPLETA

1. **README.md** - Overview e troubleshooting
2. **RELEASE_PROCESS.md** - Runbook detalhado
3. **FLOW_DIAGRAM.md** - Diagramas visuais ASCII
4. **RELEASE_SUMMARY.md** - Este arquivo

---

**Criado:** 2025-01-18
**Modelo:** Quick Release (develop = next release)
**Status:** ✅ Pronto para uso (após test release)
