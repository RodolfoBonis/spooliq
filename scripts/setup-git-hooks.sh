#!/bin/bash

# Script para configurar hooks do Git em projetos gerados pelo cookiecutter
echo "🔧 Configurando hooks do Git..."

# Verificar se estamos em um repositório Git
if [ ! -d ".git" ]; then
    echo "📝 Inicializando repositório Git..."
    git init
fi

# Criar hook de pre-commit
echo "📋 Criando hook de pre-commit..."
cat > .git/hooks/pre-commit << 'EOF'
#!/bin/bash

# Pre-commit hook para gerar documentação Swagger automaticamente
echo "🔄 Executando pre-commit hook..."

# Chamar o script de geração de swagger
if [ -f "scripts/generate-swagger.sh" ]; then
    echo "📝 Gerando documentação Swagger..."
    bash scripts/generate-swagger.sh --add-to-git
    
    if [ $? -eq 0 ]; then
        echo "✅ Documentação Swagger atualizada e adicionada ao commit!"
    else
        echo "⚠️ Houve um problema na geração do Swagger, mas o commit continuará..."
    fi
else
    echo "⚠️ Script generate-swagger.sh não encontrado, pulando geração de documentação"
fi

echo "🚀 Pre-commit concluído!"
exit 0
EOF

# Tornar o hook executável
chmod +x .git/hooks/pre-commit

echo "✅ Hooks configurados com sucesso!"
echo ""
echo "🎯 O que foi configurado:"
echo "   - Hook de pre-commit para gerar Swagger automaticamente"
echo "   - Documentação será atualizada a cada commit"
echo ""
echo "💡 Para executar manualmente: bash scripts/generate-swagger.sh" 