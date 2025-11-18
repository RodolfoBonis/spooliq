# GitHub Workflows & Configuration

This directory contains GitHub Actions workflows and configuration for the SpoolIQ project.

## 🚀 Release Model: Develop = Next Release + Quick Releases (2-8h)

### Core Principles
1. **develop branch = next release** (always deployable)
2. **Release branch = snapshot** (feature cutoff when created)
3. **Quick QA cycle** (2-8 hours target)
4. **Automatic everything** (after human approval)

---

## 📂 Workflows

### Release Workflows

#### `prepare-release.yaml`
**Trigger:** Manual (`workflow_dispatch`)
**Purpose:** Prepare a new release from develop

**What it does:**
1. Creates `release/vX.X.X` branch from develop
2. Bumps version in `version.txt`
3. Updates `CHANGELOG.md`
4. **Creates tag immediately** (before PR)
5. Pushes branch and tag
6. Creates PR to main
7. Triggers staging deployment
8. **Announces FEATURE CUTOFF** to team

**Feature Cutoff:** Any features merged to develop AFTER this runs go to the NEXT release!

#### `post-merge-release.yaml`
**Trigger:** Automatic (when release/hotfix PR merged to main)
**Purpose:** Orchestrate production deployment

**What it does:**
1. Detects release/hotfix PR merge
2. Extracts version from branch name
3. Validates tag exists
4. Ensures labels exist
5. Triggers `release.yaml` with tag
6. Notifies team

#### `release.yaml`
**Trigger:** Automatic (via `post-merge-release.yaml`) or Manual
**Purpose:** Production deployment

**What it does:**
1. **Validates version** (tag must match version.txt)
2. Runs GoReleaser
3. Builds & pushes Docker image to ECR
4. Updates K8s manifests
5. Syncs ArgoCD to production
6. Creates backport PR to develop (auto-merge enabled)
7. Notifies team

#### `hotfix.yaml`
**Trigger:** Manual (`workflow_dispatch`)
**Purpose:** Critical production fixes

**What it does:**
- Similar to prepare-release but from `main` branch
- Auto-increments patch version
- Marks PR as `priority:critical`
- Expedited timeline (1-2h target)

#### `release-staging.yaml`
**Trigger:** Automatic (push to develop or release/* branches)
**Purpose:** Staging environment deployment

**Features:**
- Blocks backport deployments (no duplicate staging deploys)
- Generates staging-specific version tags
- Deploys to staging cluster via ArgoCD

---

### CI/CD Workflows

#### `ci.yaml`
**Trigger:** Push to any branch, PRs
**Purpose:** Continuous Integration

**Runs:**
- Unit tests
- Linters (gofmt, go vet, golint, staticcheck, goimports)
- Build validation
- Code coverage

#### `auto-merge.yaml`
**Trigger:** PR events, reviews, check completions
**Purpose:** Automatic PR merging

**Auto-merges:**
- ✅ Dependabot PRs (minor/patch)
- ✅ Backport PRs (when checks pass)
- ❌ Release PRs (require manual approval!)

---

### Utility Workflows

#### `notify-release-cutoff.yaml`
**Trigger:** Automatic (when release/* or hotfix/* branch created)
**Purpose:** Team notification

Sends Telegram/n8n notification announcing feature cutoff.

#### `bot-code-reviewer.yaml`
Automated code review bot

#### `generate-description.yaml`
Auto-generates PR descriptions

---

## 🏷️ Labels Configuration

### Labels File: `.github/labels.yaml`

Labels are **automatically created** by workflows when needed. No manual setup required!

### Core Labels
- **release** - Release PRs
- **hotfix** - Critical fixes
- **backport** - Backport PRs
- **automated** - Workflow-created
- **priority:critical** - Urgent
- **auto-merge** - Auto-merge when approved

**Sync labels manually** (optional):
```bash
gh label sync -f .github/labels.yaml
```

---

## 📋 Common Tasks

### Creating a New Release

1. **Go to Actions** → "Prepare Release"
2. **Click "Run workflow"**
3. **Choose:**
   - Source branch: `develop` (default)
   - Version type: `minor` (default) or specific version
4. **Click "Run"**

**What happens:**
- ✅ Release branch created
- ✅ Tag created: `v2.X.X`
- ✅ PR to main created
- ✅ Staging deployed
- ⚠️ **FEATURE CUTOFF announced**
- ⏱️ Expected completion: 8 hours

5. **QA validates in staging** (2-4h)
6. **Approve & merge PR manually**
7. **Production deployment happens automatically**

### Creating a Hotfix

1. **Go to Actions** → "Hotfix Workflow"
2. **Describe the critical issue**
3. **Let auto-increment handle version** (or specify)
4. **Review & merge ASAP** (1-2h target)

### Feature Cutoff FAQ

**Q: I just merged to develop. Will it be in the current release?**
A: Check if a release branch exists:
- **No release branch** → YES, your feature will be included
- **Release branch exists** → NO, it goes to NEXT release

**Q: How do I know if feature cutoff is active?**
A: Check Telegram notifications or look for open release PRs to main

**Q: I NEED my feature in this release!**
A: Options:
1. Cherry-pick to release branch (risky, manual)
2. Wait for next release (we do quick releases!)

### Timeline Example

```
T+0h:    Release created (snapshot of develop)
         ↓ Feature cutoff announced
         ↓ Staging deployed

T+0.5h:  QA starts testing

T+4h:    QA completes, approves PR

T+5h:    PR merged → Production deployment triggered

T+6h:    Production deployment complete

T+6.5h:  Backport PR auto-merged to develop

T+8h:    Release cycle complete ✅
```

---

## 🔧 Troubleshooting

### "Label 'release' not found" Error

Workflows auto-create labels. If you see this error:
1. Labels will be created on next run
2. Or manually sync: `gh label sync -f .github/labels.yaml`

### Version Mismatch Error

If validation fails ("Tag version doesn't match version.txt"):
1. Check `version.txt` in release branch
2. Check tag name
3. They must match exactly

### Backport PR Has Conflicts

Manual resolution required:
1. Checkout backport branch
2. Resolve conflicts
3. Push
4. Auto-merge will proceed

### Staging Deployment Not Triggered

- Check workflow runs in Actions tab
- Verify release/* branch was pushed
- Check if backport (those are blocked)

---

## 🔐 Required Secrets

Configure in GitHub Settings → Secrets:

### GitHub
- `GH_TOKEN` - GitHub token (repo + workflow permissions)

### AWS/ECR
- `AWS_ACCESS_KEY_ID`
- `AWS_SECRET_ACCESS_KEY`
- `AWS_REGION`

### ArgoCD
- `ARGOCD_SERVER`
- `ARGOCD_TOKEN`

### Notifications
- `N8N_WEBHOOK_URL`
- `N8N_API_TOKEN`
- `CHAT_ID` (Telegram)
- `THREAD_ID` (Telegram)

### Docker Hub (optional)
- `DOCKERHUB_USERNAME`
- `DOCKERHUB_TOKEN`

---

## 📚 Additional Documentation

- **Detailed Process:** `.github/RELEASE_PROCESS.md`
- **Project Docs:** `CLAUDE.md` (root directory)

---

## 🎯 Best Practices

1. **Always use conventional commits** - See `.cursor/rules/commit-flow.mdc`
2. **Test in staging before approving** - Don't skip QA!
3. **Monitor Telegram notifications** - Stay informed
4. **Respect feature cutoff** - Don't force features into releases
5. **Keep releases small and frequent** - Better than big releases!

---

## 🚨 Emergency Procedures

### Rollback Production

If deployment fails or introduces issues:

```bash
# 1. Find last good tag
git tag -l 'v*' --sort=-v:refname | head -5

# 2. Trigger release.yaml manually with good tag
# Go to Actions → "Release with GoReleaser" → "Run workflow"
# Enter the good tag (e.g., v2.0.9)
```

### Skip Feature Cutoff (Emergency Only)

If absolutely critical feature must go into existing release:

```bash
# Cherry-pick commit to release branch
git checkout release/vX.X.X
git cherry-pick <commit-hash>
git push
```

⚠️ This breaks the quick release model. Use sparingly!

---

**Questions?** Check Telegram or create an issue.

*This documentation reflects the Quick Release model (develop = next release).*
