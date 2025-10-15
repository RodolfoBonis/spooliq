#!/bin/bash

# Colors for output
GREEN='\033[0;32m'
BLUE='\033[0;34m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

API_URL="http://localhost:8080/v1"
OUTPUT_DIR="test_results"

# Create output directory
mkdir -p "$OUTPUT_DIR"

echo -e "${BLUE}🎨 Testando Upload de Logo e Geração de PDF${NC}"
echo "========================================"
echo ""

# Step 1: Check if we have a valid token
if [ ! -f "$OUTPUT_DIR/02_login.json" ]; then
  echo -e "${RED}❌ Token não encontrado. Execute ./test_full_flow.sh primeiro${NC}"
  exit 1
fi

TOKEN=$(cat "$OUTPUT_DIR/02_login.json" | jq -r '.accessToken // .access_token')
if [ -z "$TOKEN" ] || [ "$TOKEN" = "null" ]; then
  echo -e "${RED}❌ Token inválido${NC}"
  exit 1
fi

echo -e "${GREEN}✅ Token encontrado${NC}"
echo ""

# Step 2: Create/download a test logo
echo -e "${BLUE}📸 Preparando logo de teste...${NC}"

# Try to create with ImageMagick first
convert -size 200x200 xc:red "$OUTPUT_DIR/test_logo.png" 2>/dev/null

if [ ! -f "$OUTPUT_DIR/test_logo.png" ]; then
  # Download a simple test image from a public source
  curl -s "https://via.placeholder.com/200x200/ff69b4/ffffff?text=Logo" -o "$OUTPUT_DIR/test_logo.png"
  
  if [ ! -f "$OUTPUT_DIR/test_logo.png" ] || [ ! -s "$OUTPUT_DIR/test_logo.png" ]; then
    echo -e "${RED}❌ Não foi possível criar/baixar logo de teste${NC}"
    exit 1
  fi
fi

echo -e "${GREEN}✅ Logo de teste criada${NC}"
echo ""

# Step 3: Upload logo
echo -e "${BLUE}☁️  STEP 1: Fazendo upload da logo para o CDN${NC}"
curl -s -X POST "$API_URL/company/logo" \
  -H "Authorization: Bearer $TOKEN" \
  -F "logo=@$OUTPUT_DIR/test_logo.png" \
  -o "$OUTPUT_DIR/logo_upload_response.json"

LOGO_URL=$(cat "$OUTPUT_DIR/logo_upload_response.json" | jq -r '.logo_url // empty')

if [ ! -z "$LOGO_URL" ] && [ "$LOGO_URL" != "null" ]; then
  echo -e "${GREEN}✅ Logo enviada com sucesso!${NC}"
  echo "   URL: $LOGO_URL"
  cat "$OUTPUT_DIR/logo_upload_response.json" | jq '.'
else
  echo -e "${RED}❌ Erro ao enviar logo:${NC}"
  cat "$OUTPUT_DIR/logo_upload_response.json" | jq '.'
  exit 1
fi

echo ""

# Step 4: Verify company has logo URL
echo -e "${BLUE}🏢 STEP 2: Verificando se a logo foi salva na empresa${NC}"
curl -s -X GET "$API_URL/company/" \
  -H "Authorization: Bearer $TOKEN" \
  -o "$OUTPUT_DIR/company_with_logo.json"

SAVED_LOGO_URL=$(cat "$OUTPUT_DIR/company_with_logo.json" | jq -r '.logo_url // empty')

if [ ! -z "$SAVED_LOGO_URL" ] && [ "$SAVED_LOGO_URL" != "null" ]; then
  echo -e "${GREEN}✅ Logo URL salva na empresa!${NC}"
  echo "   Logo URL: $SAVED_LOGO_URL"
else
  echo -e "${RED}❌ Logo URL não encontrada na empresa${NC}"
fi

echo ""

# Step 5: Generate PDF with logo
if [ -f "$OUTPUT_DIR/10_budget.json" ]; then
  BUDGET_ID=$(cat "$OUTPUT_DIR/10_budget.json" | jq -r '.budget.id')
  
  if [ ! -z "$BUDGET_ID" ] && [ "$BUDGET_ID" != "null" ]; then
    echo -e "${BLUE}📄 STEP 3: Gerando PDF com logo${NC}"
    
    curl -s -X GET "$API_URL/budgets/$BUDGET_ID/pdf" \
      -H "Authorization: Bearer $TOKEN" \
      --output "$OUTPUT_DIR/budget_with_logo.pdf"
    
    if [ -f "$OUTPUT_DIR/budget_with_logo.pdf" ]; then
      FILE_HEADER=$(head -c 4 "$OUTPUT_DIR/budget_with_logo.pdf")
      if [[ "$FILE_HEADER" == "%PDF" ]]; then
        echo -e "${GREEN}✅ PDF com logo gerado com sucesso!${NC}"
        ls -lh "$OUTPUT_DIR/budget_with_logo.pdf"
        echo ""
        echo -e "${BLUE}📖 Abrindo PDF...${NC}"
        open "$OUTPUT_DIR/budget_with_logo.pdf"
      else
        echo -e "${RED}❌ Erro ao gerar PDF:${NC}"
        cat "$OUTPUT_DIR/budget_with_logo.pdf" | jq '.' 2>/dev/null || cat "$OUTPUT_DIR/budget_with_logo.pdf"
      fi
    fi
  else
    echo -e "${YELLOW}⚠️  Budget ID não encontrado. Pulando geração de PDF.${NC}"
  fi
else
  echo -e "${YELLOW}⚠️  Arquivo de budget não encontrado. Execute ./test_full_flow.sh primeiro.${NC}"
fi

echo ""
echo "═══════════════════════════════════════"
echo -e "${GREEN}✅ TESTE DE LOGO COMPLETO!${NC}"
echo "═══════════════════════════════════════"
echo ""
echo "Arquivos gerados:"
echo "  • test_logo.png - Logo de teste"
echo "  • logo_upload_response.json - Resposta do upload"
echo "  • company_with_logo.json - Empresa com logo"
echo "  • budget_with_logo.pdf - PDF com logo"
echo ""

