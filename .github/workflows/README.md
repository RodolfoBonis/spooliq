# GitHub Actions Workflows - spooliq

This directory contains automated GitHub Actions workflows for CI/CD, code review, and documentation generation.

## 📋 Available Workflows

### 1. **CI (Continuous Integration)** - `ci.yaml`
Runs tests, linting, and vulnerability checks on every push and pull request.

**Triggers:**
- Push to `main` branch
- Pull requests (opened, synchronize, reopened)

**Jobs:**
- ✅ **Lint Go**: Runs Go code linting
- ✅ **Tests**: Runs unit tests with coverage
- ✅ **Vulnerabilities**: Checks vulnerabilities with `govulncheck`
- 📱 **Notify**: Sends Telegram notifications (optional)

### 2. **CD (Continuous Deploy)** - `cd.yaml`
Automates build, release creation, and deployment when CI is successful.

**Triggers:**
- Successful completion of CI workflow on `main` branch

**Jobs:**
- 📝 **Get Commit Messages**: Collects commit messages for release notes
- 🚀 **Build and Deploy**: Docker build, version increment, release creation
- 📱 **Notify**: Success/error notifications

### 3. **Code Review Bot** - `bot-code-reviewer.yaml`
Automated code review bot using GPT.

**Triggers:**
- Pull requests

**Features:**
- 🤖 Automated review with GPT-4o-mini
- 💬 Automatic comments on PRs
- 🔍 Code quality analysis

### 4. **Generate PR Description** - `generate-description.yaml`
Generates automatic descriptions for pull requests.

**Triggers:**
- Pull requests (opened)

**Features:**
- 📝 Generates automatic descriptions using AI
- 🔄 Analyzes code changes
- ✨ Improves PR documentation

## 🔧 Secrets Configuration

To use all workflow features, configure the following secrets in your repository:

### Required Secrets
```bash
# GitHub (already available by default)
GITHUB_TOKEN          # Automatic GitHub Actions token
```

### Optional Secrets

#### For Code Review and Description Generation
```bash
OPENAI_TOKEN          # OpenAI API token for GPT
```

#### For Telegram Notifications
```bash
TELEGRAM_BOT_TOKEN    # Telegram bot token
TELEGRAM_CHAT_ID      # Chat ID for notifications
TELEGRAM_THREAD_ID    # Thread ID (optional)
```

#### For Docker Registry (choose one)

**AWS ECR:**
```bash
AWS_ACCESS_KEY_ID     # AWS Access Key
AWS_SECRET_ACCESS_KEY # AWS Secret Key
AWS_REGION            # AWS Region (e.g., us-east-1)
```

**Docker Hub:**
```bash
DOCKER_USERNAME       # Docker Hub username
DOCKER_PASSWORD       # Docker Hub password/token
```

**GitHub Container Registry (GHCR):**
```bash
# Uses GITHUB_TOKEN automatically - no configuration needed
```

#### For Kubernetes/ArgoCD Deploy
```bash
ARGOCD_SERVER         # ArgoCD server URL
ARGOCD_TOKEN          # ArgoCD authentication token
ARGOCD_APP_NAME       # Application name in ArgoCD
K8S_MANIFEST_REPO     # K8s manifests repository (e.g., user/k8s-manifests)
K8S_DEPLOYMENT_PATH   # Deployment file path (e.g., ./apps/my-app/deployment.yaml)
```

#### For Tests and Coverage
```bash
CODECOV_TOKEN         # Codecov token (optional)
```

## 🚀 How to Setup

### 1. **Configure Secrets**
1. Go to **Settings** → **Secrets and variables** → **Actions**
2. Add necessary secrets according to your needs
3. Optional secrets can be omitted - workflows will continue working

### 2. **Customize Workflows**

#### Adjust Versioning:
```bash
# Create custom versioning script (optional)
.config/scripts/increment_version.sh
```

#### Configure Linting:
```bash
# Create custom linting script (optional)
.config/scripts/lint.sh
```

#### Python Scripts (optional):
```bash
.config/scripts/requirements.txt              # Python dependencies
.config/scripts/generate_lint_report.py       # Lint report generator
.config/scripts/generate_vulnerability_report.py  # Vulnerability report generator
.config/scripts/generate_pr_description.py    # PR description generator
```

### 3. **Docker Registry**

Workflows support multiple registries:

**Order of Preference:**
1. **AWS ECR** (if `AWS_ACCESS_KEY_ID` is configured)
2. **Docker Hub** (if `DOCKER_USERNAME` is configured)  
3. **GitHub Container Registry** (default - uses `GITHUB_TOKEN`)

## 📝 Optional File Structure

```
.config/
└── scripts/
    ├── requirements.txt                    # Python dependencies
    ├── increment_version.sh               # Versioning script
    ├── lint.sh                           # Linting script
    ├── generate_lint_report.py           # Lint report generator
    ├── generate_vulnerability_report.py   # Vulnerability report generator
    └── generate_pr_description.py        # PR description generator
```

## 🔄 Workflow

1. **Development**: Push or create PR
2. **CI**: Automatic execution of tests and checks
3. **Code Review**: Bot performs automatic review (if configured)
4. **Merge**: After approval, merge to main
5. **CD**: Automatic deployment with versioning
6. **Release**: Automatic release creation
7. **Notifications**: Feedback via Telegram (if configured)

## 🛠️ Troubleshooting

### Workflow Fails Due to Missing Scripts
**Problem**: Workflow fails because optional scripts don't exist.
**Solution**: Workflows are already configured to continue even without optional scripts.

### Docker Push Failure
**Problem**: Error when pushing Docker image.
**Solution**: Check if registry secrets are correct and repository exists.

### Code Review Bot Not Working
**Problem**: Bot doesn't comment on PRs.
**Solution**: Check if `OPENAI_TOKEN` is configured correctly.

### Telegram Notifications Not Arriving
**Problem**: Messages are not sent.
**Solution**: Check `TELEGRAM_BOT_TOKEN` and `TELEGRAM_CHAT_ID`.

## 📚 Additional Resources

- [GitHub Actions Documentation](https://docs.github.com/en/actions)
- [Docker Build and Push Action](https://github.com/docker/build-push-action)
- [OpenAI API Documentation](https://platform.openai.com/docs)
- [ArgoCD CLI Documentation](https://argo-cd.readthedocs.io/en/stable/cli_installation/)

## 🤝 Contributing

To improve workflows:

1. Fork the repository
2. Create a branch for your modifications
3. Test the changes
4. Open a Pull Request

---

**💡 Tip**: Start by configuring only `GITHUB_TOKEN` (which is already available) and add other secrets as needed for your specific functionalities. 