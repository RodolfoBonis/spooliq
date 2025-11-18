# SpoolIQ Release Process

## Quick Release Model (2-8h Cycle)

**Model:** develop = next release
**Cycle Time:** 2-8 hours from release creation to production
**Philosophy:** Small, frequent releases with rapid QA validation

---

## Table of Contents

1. [Prerequisites](#prerequisites)
2. [Regular Release (from develop)](#regular-release-from-develop)
3. [Hotfix Release (from main)](#hotfix-release-from-main)
4. [Understanding Feature Cutoff](#understanding-feature-cutoff)
5. [QA Validation Checklist](#qa-validation-checklist)
6. [Post-Release](#post-release)
7. [Rollback Procedure](#rollback-procedure)
8. [Troubleshooting](#troubleshooting)

---

## Prerequisites

Before creating any release:

- [ ] All features merged to `develop` are tested
- [ ] CI is green on `develop` branch
- [ ] No known critical bugs in `develop`
- [ ] Team is aware a release is planned
- [ ] Staging environment is healthy

---

## Regular Release (from develop)

### Step 1: Prepare Release (5 minutes)

1. **Navigate to GitHub Actions**
   - Go to repository → Actions tab
   - Select "Prepare Release" workflow

2. **Configure Release**
   - Click "Run workflow"
   - **Source branch:** `develop` (default)
   - **Version:** Leave empty for auto-increment OR specify (e.g., `2.1.0`)
   - **Increment type:** `minor` (default), `major`, or `patch`
   - Click "Run workflow" button

3. **What Happens Automatically**
   ```
   ✅ Release branch created: release/vX.X.X
   ✅ version.txt bumped to X.X.X
   ✅ CHANGELOG.md updated with commits since last release
   ✅ Git tag created: vX.X.X
   ✅ Branch and tag pushed to GitHub
   ✅ PR created to main branch
   ✅ Staging deployment triggered
   ✅ Team notified via Telegram
   ⚠️  FEATURE CUTOFF announced
   ```

4. **Verify Success**
   - Check Actions tab for successful completion
   - Find PR in Pull Requests tab
   - Verify tag exists: `git tag -l | grep vX.X.X`
   - Check Telegram notification received

### Step 2: Feature Cutoff (Immediate)

**From this moment:**
- ✅ Features in develop NOW → Going to production in this release
- ❌ Features merged to develop LATER → Going to NEXT release

**Team Communication:**
```
🚀 Release v2.1.0 is being prepared!

⚠️ FEATURE CUTOFF IS NOW ACTIVE

✅ Features IN release: Everything in develop as of [timestamp]
❌ Features NOT in release: Anything merged after [timestamp]

Next features will go to v2.2.0

📋 PR: [link]
🧪 Staging: [link]
⏱️ Target prod: 8h from now
```

### Step 3: QA Validation (2-4 hours)

**QA Team Responsibilities:**

1. **Access Staging Environment**
   - URL: Check deployment logs or team channel
   - Version: Verify shows `vX.X.X-rc.TIMESTAMP`

2. **Test New Features**
   - Check CHANGELOG in PR for list of changes
   - Test each new feature mentioned
   - Verify existing features still work

3. **Regression Testing**
   - Core user flows (auth, main features)
   - API endpoints (if applicable)
   - Mobile responsiveness
   - Performance check

4. **Validation Checklist** (see section below)

5. **Approval**
   - If all checks pass: Approve the PR on GitHub
   - If issues found: Comment on PR, request fixes

### Step 4: Merge to Main (Manual - 1 minute)

**⚠️ Important:** This is a manual step to ensure conscious deployment!

1. **Final Checks**
   - [ ] QA approved the PR
   - [ ] All CI checks passed (green checkmarks)
   - [ ] No merge conflicts
   - [ ] Staging is stable

2. **Merge PR**
   - Go to the release PR
   - Click "Merge pull request"
   - **Use "Merge commit"** (not squash or rebase)
   - Click "Confirm merge"

3. **What Happens Automatically**
   ```
   ✅ post-merge-release.yaml detects merge
   ✅ Validates tag exists
   ✅ Triggers release.yaml workflow
   ✅ release.yaml runs:
      - Validates version consistency
      - Runs GoReleaser
      - Builds Docker image
      - Pushes to ECR
      - Updates K8s manifests
      - Syncs ArgoCD to production
      - Creates backport PR to develop
   ✅ Backport PR auto-merges (when CI passes)
   ✅ Team notified of deployment
   ```

### Step 5: Monitor Deployment (15-30 minutes)

1. **Watch Workflow Execution**
   - Go to Actions tab
   - Watch "Release with GoReleaser" workflow
   - All steps should complete with green checkmarks

2. **Verify Production Deployment**
   - Check ArgoCD dashboard
   - Verify pod version matches release
   - Check application logs for errors
   - Test critical endpoints

3. **Monitor Metrics**
   - Error rates (should not spike)
   - Response times (should be normal)
   - User traffic (watch for anomalies)

4. **Telegram Notifications**
   - Deployment success message should arrive
   - Contains: version, Docker image, build time, release URL

### Step 6: Post-Deployment Validation (10 minutes)

1. **Smoke Tests in Production**
   - [ ] Health check endpoint responds
   - [ ] User login works
   - [ ] Core features function
   - [ ] New features are live

2. **Backport Verification**
   - Check that backport PR to develop was created
   - Verify it auto-merged (or merge manually if conflicts)
   - Pull develop: `git checkout develop && git pull`
   - Verify version.txt matches release version

3. **Documentation**
   - GitHub Release created automatically
   - CHANGELOG reflected in GitHub Release
   - Verify accuracy

### Step 7: Team Communication

Post in team channel:
```
✅ Release v2.1.0 DEPLOYED TO PRODUCTION

🚀 Deployment successful
⏱️ Time: X hours (target was 8h)
📦 Features included: [link to CHANGELOG]
🔗 Release notes: [GitHub Release link]

Thanks to QA team for quick validation! 🎉
```

---

## Hotfix Release (from main)

### When to Use Hotfix

- 🚨 Critical production bug affecting users
- 🚨 Security vulnerability discovered
- 🚨 Data integrity issue
- 🚨 Service outage or degraded performance

**NOT for:**
- Regular features
- Non-critical bugs (wait for regular release)
- Performance optimizations (unless critical)

### Hotfix Process (1-2 hours target)

#### Step 1: Create Hotfix (5 minutes)

1. **Navigate to Actions**
   - Go to "Hotfix Workflow"

2. **Configure Hotfix**
   - Click "Run workflow"
   - **Description:** Clearly describe the critical issue
   - **Version:** Leave empty for auto-patch increment
   - Click "Run"

3. **Automatic Actions**
   ```
   ✅ Hotfix branch created from main: hotfix/vX.X.Y
   ✅ version.txt patch bumped
   ✅ Tag created: vX.X.Y
   ✅ PR created to main (labeled priority:critical)
   ✅ Team notified with CRITICAL alert
   ```

#### Step 2: Implement Fix (30 minutes - 1 hour)

1. **Checkout Hotfix Branch**
   ```bash
   git fetch
   git checkout hotfix/vX.X.Y
   ```

2. **Implement Fix**
   - Make minimal changes (only fix the issue)
   - Avoid refactoring or unrelated changes
   - Add/update tests if needed

3. **Push Changes**
   ```bash
   git add .
   git commit -m "fix: [description of fix]"
   git push
   ```

4. **Verify CI Passes**

#### Step 3: Expedited Review (15-30 minutes)

**Reviewer Responsibilities:**

- [ ] Fix addresses the critical issue
- [ ] No unintended side effects
- [ ] Tests included/updated
- [ ] Code is minimal and focused

**Approve Immediately** if checks pass!

#### Step 4: Deploy (Same as regular release)

1. **Merge PR to main**
2. **Automatic production deployment**
3. **Monitor closely** (hotfixes are higher risk!)
4. **Verify issue is resolved**

#### Step 5: Backport to Develop

**Automatic:** Backport PR created and auto-merged

**Manual verification:**
```bash
git checkout develop
git pull
# Verify hotfix is present
```

---

## Understanding Feature Cutoff

### What is Feature Cutoff?

**Feature cutoff** is the moment when a release branch is created. It defines which features go into the current release vs. the next release.

### Timeline Example

```
Monday 9:00 AM    Dev1 merges feature A to develop
Monday 10:00 AM   Dev2 merges feature B to develop

Monday 11:00 AM   🚀 Release v2.1.0 created
                  ⚠️  FEATURE CUTOFF ACTIVE

Monday 12:00 PM   Dev3 merges feature C to develop  ← Goes to v2.2.0!

Monday 3:00 PM    QA validates staging (A + B, not C)
Monday 4:00 PM    PR approved and merged
Monday 5:00 PM    Production deployed (A + B, not C)
Monday 6:00 PM    Backport merged, develop ready for next

Tuesday 9:00 AM   Next release prepared with feature C
```

### Why Feature Cutoff?

1. **QA Stability:** QA tests a fixed set of features
2. **No Moving Target:** What QA tests is what gets deployed
3. **Clear Communication:** Everyone knows what's in the release
4. **Fast Cycles:** Don't wait for "one more feature"

### How to Handle It

**As a Developer:**
- Check for open release PRs before expecting feature in production
- If cutoff passed, wait for next release (usually 1-2 days max)
- Don't try to force features into releases

**As a PM:**
- Communicate release schedule in advance
- Plan feature completion before cutoff
- Accept that some features wait for next release

**Emergency Exception:**
If absolutely critical:
```bash
git checkout release/vX.X.X
git cherry-pick <commit-hash>
git push
```
⚠️ Requires re-QA of the modified release!

---

## QA Validation Checklist

### Pre-Deployment Checklist

Use this checklist when validating in staging:

#### Functionality
- [ ] All new features work as expected
- [ ] No regressions in existing features
- [ ] Error handling works correctly
- [ ] Edge cases handled

#### Performance
- [ ] Page load times acceptable
- [ ] API response times normal
- [ ] No memory leaks observed
- [ ] Database queries performant

#### UI/UX
- [ ] UI renders correctly (desktop)
- [ ] UI renders correctly (mobile)
- [ ] No console errors in browser
- [ ] Accessibility maintained

#### Integration
- [ ] External APIs functional
- [ ] Payment processing works (if applicable)
- [ ] Email notifications sent
- [ ] File uploads/downloads work

#### Security
- [ ] Authentication works
- [ ] Authorization rules enforced
- [ ] No sensitive data exposed in logs
- [ ] CORS configured correctly

#### Data
- [ ] Database migrations successful
- [ ] Data integrity maintained
- [ ] Backups tested (if schema changes)
- [ ] Rollback plan exists

### Post-Deployment Checklist

After production deployment:

- [ ] Health check endpoint returns 200
- [ ] Application logs show no errors
- [ ] Metrics dashboards normal
- [ ] User login/signup works
- [ ] Critical user flows function
- [ ] External integrations working
- [ ] Error tracking shows no spikes

---

## Post-Release

### Immediate (Within 1 hour)

1. **Monitor Production**
   - Watch error rates
   - Monitor user feedback channels
   - Check Telegram alerts

2. **Verify Backport**
   - Ensure develop has release changes
   - Update local develop branch

3. **Update Team**
   - Post deployment success message
   - Note any issues encountered

### Same Day

1. **Review Metrics**
   - Compare to pre-release baseline
   - Note any anomalies

2. **User Feedback**
   - Monitor support channels
   - Address urgent issues

3. **Documentation**
   - Verify GitHub Release accuracy
   - Update user-facing docs if needed

### Next Day

1. **Retrospective** (if issues occurred)
   - What went wrong?
   - How to prevent in future?
   - Update process if needed

2. **Plan Next Release**
   - Review features in develop
   - Estimate next release timing

---

## Rollback Procedure

### When to Rollback

- Critical bug introduced
- Performance degradation severe
- Data corruption detected
- Service unavailable

### Quick Rollback (15 minutes)

1. **Identify Last Good Version**
   ```bash
   git tag -l 'v*' --sort=-v:refname | head -10
   ```

2. **Trigger Manual Deployment**
   - Go to Actions → "Release with GoReleaser"
   - Click "Run workflow"
   - **Tag:** Enter last good tag (e.g., `v2.0.9`)
   - **Version:** Enter version without v (e.g., `2.0.9`)
   - Click "Run"

3. **Monitor Deployment**
   - Watch workflow complete
   - Verify ArgoCD synced
   - Check application healthy

4. **Notify Team**
   ```
   🚨 ROLLBACK EXECUTED

   Rolled back from v2.1.0 to v2.0.9
   Reason: [description]
   Status: [current status]

   Next steps: [investigation plan]
   ```

5. **Investigate Issue**
   - Don't merge new releases until root cause found
   - Fix issue in develop
   - Test thoroughly before next release attempt

### Post-Rollback

1. **Update version.txt in develop**
   ```bash
   git checkout develop
   echo "2.0.9" > version.txt
   git add version.txt
   git commit -m "chore: sync version after rollback"
   git push
   ```

2. **Document Incident**
   - What failed
   - Why it failed
   - How it was fixed
   - How to prevent

---

## Troubleshooting

### Release Branch Creation Failed

**Symptom:** prepare-release.yaml fails

**Common Causes:**
- `version.txt` missing or malformed
- Git permissions issue
- Conflicting tags

**Fix:**
1. Check `version.txt` format: `X.Y.Z` (no 'v' prefix, no whitespace)
2. Verify `GH_TOKEN` has correct permissions
3. Check for conflicting tags: `git tag -l | grep vX.X.X`

### Tag Already Exists

**Symptom:** "tag vX.X.X already exists"

**Fix:**
1. Delete tag if incorrect:
   ```bash
   git tag -d vX.X.X
   git push origin :refs/tags/vX.X.X
   ```
2. Or increment version number if tag is correct

### Staging Deployment Not Triggered

**Symptom:** Staging not deployed after release branch created

**Check:**
1. Workflow run status in Actions tab
2. Branch naming: must be `release/vX.X.X`
3. Not a backport (those are blocked)

**Fix:**
- Manually trigger: Actions → "Release Staging" → "Run workflow"

### Version Validation Failed

**Symptom:** "Tag version doesn't match version.txt"

**Cause:** Tag and version.txt out of sync

**Fix:**
1. Check tag: `git show vX.X.X:version.txt`
2. Tag should contain version matching its name
3. If mismatch, delete tag and recreate release

### Backport PR Has Conflicts

**Symptom:** Backport PR shows merge conflicts

**Fix:**
1. Checkout backport branch:
   ```bash
   git fetch
   git checkout backport/vX.X.X-to-develop
   ```

2. Merge main and resolve conflicts:
   ```bash
   git merge origin/main
   # Resolve conflicts
   git add .
   git commit
   git push
   ```

3. Auto-merge will proceed once CI passes

### Production Deployment Stuck

**Symptom:** ArgoCD sync not completing

**Check:**
1. ArgoCD dashboard for errors
2. K8s pod status
3. Resource limits (CPU/memory)

**Fix:**
1. Check ArgoCD logs
2. Manual sync if needed
3. Rollback if unrecoverable

### Backport Not Auto-Merged

**Symptom:** Backport PR still open after CI passes

**Check:**
1. PR has `auto-merge` label
2. CI actually passed (all green)
3. No merge conflicts

**Fix:**
- Manually approve and merge if needed

---

## Tips for Success

1. **Release Early, Release Often**
   - Don't batch features for weeks
   - Small releases = less risk

2. **Respect the Process**
   - Feature cutoff is firm
   - QA validation is mandatory
   - No shortcuts!

3. **Communicate Proactively**
   - Announce releases in advance
   - Keep team informed of progress
   - Share delays immediately

4. **Monitor Everything**
   - Staging before production
   - Production after deployment
   - Metrics always

5. **Document Issues**
   - Every problem is a learning opportunity
   - Update this document!
   - Share knowledge

---

## Questions or Issues?

- Check Telegram channel
- Review this document
- Ask tech lead
- Create GitHub issue

---

**Last Updated:** 2025-01-18
**Process Version:** 2.0 (Quick Release Model)
