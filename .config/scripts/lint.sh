#!/bin/bash
# Script para rodar ferramentas de lint e análise estática no projeto Go
# Agora coleta todos os erros e só falha no final, mostrando um resumo

FAIL=0

# 1. gofmt (formatação)
FMT_OUT=$(gofmt -l .)
if [ -n "$FMT_OUT" ]; then
  echo -e "\nArquivos com problemas de formatação (gofmt):"
  echo "$FMT_OUT"
  FAIL=1
else
  echo "gofmt: OK"
fi

# 2. go vet (erros comuns)
VET_OUT=$(go vet ./... 2>&1)
if [ -n "$VET_OUT" ]; then
  echo -e "\nProblemas encontrados pelo go vet:"
  echo "$VET_OUT"
  FAIL=1
else
  echo "go vet: OK"
fi

# 3. golint (boas práticas)
if ! command -v golint &> /dev/null; then
  echo "golint não encontrado. Instale com: go install golang.org/x/lint/golint@latest"
  FAIL=1
else
  LINT_OUT=$(golint ./...)
  if [ -n "$LINT_OUT" ]; then
    echo -e "\nProblemas encontrados pelo golint:"
    echo "$LINT_OUT"
    FAIL=1
  else
    echo "golint: OK"
  fi
fi

# 4. staticcheck (análise avançada)
if ! command -v staticcheck &> /dev/null; then
  echo "staticcheck não encontrado. Instale com: go install honnef.co/go/tools/cmd/staticcheck@latest"
  FAIL=1
else
  STATIC_OUT=$(staticcheck ./... 2>&1)
  if [ -n "$STATIC_OUT" ]; then
    echo -e "\nProblemas encontrados pelo staticcheck:"
    echo "$STATIC_OUT"
    FAIL=1
  else
    echo "staticcheck: OK"
  fi
fi

# 5. goimports (organização dos imports)
if ! command -v goimports &> /dev/null; then
  echo "goimports não encontrado. Instale com: go install golang.org/x/tools/cmd/goimports@latest"
  FAIL=1
else
  IMP_OUT=$(goimports -l .)
  if [ -n "$IMP_OUT" ]; then
    echo -e "\nArquivos com imports desorganizados (goimports):"
    echo "$IMP_OUT"
    FAIL=1
  else
    echo "goimports: OK"
  fi
fi

if [ $FAIL -eq 0 ]; then
  echo -e "\n✅ Lint finalizado com sucesso!"
else
  echo -e "\n❌ Foram encontrados problemas de lint. Veja os detalhes acima."
  exit 1
fi 