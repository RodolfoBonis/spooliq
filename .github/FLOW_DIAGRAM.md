# 🔄 SpoolIQ Release Flow - Complete Visual Guide

## 📊 High-Level Architecture

```
┌─────────────────────────────────────────────────────────────────────────┐
│                        SPOOLIQ RELEASE SYSTEM                            │
│                     (Quick Release Model - 2-8h)                         │
└─────────────────────────────────────────────────────────────────────────┘

                         develop = NEXT RELEASE
                                   │
                                   │ Feature merges continuously
                                   ▼
                    ┌──────────────────────────┐
                    │   develop branch         │
                    │   (always deployable)    │
                    └──────────────────────────┘
                                   │
                                   │ Manual: prepare-release.yaml
                                   ▼
                    ┌──────────────────────────┐
                    │  ⚠️  FEATURE CUTOFF!     │
                    │  Snapshot taken NOW      │
                    └──────────────────────────┘
                                   │
                    ┌──────────────┴───────────────┐
                    │                              │
                    ▼                              ▼
         release/vX.X.X branch          Tag vX.X.X created
         (frozen snapshot)              (for validation)
                    │                              │
                    │                              │
                    └──────────────┬───────────────┘
                                   │
                                   ▼
                    ┌──────────────────────────┐
                    │   PR to main created     │
                    │   (requires manual       │
                    │    approval)             │
                    └──────────────────────────┘
                                   │
                                   │ Manual: Review, Approve & Merge PR (2-4h)
                                   ▼
                    ┌──────────────────────────┐
                    │   main branch updated    │
                    └──────────────────────────┘
                                   │
                                   │ Auto: post-merge-release.yaml
                                   ▼
                    ┌──────────────────────────┐
                    │   Triggers release.yaml  │
                    │   with validation        │
                    └──────────────────────────┘
                                   │
                                   │ Auto: release.yaml
                                   ▼
                    ┌──────────────────────────┐
                    │   Production Deployment  │
                    │   (GoReleaser + ArgoCD)  │
                    └──────────────────────────┘
                                   │
                                   │ Auto: backport created
                                   ▼
                    ┌──────────────────────────┐
                    │   develop updated        │
                    │   (auto-merge)           │
                    └──────────────────────────┘
                                   │
                                   ▼
                              🎉 COMPLETE!
```

---

## 🚀 REGULAR RELEASE FLOW (Detailed)

### Timeline: T+0h to T+8h

```
═══════════════════════════════════════════════════════════════════════════
T+0h: RELEASE PREPARATION (5 minutes)
═══════════════════════════════════════════════════════════════════════════

Developer/PM triggers prepare-release.yaml

┌─────────────────────────────────────────────────────────────────┐
│  prepare-release.yaml (Manual Trigger)                          │
│                                                                  │
│  Inputs:                                                         │
│  • source_branch: develop (default)                             │
│  • version: "" (auto-increment) OR "2.2.0" (specific)           │
│  • increment_type: minor (default)                              │
└─────────────────────────────────────────────────────────────────┘
                             │
                             │ Step 1: Checkout develop
                             ▼
                ┌────────────────────────────┐
                │  git checkout develop      │
                │  git pull origin develop   │
                └────────────────────────────┘
                             │
                             │ Step 2: Determine version
                             ▼
                ┌────────────────────────────┐
                │  Read version.txt: 2.1.1   │
                │  Increment: minor          │
                │  New version: 2.2.0        │
                └────────────────────────────┘
                             │
                             │ Step 3: Create release branch
                             ▼
                ┌────────────────────────────┐
                │  git checkout -b           │
                │    release/v2.2.0          │
                └────────────────────────────┘
                             │
                             │ Step 4: Update version.txt
                             ▼
                ┌────────────────────────────┐
                │  echo "2.2.0" >            │
                │    version.txt             │
                │  git commit -m             │
                │    "chore: bump v2.2.0"    │
                └────────────────────────────┘
                             │
                             │ Step 5: Generate CHANGELOG
                             ▼
                ┌────────────────────────────┐
                │  Get commits since v2.1.1  │
                │  Update CHANGELOG.md       │
                │  git commit -m             │
                │    "docs: update CHANGELOG"│
                └────────────────────────────┘
                             │
                             │ Step 6: Create & push tag
                             ▼
                ┌────────────────────────────┐
                │  ⭐ KEY CHANGE!            │
                │  git tag -a v2.2.0         │
                │  git push origin           │
                │    release/v2.2.0          │
                │  git push origin v2.2.0    │
                │                            │
                │  Tag exists BEFORE PR!     │
                └────────────────────────────┘
                             │
                             │ Step 7: Ensure labels exist
                             ▼
                ┌────────────────────────────┐
                │  gh label create           │
                │    --force:                │
                │    • release               │
                │    • automated             │
                │    • backport              │
                └────────────────────────────┘
                             │
                             │ Step 8: Create PR to main
                             ▼
┌────────────────────────────────────────────────────────────────────┐
│  PR Created: "🚀 Release v2.2.0"                                   │
│                                                                     │
│  ✅ Tag v2.2.0 created                                             │
│  ✅ Snapshot of develop at: 2025-01-18 10:00:00 UTC               │
│                                                                     │
│  ⚠️  FEATURE CUTOFF ACTIVE                                         │
│  Features merged to develop AFTER this snapshot → v2.3.0          │
│                                                                     │
│  Timeline:                                                          │
│  - [x] Snapshot created & tag pushed                               │
│  - [ ] Release PR review (2-4h)                                    │
│  - [ ] PR approval & merge                                         │
│  - [ ] Production deployment (automatic)                           │
│  - [ ] Backport to develop (automatic)                             │
│                                                                     │
│  Expected completion: 2025-01-18 18:00:00 UTC (8h)                │
└────────────────────────────────────────────────────────────────────┘
                             │
                             │ Step 9: Notify team
                             ▼
┌────────────────────────────────────────────────────────────────────┐
│  📱 Telegram Notification                                          │
│                                                                     │
│  🚀 Release v2.2.0 prepared!                                       │
│                                                                     │
│  ⚠️ FEATURE CUTOFF: New features → next release                   │
│  ⏱️ Target: 8h                                                     │
│  📋 PR: https://github.com/.../pull/123                           │
└────────────────────────────────────────────────────────────────────┘

───────────────────────────────────────────────────────────────────────────

MEANWHILE: notify-release-cutoff.yaml also triggered

┌────────────────────────────────────────────────────────────────────┐
│  notify-release-cutoff.yaml (Automatic on branch create)          │
│                                                                     │
│  Sends additional Telegram message:                                │
│                                                                     │
│  📸 Snapshot of develop taken                                      │
│  ⚠️ FEATURE CUTOFF ACTIVE                                          │
│                                                                     │
│  ✅ Features IN this release: Everything in develop NOW           │
│  ❌ Features NOT in this release: Anything merged after this      │
│                                                                     │
│  💡 Tip: Check the release PR for full changelog                  │
└────────────────────────────────────────────────────────────────────┘

═══════════════════════════════════════════════════════════════════════════
T+0h to T+4h: RELEASE PR REVIEW (2-4h)
═══════════════════════════════════════════════════════════════════════════

Reviewer checks the release PR

┌────────────────────────────────────────────────────────────────────┐
│  Release Validation Checklist                                      │
│                                                                     │
│  ☐ CHANGELOG reviewed                                              │
│  ☐ All CI checks passing                                           │
│  ☐ Breaking changes documented (if any)                            │
│  ☐ No sensitive data included                                      │
└────────────────────────────────────────────────────────────────────┘
                             │
                             │ After review
                             ▼
                ┌────────────────────────────┐
                │  ✅ Reviewer approves PR   │
                │     on GitHub              │
                └────────────────────────────┘

═══════════════════════════════════════════════════════════════════════════
T+4h: MANUAL MERGE (Conscious approval - 1 minute)
═══════════════════════════════════════════════════════════════════════════

Developer/PM merges PR manually (NO auto-merge!)

┌────────────────────────────────────────────────────────────────────┐
│  GitHub PR: "🚀 Release v2.2.0"                                    │
│                                                                     │
│  Checks:                                                            │
│  ✅ PR approved                                                    │
│  ✅ CI passed                                                      │
│  ✅ No merge conflicts                                             │
│                                                                     │
│  ⚠️  This is a MANUAL step!                                        │
│  Click "Merge pull request"                                        │
│  → "Merge commit" (not squash!)                                    │
│  → "Confirm merge"                                                 │
└────────────────────────────────────────────────────────────────────┘
                             │
                             │ PR merged!
                             ▼
                ┌────────────────────────────┐
                │  main branch updated       │
                │  Merge commit: abc123      │
                └────────────────────────────┘

═══════════════════════════════════════════════════════════════════════════
T+4h: POST-MERGE ORCHESTRATION (Automatic - 1 minute)
═══════════════════════════════════════════════════════════════════════════

post-merge-release.yaml automatically detects merge

┌─────────────────────────────────────────────────────────────────┐
│  post-merge-release.yaml (Auto-triggered on PR merge)          │
│                                                                  │
│  Trigger:                                                        │
│  • Event: pull_request (closed)                                 │
│  • Branch: main                                                  │
│  • Head ref: release/* OR hotfix/*                              │
│  • Merged: true                                                  │
└─────────────────────────────────────────────────────────────────┘
                             │
                             │ Step 1: Extract version from branch
                             ▼
                ┌────────────────────────────┐
                │  Branch: release/v2.2.0    │
                │  Extract: v2.2.0           │
                │  Version: 2.2.0            │
                └────────────────────────────┘
                             │
                             │ Step 2: Verify tag exists
                             ▼
                ┌────────────────────────────┐
                │  ✅ CHECK:                 │
                │  git rev-parse v2.2.0      │
                │                            │
                │  Tag exists? YES!          │
                │  (created by               │
                │   prepare-release)         │
                └────────────────────────────┘
                             │
                             │ Step 3: Ensure labels exist
                             ▼
                ┌────────────────────────────┐
                │  Create if missing:        │
                │  • release                 │
                │  • automated               │
                │  • backport                │
                │  • hotfix                  │
                │  • priority:critical       │
                └────────────────────────────┘
                             │
                             │ Step 4: Trigger release.yaml
                             ▼
┌────────────────────────────────────────────────────────────────────┐
│  gh workflow run release.yaml                                      │
│    --ref v2.2.0                                                    │
│    -f tag=v2.2.0                                                   │
│    -f version=2.2.0                                                │
│                                                                     │
│  🔥 KEY: Single trigger point!                                     │
│  No more conflicting triggers!                                     │
└────────────────────────────────────────────────────────────────────┘
                             │
                             │ Step 5: Notify team
                             ▼
┌────────────────────────────────────────────────────────────────────┐
│  📱 Telegram Notification                                          │
│                                                                     │
│  🚀 Production deployment triggered for v2.2.0                     │
│                                                                     │
│  PR #123 merged by @developer                                      │
│  Watch deployment: [Actions link]                                  │
└────────────────────────────────────────────────────────────────────┘

═══════════════════════════════════════════════════════════════════════════
T+4h to T+5h: PRODUCTION DEPLOYMENT (Automatic - 15-30 min)
═══════════════════════════════════════════════════════════════════════════

release.yaml executes (workflow_dispatch trigger ONLY)

┌─────────────────────────────────────────────────────────────────┐
│  release.yaml                                                    │
│                                                                  │
│  ⭐ NEW: Only 1 trigger (workflow_dispatch)                     │
│  ⭐ NEW: Validation job runs FIRST                              │
└─────────────────────────────────────────────────────────────────┘
                             │
                             │ JOB 1: VALIDATE
                             ▼
┌────────────────────────────────────────────────────────────────────┐
│  Validation Job (CRITICAL!)                                        │
│                                                                     │
│  Input tag: v2.2.0                                                 │
│  Input version: 2.2.0                                              │
│                                                                     │
│  ✅ CHECK 1: Tag format valid (v1.2.3)                            │
│     v2.2.0 matches pattern ✓                                      │
│                                                                     │
│  ✅ CHECK 2: Input version matches tag                            │
│     2.2.0 == 2.2.0 ✓                                              │
│                                                                     │
│  ✅ CHECK 3: version.txt matches tag                              │
│     Checkout v2.2.0                                                │
│     Read version.txt: 2.2.0                                        │
│     2.2.0 == 2.2.0 ✓                                              │
│                                                                     │
│  🎉 ALL VALIDATIONS PASSED!                                        │
│  Safe to proceed with deployment.                                  │
└────────────────────────────────────────────────────────────────────┘
                             │
                             │ If validation fails, STOP!
                             │ If validation passes, continue
                             │
                             │ JOB 2: RELEASE (needs: validate)
                             ▼
┌────────────────────────────────────────────────────────────────────┐
│  Release Job (Production Deployment)                               │
│                                                                     │
│  Step 1: Checkout tag v2.2.0                                       │
│  Step 2: Setup Go 1.23                                             │
│  Step 3: Configure AWS/ECR                                         │
│  Step 4: Login to ECR                                              │
│  Step 5: Setup Docker Buildx                                       │
│  Step 6: Run GoReleaser                                            │
│         ├─> Build binaries (linux, darwin, windows)               │
│         ├─> Create archives                                        │
│         ├─> Build Docker image                                     │
│         ├─> Push to ECR: spooliq:2.2.0                            │
│         └─> Create GitHub Release                                  │
│  Step 7: Checkout K8s manifests repo                              │
│  Step 8: Update production values                                  │
│         └─> values-prod.yaml: image.tag = "2.2.0"                 │
│  Step 9: Commit & push manifest changes                           │
│  Step 10: Sync ArgoCD production                                   │
│          └─> argocd app sync spooliq-app-prod                     │
│  Step 11: Calculate build duration                                 │
│  Step 12: Notify success                                           │
└────────────────────────────────────────────────────────────────────┘
                             │
                             │ On success
                             ▼
┌────────────────────────────────────────────────────────────────────┐
│  📱 Telegram Notification                                          │
│                                                                     │
│  ✅ Production deployment SUCCESS!                                 │
│                                                                     │
│  Version: v2.2.0                                                   │
│  Docker: xxx.dkr.ecr.../spooliq:2.2.0                             │
│  Build time: 5m 23s                                                │
│  Release: https://github.com/.../releases/tag/v2.2.0              │
│                                                                     │
│  ArgoCD sync: ✅                                                   │
│  Status: LIVE IN PRODUCTION 🎉                                     │
└────────────────────────────────────────────────────────────────────┘
                             │
                             │ JOB 3: BACKPORT (needs: validate, release)
                             ▼
┌────────────────────────────────────────────────────────────────────┐
│  Backport Job                                                      │
│                                                                     │
│  Purpose: Sync main changes back to develop                        │
│                                                                     │
│  Step 1: Checkout main                                             │
│  Step 2: Ensure labels exist (backport, automated, auto-merge)    │
│  Step 3: Create backport branch from develop                      │
│         └─> backport/v2.2.0-to-develop                            │
│  Step 4: Merge main into backport branch                          │
│         └─> git merge origin/main --no-ff                         │
│  Step 5: Push backport branch                                      │
│  Step 6: Create PR to develop                                      │
│         ├─> Title: "🔄 Backport v2.2.0 to develop"                │
│         ├─> Labels: backport, automated, auto-merge               │
│         └─> Auto-merge enabled!                                    │
│  Step 7: Notify backport created                                   │
└────────────────────────────────────────────────────────────────────┘
                             │
                             ▼
                ┌────────────────────────────┐
                │  Backport PR created       │
                │  Will auto-merge when      │
                │  CI passes                 │
                └────────────────────────────┘

═══════════════════════════════════════════════════════════════════════════
T+5h to T+6h: BACKPORT AUTO-MERGE (Automatic - 5-10 min)
═══════════════════════════════════════════════════════════════════════════

auto-merge.yaml detects backport PR

┌─────────────────────────────────────────────────────────────────┐
│  auto-merge.yaml                                                 │
│                                                                  │
│  Detects:                                                        │
│  • PR from backport/* branch                                    │
│  • Has "auto-merge" label                                       │
│  • CI checks passed                                             │
│  • No merge conflicts                                           │
└─────────────────────────────────────────────────────────────────┘
                             │
                             │ Auto-approves & merges
                             ▼
                ┌────────────────────────────┐
                │  ✅ PR auto-approved       │
                │  ✅ PR auto-merged         │
                │  develop updated!          │
                └────────────────────────────┘
                             │
                             ▼
┌────────────────────────────────────────────────────────────────────┐
│  develop branch now contains:                                      │
│  • version.txt = 2.2.0                                             │
│  • CHANGELOG.md updated                                            │
│  • All production code                                             │
│  • Ready for next release!                                         │
└────────────────────────────────────────────────────────────────────┘

═══════════════════════════════════════════════════════════════════════════
T+6h to T+8h: POST-DEPLOYMENT (Monitoring & Verification)
═══════════════════════════════════════════════════════════════════════════

Team monitors production

┌────────────────────────────────────────────────────────────────────┐
│  Production Health Checks:                                         │
│                                                                     │
│  ✅ Application responding (200 OK)                                │
│  ✅ No error spikes in logs                                        │
│  ✅ Response times normal                                          │
│  ✅ User traffic stable                                            │
│  ✅ ArgoCD shows green                                             │
│  ✅ Pods running: 3/3                                              │
│                                                                     │
│  Smoke tests:                                                       │
│  ✅ User login works                                               │
│  ✅ Core features functional                                       │
│  ✅ New features live                                              │
│                                                                     │
│  GitHub Release:                                                    │
│  ✅ Created automatically                                          │
│  ✅ CHANGELOG included                                             │
│  ✅ Binaries attached                                              │
└────────────────────────────────────────────────────────────────────┘

───────────────────────────────────────────────────────────────────────────

Team communication

┌────────────────────────────────────────────────────────────────────┐
│  📢 Slack/Telegram Announcement                                    │
│                                                                     │
│  ✅ Release v2.2.0 DEPLOYED TO PRODUCTION                          │
│                                                                     │
│  🚀 Deployment successful                                          │
│  ⏱️  Time: 6 hours (target was 8h) ✨                              │
│  📦 Features included: [link to CHANGELOG]                         │
│  🔗 Release notes: [GitHub Release link]                           │
│                                                                     │
│  Timeline:                                                          │
│  T+0h: Release prepared                                            │
│  T+4h: Reviewed & merged                                         │
│  T+5h: Production deployed                                         │
│  T+6h: Backport completed                                          │
│                                                                     │
│  Thanks to QA team for quick validation! 🎉                        │
│                                                                     │
│  develop branch is now ready for v2.3.0 features!                  │
└────────────────────────────────────────────────────────────────────┘

═══════════════════════════════════════════════════════════════════════════
🎉 RELEASE COMPLETE!
═══════════════════════════════════════════════════════════════════════════

Final state:

┌────────────────────────────────────────────────────────────────────┐
│  main branch:                                                      │
│  • version.txt = 2.2.0                                             │
│  • Tag v2.2.0 exists                                               │
│  • Production running v2.2.0                                       │
│                                                                     │
│  develop branch:                                                    │
│  • version.txt = 2.2.0                                             │
│  • Synced with main                                                │
│  • Ready for new features → v2.3.0                                 │
│                                                                     │
│  release/v2.2.0 branch:                                            │
│  • Can be deleted (cleanup automatic)                              │
│                                                                     │
└────────────────────────────────────────────────────────────────────┘
```

---

## 🚨 HOTFIX FLOW (Critical - 1-2h)

```
═══════════════════════════════════════════════════════════════════════════
HOTFIX: CRITICAL PRODUCTION ISSUE
═══════════════════════════════════════════════════════════════════════════

Current state:
• Production: v2.2.0
• Critical bug discovered!

┌────────────────────────────────────────────────────────────────────┐
│  Developer triggers hotfix.yaml (Manual)                           │
│                                                                     │
│  Inputs:                                                            │
│  • description: "Fix critical auth bug causing 401 errors"        │
│  • version: "" (auto-patch: 2.2.0 → 2.2.1)                        │
└────────────────────────────────────────────────────────────────────┘
                             │
                             │ Same pattern as prepare-release
                             ▼
                ┌────────────────────────────┐
                │  hotfix/v2.2.1 created     │
                │  from main (not develop!)  │
                └────────────────────────────┘
                             │
                             ▼
                ┌────────────────────────────┐
                │  version.txt: 2.2.0→2.2.1  │
                │  Tag v2.2.1 created        │
                │  PR to main created        │
                │  Label: priority:critical  │
                └────────────────────────────┘
                             │
                             ▼
┌────────────────────────────────────────────────────────────────────┐
│  📱 CRITICAL Telegram Alert                                        │
│                                                                     │
│  🚨 HOTFIX v2.2.1 created!                                         │
│                                                                     │
│  Fix critical auth bug causing 401 errors                          │
│                                                                     │
│  ⚠️  CRITICAL - Review ASAP                                        │
│  📋 PR: [link]                                                     │
└────────────────────────────────────────────────────────────────────┘
                             │
                             │ Developer implements fix
                             │ (30min - 1h)
                             ▼
                ┌────────────────────────────┐
                │  git checkout              │
                │    hotfix/v2.2.1           │
                │  [implement fix]           │
                │  git push                  │
                └────────────────────────────┘
                             │
                             │ Expedited review (15-30min)
                             ▼
                ┌────────────────────────────┐
                │  ✅ Reviewer approves      │
                │  Merge PR immediately!     │
                └────────────────────────────┘
                             │
                             │ Same automatic flow
                             ▼
                ┌────────────────────────────┐
                │  post-merge-release.yaml   │
                │  → release.yaml            │
                │  → Production deployed     │
                │  → Backport to develop     │
                └────────────────────────────┘
                             │
                             ▼
┌────────────────────────────────────────────────────────────────────┐
│  🎉 HOTFIX DEPLOYED (1-2h total)                                   │
│                                                                     │
│  Production: v2.2.1 ✅                                             │
│  Issue resolved ✅                                                 │
│  develop updated ✅                                                │
└────────────────────────────────────────────────────────────────────┘
```

---

## 🎯 KEY IMPROVEMENTS VISUALIZED

### Before vs After

```
═══════════════════════════════════════════════════════════════════════════
BEFORE (PROBLEMATIC)
═══════════════════════════════════════════════════════════════════════════

Version Increment:
prepare-release: 2.0.4 → 2.1.0 (in version.txt)
     ↓
release.yaml:   2.1.0 → 2.1.1 (auto-increment AGAIN!)
     ↓
Tag created:    v2.1.1 (doesn't match intended 2.1.0!)
     ↓
Production:     v2.1.1 (wrong version!)

Result: ❌ Version chaos, CHANGELOG wrong, confusion

───────────────────────────────────────────────────────────────────────────

Triggers:
PR merged → Triggers release.yaml via pull_request
         → CI completes → Triggers release.yaml via workflow_run
         → Tag created internally → Triggers release.yaml via push.tags
     ↓
Result: ❌ Same release runs 3 times! Race conditions!

───────────────────────────────────────────────────────────────────────────

Tag Creation:
PR created (no tag yet)
     ↓
PR reviewed
     ↓
PR merged
     ↓
release.yaml creates tag
     ↓
Result: ❌ Tag created AFTER deployment starts!
        ❌ No way to validate version beforehand

═══════════════════════════════════════════════════════════════════════════
AFTER (FIXED)
═══════════════════════════════════════════════════════════════════════════

Version Increment:
prepare-release: 2.1.1 → 2.2.0 (in version.txt)
     ↓
Tag created:    v2.2.0 (immediately, before PR)
     ↓
release.yaml:   NO INCREMENT (uses existing tag/version)
     ↓
Production:     v2.2.0 (correct!)

Result: ✅ Single source of truth, version.txt == tag == production

───────────────────────────────────────────────────────────────────────────

Triggers:
PR merged → post-merge-release.yaml (orchestrator)
         → Validates tag exists
         → Triggers release.yaml ONCE via workflow_dispatch
     ↓
Result: ✅ Single, controlled trigger point!

───────────────────────────────────────────────────────────────────────────

Tag Creation:
prepare-release creates tag v2.2.0 EARLY
     ↓
Tag pushed to GitHub
     ↓
PR created (tag already exists!)
     ↓
Reviewer can checkout tag directly
     ↓
PR merged
     ↓
release.yaml validates tag == version.txt
     ↓
Result: ✅ Tag exists before review even starts!
        ✅ Validation ensures consistency
```

---

## 📊 Component Responsibilities

```
┌──────────────────────────────────────────────────────────────────┐
│  prepare-release.yaml (The Initiator)                            │
├──────────────────────────────────────────────────────────────────┤
│  Responsibilities:                                                │
│  ✓ Determine version (auto or manual)                           │
│  ✓ Create release branch                                         │
│  ✓ Update version.txt                                            │
│  ✓ Generate CHANGELOG.md                                         │
│  ✓ CREATE TAG (early!)                                           │
│  ✓ Push branch + tag                                             │
│  ✓ Create labels                                                  │
│  ✓ Create PR to main                                             │
│  ✓ Notify team (feature cutoff!)                                │
└──────────────────────────────────────────────────────────────────┘

┌──────────────────────────────────────────────────────────────────┐
│  post-merge-release.yaml (The Orchestrator)                      │
├──────────────────────────────────────────────────────────────────┤
│  Responsibilities:                                                │
│  ✓ Detect release/hotfix PR merge                               │
│  ✓ Extract version from branch name                             │
│  ✓ VALIDATE tag exists                                           │
│  ✓ Ensure labels exist                                           │
│  ✓ Trigger release.yaml with correct parameters                 │
│  ✓ Single point of deployment trigger                           │
└──────────────────────────────────────────────────────────────────┘

┌──────────────────────────────────────────────────────────────────┐
│  release.yaml (The Deployer)                                     │
├──────────────────────────────────────────────────────────────────┤
│  Responsibilities:                                                │
│  ✓ VALIDATE version consistency (tag == version.txt)            │
│  ✓ Build with GoReleaser                                         │
│  ✓ Push Docker image to ECR                                      │
│  ✓ Update K8s manifests                                          │
│  ✓ Sync ArgoCD to production                                     │
│  ✓ Create backport PR                                            │
│  ✓ Notify success/failure                                        │
│                                                                   │
│  Does NOT:                                                        │
│  ✗ Increment version                                             │
│  ✗ Create tags                                                    │
│  ✗ Have multiple triggers                                        │
└──────────────────────────────────────────────────────────────────┘

┌──────────────────────────────────────────────────────────────────┐
│  auto-merge.yaml (The Automator)                                 │
├──────────────────────────────────────────────────────────────────┤
│  Responsibilities:                                                │
│  ✓ Auto-merge dependabot PRs (minor/patch)                      │
│  ✓ Auto-merge backport PRs (when CI passes)                     │
│                                                                   │
│  Does NOT:                                                        │
│  ✗ Auto-merge release PRs (requires manual merge!)              │
└──────────────────────────────────────────────────────────────────┘

┌──────────────────────────────────────────────────────────────────┐
│  notify-release-cutoff.yaml (The Communicator)                   │
├──────────────────────────────────────────────────────────────────┤
│  Responsibilities:                                                │
│  ✓ Detect release/hotfix branch creation                        │
│  ✓ Send feature cutoff notification                             │
│  ✓ Inform team of snapshot timing                               │
└──────────────────────────────────────────────────────────────────┘
```

---

## 🎓 Understanding Feature Cutoff

```
Day 1 - Monday Morning
───────────────────────────────────────────────────────────────────

9:00 AM:  Developer A merges Feature X to develop
          [Feature X is IN develop]

10:00 AM: Developer B merges Feature Y to develop
          [Feature Y is IN develop]

11:00 AM: PM triggers prepare-release.yaml

          ┌────────────────────────────────────────┐
          │  🚨 FEATURE CUTOFF HAPPENS NOW!        │
          │                                        │
          │  Snapshot taken of develop:            │
          │  • Feature X ✅ (in release)           │
          │  • Feature Y ✅ (in release)           │
          │                                        │
          │  Tag v2.3.0 created                    │
          │  Release branch: release/v2.3.0        │
          └────────────────────────────────────────┘

12:00 PM: Developer C merges Feature Z to develop
          [Feature Z is IN develop... BUT!]

          ⚠️  Feature Z will go to v2.4.0!

          Why? Because release/v2.3.0 was already created at 11:00 AM.
          The snapshot for v2.3.0 doesn't include Feature Z.

Timeline:

┌─────────────────────────────────────────────────────────────────┐
│  develop branch timeline:                                       │
│                                                                  │
│  9:00  ──┬──> Feature X merged                                 │
│          │                                                       │
│  10:00 ──┼──> Feature Y merged                                 │
│          │                                                       │
│  11:00 ──┼──> ⚡ SNAPSHOT (Feature X + Y)                      │
│          │    release/v2.3.0 created                           │
│          │    ⚠️  FEATURE CUTOFF!                              │
│          │                                                       │
│  12:00 ──┴──> Feature Z merged (goes to v2.4.0)                │
│               ↓                                                  │
│               This is on develop, but NOT in v2.3.0!           │
└─────────────────────────────────────────────────────────────────┘

Result:

v2.3.0 release contains: Feature X + Feature Y
v2.4.0 release will contain: Feature Z + future features

Why this model works:

1. QA knows exactly what they're testing (fixed scope)
2. No "one more feature" delays
3. Fast release cycles (if Z is critical, v2.4.0 comes soon!)
4. Clear communication (everyone knows the cutoff)
```

---

## 🔍 Validation Flow

```
┌──────────────────────────────────────────────────────────────────┐
│  Version Consistency Validation (NEW!)                           │
│  Runs in release.yaml BEFORE any deployment                      │
└──────────────────────────────────────────────────────────────────┘

Input: tag=v2.2.0, version=2.2.0

Step 1: Format validation
┌────────────────────────────┐
│  Is tag format valid?      │
│  Pattern: ^v[0-9]+.[0-9]+  │
│          .[0-9]+$          │
│                            │
│  v2.2.0 matches? ✅        │
└────────────────────────────┘
        │
        ▼
Step 2: Tag/version match
┌────────────────────────────┐
│  Does tag match version?   │
│  v2.2.0 → 2.2.0            │
│  2.2.0 == 2.2.0? ✅        │
└────────────────────────────┘
        │
        ▼
Step 3: File consistency
┌────────────────────────────┐
│  Checkout tag v2.2.0       │
│  Read version.txt          │
│  Content: "2.2.0"          │
│                            │
│  2.2.0 == 2.2.0? ✅        │
└────────────────────────────┘
        │
        ▼
┌────────────────────────────┐
│  ✅ ALL VALIDATIONS PASS!  │
│  Safe to deploy!           │
└────────────────────────────┘

If ANY check fails:
┌────────────────────────────┐
│  ❌ VALIDATION FAILED!     │
│  Workflow stops            │
│  Deployment prevented      │
│  Team notified of error    │
└────────────────────────────┘

This prevents:
• Deploying wrong version
• Version.txt out of sync
• Tag mismatches
• Accidental deployments
```

---

## 📱 Notification Flow

```
┌──────────────────────────────────────────────────────────────────┐
│  Telegram/n8n Notification Timeline                              │
└──────────────────────────────────────────────────────────────────┘

T+0h: Release prepared
┌────────────────────────────────────────────────────────────────────┐
│  From: prepare-release.yaml                                        │
│                                                                     │
│  🚀 Release v2.2.0 prepared!                                       │
│                                                                     │
│  ⚠️  FEATURE CUTOFF: New features → next release                  │
│  ⏱️  Target: 8h                                                    │
│  📋 PR: https://github.com/.../pull/123                           │
└────────────────────────────────────────────────────────────────────┘

T+0h: Feature cutoff announced
┌────────────────────────────────────────────────────────────────────┐
│  From: notify-release-cutoff.yaml                                  │
│                                                                     │
│  📸 Snapshot of develop taken                                      │
│  ⚠️  FEATURE CUTOFF ACTIVE                                         │
│                                                                     │
│  ✅ Features IN: Everything in develop NOW                         │
│  ❌ Features NOT in: Anything merged after this                    │
└────────────────────────────────────────────────────────────────────┘

T+4h: Production deployment triggered
┌────────────────────────────────────────────────────────────────────┐
│  From: post-merge-release.yaml                                     │
│                                                                     │
│  🚀 Production deployment triggered for v2.2.0                     │
│                                                                     │
│  PR #123 merged by @developer                                      │
│  Watch: [Actions link]                                             │
└────────────────────────────────────────────────────────────────────┘

T+5h: Deployment success
┌────────────────────────────────────────────────────────────────────┐
│  From: release.yaml                                                │
│                                                                     │
│  ✅ Production deployment SUCCESS!                                 │
│                                                                     │
│  Version: v2.2.0                                                   │
│  Docker: xxx.dkr.ecr.../spooliq:2.2.0                             │
│  Build time: 5m 23s                                                │
│  Release: https://github.com/.../releases/tag/v2.2.0              │
└────────────────────────────────────────────────────────────────────┘

T+6h: Backport created
┌────────────────────────────────────────────────────────────────────┐
│  From: release.yaml (backport job)                                 │
│                                                                     │
│  🔄 Backport PR created                                            │
│                                                                     │
│  Version: v2.2.0                                                   │
│  Target: develop                                                    │
│  Auto-merge: enabled                                               │
└────────────────────────────────────────────────────────────────────┘
```

---

## End of Flow Diagram

See `.github/README.md` for troubleshooting
See `.github/RELEASE_PROCESS.md` for detailed runbook
