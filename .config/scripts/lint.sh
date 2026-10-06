#!/usr/bin/env bash
# Run lint and static-analysis tools for the Go project.
# Collects every finding and only fails at the end, printing a summary.
#
# Tools are pinned and executed via `go run`, so a fresh machine needs no
# global installs. Keep these versions in sync with CI (ci.yaml).
set -uo pipefail

STATICCHECK_VERSION="v0.7.0" # latest supporting Go 1.25 (go.mod); v0.8+ needs Go 1.26
GOIMPORTS_VERSION="v0.49.0" # x/tools v0.50+ needs Go 1.26

FAIL=0

# 1. gofmt (formatting)
FMT_OUT=$(gofmt -l .)
if [ -n "$FMT_OUT" ]; then
  echo -e "\nFiles with formatting problems (gofmt):"
  echo "$FMT_OUT"
  FAIL=1
else
  echo "gofmt: OK"
fi

# 2. go vet (common mistakes)
VET_OUT=$(go vet ./... 2>&1 | grep -v "github.com/shoenig/go-m1cpu")
if [ -n "$VET_OUT" ]; then
  echo -e "\nProblems found by go vet:"
  echo "$VET_OUT"
  FAIL=1
else
  echo "go vet: OK"
fi

# 3. staticcheck (advanced static analysis)
# Findings go to stdout and set a non-zero exit; stderr only carries `go run`
# noise (module downloads, toolchain notices), shown just when the run fails.
STATIC_ERR=$(mktemp)
STATIC_OUT=$(go run "honnef.co/go/tools/cmd/staticcheck@${STATICCHECK_VERSION}" ./... 2>"$STATIC_ERR")
STATIC_STATUS=$?
if [ $STATIC_STATUS -ne 0 ]; then
  echo -e "\nProblems found by staticcheck:"
  echo "$STATIC_OUT"
  cat "$STATIC_ERR"
  FAIL=1
else
  echo "staticcheck: OK"
fi
rm -f "$STATIC_ERR"

# 4. goimports (import organization)
IMP_OUT=$(go run "golang.org/x/tools/cmd/goimports@${GOIMPORTS_VERSION}" -l .)
IMP_STATUS=$?
if [ $IMP_STATUS -ne 0 ] || [ -n "$IMP_OUT" ]; then
  echo -e "\nFiles with unorganized imports (goimports):"
  echo "$IMP_OUT"
  FAIL=1
else
  echo "goimports: OK"
fi

if [ $FAIL -eq 0 ]; then
  echo -e "\n✅ Lint finished successfully!"
else
  echo -e "\n❌ Lint problems were found. See the details above."
  exit 1
fi
