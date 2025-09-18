#!/bin/bash

# Script para gerar documentação Swagger
# Pode ser usado manualmente ou pelo hook de pre-commit

set -e

PROJECT_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$PROJECT_ROOT"

echo "🔄 Gerando documentação Swagger para $(basename "$PROJECT_ROOT")..."

# Verificar se existe go.mod
if [ ! -f "go.mod" ]; then
    echo "❌ Erro: go.mod não encontrado na raiz do projeto"
    exit 1
fi

# Função para gerar swagger usando Docker
generate_swagger_docker() {
    echo "📦 Usando Docker para gerar documentação..."
    
    docker run --rm \
        -v "$PROJECT_ROOT:/workspace" \
        -w /workspace \
        golang:1.23.6-alpine \
        sh -c "
            echo '📥 Instalando swag...' && \
            apk add --no-cache git >/dev/null 2>&1 && \
            go install github.com/swaggo/swag/cmd/swag@latest >/dev/null 2>&1 && \
            echo '📝 Gerando documentação...' && \
            swag init --output docs --outputTypes go,json
        "
}

# Função para verificar se swag está disponível localmente
check_local_swag() {
    if command -v swag >/dev/null 2>&1; then
        echo "🔧 Usando swag local..."
        swag init --output docs --outputTypes go,json
        return 0
    else
        return 1
    fi
}

# Tentar usar swag local, se não funcionar usar Docker
if ! check_local_swag; then
    generate_swagger_docker
fi

# Verificar se os arquivos foram gerados
if [ -f "docs/docs.go" ] && [ -f "docs/swagger.json" ]; then
    echo "✅ Documentação Swagger gerada com sucesso!"
    echo "📁 Arquivos gerados:"
    echo "   - docs/docs.go"
    echo "   - docs/swagger.json" 
    if [ -f "docs/swagger.yaml" ]; then
        echo "   - docs/swagger.yaml"
    fi
    
    # Se chamado com --add-to-git, adicionar ao staging
    if [ "$1" = "--add-to-git" ]; then
        git add docs/
        echo "📝 Arquivos adicionados ao staging do Git"
    fi
else
    echo "❌ Erro: Arquivos de documentação não foram gerados"
    echo "💡 Verifique se há anotações Swagger válidas no código"
    exit 1
fi

echo "🚀 Concluído!"
