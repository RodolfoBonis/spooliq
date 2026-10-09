# Changelog

## v2.16.0 - 2026-10-09

### Changes
- chore: bump version to 2.16.0 (62fd6f3)
- feat(csv): export budgets and customers, import customers from CSV (#108) (f7eab43)
- feat(notification): in-app notifications for budget responses, expiry, low stock and payments (#107) (fa2437b)
- feat(account): forgot password, own profile and password change; record cancel reason (#106) (3308005)


## v2.6.2 - 2026-07-02

### Changes
- chore: bump version to 2.6.2 (003db93)
- fix(cache): fail open when Redis is unavailable instead of panicking (#43) (d802eba)


## v2.6.1 - 2026-06-23

### Changes
- chore: bump version to 2.6.1 (002809c)
- feat(cdn): direct MinIO upload/download (replace rb-cdn) (#41) (99b9c8d)
- 🚀 Release v2.6.0 (#40) (8a51324)
- style(lint): apply gofmt/goimports across lint-target files (1f5bdd7)
- style(redis): apply gofmt import ordering for lint compliance (94d5dee)
- feat(database,redis): add TLS support for PostgreSQL and Redis (b635210)


## v2.6.0 - 2026-03-30

### Changes
- chore: bump version to 2.6.0 (2c36574)
- style(lint): apply gofmt/goimports across lint-target files (1f5bdd7)
- style(redis): apply gofmt import ordering for lint compliance (94d5dee)
- feat(database,redis): add TLS support for PostgreSQL and Redis (b635210)


## v2.5.0 - 2026-02-19

### Changes
- chore: bump version to 2.5.0 (fa2784c)
- fix: resolve errcheck and staticcheck lint errors (d7b78b1)
- chore(deps): bump go-otel-agent to v0.4.0 (4e86108)
- feat(otel): full trace enrichment with HTTP client instrumentation (bb07ed6)
- chore(deps): bump go-otel-agent to v0.2.1 (b66a38b)
- feat: upgrade go-otel-agent to v0.2.0 for full observability (7a7d06a)
- refactor: migrate observability to go-otel-agent library (94457da)
- fix: insert initial status history record on budget creation (6b503d6)
- perf: add missing indexes and consolidate dashboard queries (aac76b5)
- fix: use budget_status_history for step-to-step funnel conversion rates (4d1e88e)
- fix: propagate response data to client on cache miss (9e90b9d)
- feat: add dashboard module with activity tracking (93d2e67)
- fix: add checkout step to backport-auto-merge job (a3cf228)
- chore: trigger CI (b019f13)
- fix: add checkout step to backport-auto-merge job (ba72e59)


## v2.4.0 - 2025-12-28

### Changes
- fix: handle no-change case in prepare-release version bump (fd79849)
- fix: prevent prepare-release from failing when version unchanged (bf30165)
- fix: add SSH setup before GoReleaser for private module access (#20) (01e26bb)
- fix: add SSH setup before GoReleaser for private module access (a070a04)
- 🚀 Release v2.4.0 (#19) (707ad7b)
- Feature/spooliq 44 (#18) (3e59b46)
- Feature/spooliq 44 (#17) (11a9fd0)
- Feature/spooliq 44 (#16) (b53a2c4)
- Feature/spooliq 44 (#15) (67ef42d)
- fix: increase Fx startup timeout to 60s (#14) (02febe2)


## v2.4.0 - 2025-12-28

### Changes
- chore: bump version to 2.4.0 (4052e0a)
- Feature/spooliq 44 (#18) (3e59b46)
- Feature/spooliq 44 (#17) (11a9fd0)
- Feature/spooliq 44 (#16) (b53a2c4)
- Feature/spooliq 44 (#15) (67ef42d)
- fix: increase Fx startup timeout to 60s (#14) (02febe2)


## v2.3.2 - 2025-11-18

### Changes
- chore: bump version to 2.3.2 (991b473)
- fix: update GoReleaser to v2.12.3 and use 'ids' field (38b75c7)


## v2.3.1 - 2025-11-18

### Changes
- chore: bump version to 2.3.1 (ad53e8a)
- fix: pin GoReleaser to specific stable version v2.3.2 (42a090f)


## v2.3.0 - 2025-11-18

### Changes
- chore: bump version to 2.3.0 (eeadfc3)
- fix: increase GoReleaser timeout and disable UPX to resolve error 500 (c6197d4)
- feat: add automatic sync with main in prepare-release workflow (805a0ff)
- chore: sync CHANGELOG.md and version.txt from main (225366c)
- docs: update CHANGELOG for v2.2.0 (06cdc05)
- chore: bump version to 2.2.0 (33704a0)
- Increment version (1122c4d)
- docs: update CHANGELOG for v2.1.0 (2f7c480)
- chore: bump version to 2.1.0 (3eb2c44)


## v2.2.1 - 2025-11-18

### Changes
- chore: bump version to 2.2.1 (0205b62)
- fix: especifica versão do GoReleaser como '~> v2' (85bd9d2)
- fix: usa printf para strings multi-linha (YAML-safe) (1a31994)
- fix: usa heredoc correto para strings multi-linha em bash (9d25504)
- fix: simplifica mensagens em notify-release-cutoff.yaml (6321711)
- fix: corrige sintaxe YAML em notify-release-cutoff.yaml (deae910)
- docs: add executive summary of release workflow (da4bbd1)
- docs: add comprehensive visual flow diagram (465d26b)
- feat: comprehensive redesign of release workflows (138f25e)
- feat: implement total_budgets calculation for customers (3b7876f)
- fix: resolve Swagger documentation errors and warnings (753467a)
- style: fix Go formatting and imports in budget module (e9c6536)
- fix: distribute overhead and profit in PDF item subtotals (9c00c47)
- refactor: improve labor cost calculation with realistic breakdown (2ed96e2)
- feat: add budget list on get customer by id endpoint (8b1d1ab)

