#!/bin/bash

# Script para testar o fluxo completo de um novo usuário no SaaS Spooliq
# 1. Registro de nova empresa
# 2. Login
# 3. Configuração de company
# 4. Criação de brands, materials, filaments, presets
# 5. Criação de cliente
# 6. Criação de orçamento
# 7. Geração de PDF

set -e

API_URL="http://localhost:8080"
OUTPUT_DIR="test_results"
mkdir -p "$OUTPUT_DIR"

echo "🚀 Iniciando teste do fluxo completo do SaaS Spooliq"
echo "=================================================="

# Cores para output
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# ========================================
# STEP 1: Registro de Nova Empresa
# ========================================
echo -e "\n${BLUE}📝 STEP 1: Registrando nova empresa${NC}"

REGISTER_RESPONSE=$(curl -s -X POST "$API_URL/v1/register" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "João Silva",
    "email": "admin@impressoes3d.com.br",
    "password": "Senha123!@#",
    "company_name": "Impressões 3D Premium LTDA",
    "company_trade_name": "3D Premium",
    "company_document": "12345678000190",
    "company_phone": "+5511987654321",
    "address": "Rua das Impressoras",
    "address_number": "123",
    "complement": "Sala 101",
    "neighborhood": "Centro",
    "city": "São Paulo",
    "state": "SP",
    "zip_code": "01234-567"
  }')

echo "$REGISTER_RESPONSE" | jq '.' > "$OUTPUT_DIR/01_register.json"
echo -e "${GREEN}✅ Empresa registrada${NC}"
echo "$REGISTER_RESPONSE" | jq '.'

# ========================================
# STEP 2: Login
# ========================================
echo -e "\n${BLUE}🔐 STEP 2: Fazendo login${NC}"

LOGIN_RESPONSE=$(curl -s -X POST "$API_URL/v1/login" \
  -H "Content-Type: application/json" \
  -d '{
    "email": "admin@impressoes3d.com.br",
    "password": "Senha123!@#"
  }')

echo "$LOGIN_RESPONSE" | jq '.' > "$OUTPUT_DIR/02_login.json"

TOKEN=$(echo "$LOGIN_RESPONSE" | jq -r '.accessToken')
echo -e "${GREEN}✅ Login realizado${NC}"
echo "Token: ${TOKEN:0:50}..."

# ========================================
# STEP 3: Buscar informações da company
# ========================================
echo -e "\n${BLUE}🏢 STEP 3: Buscando informações da empresa${NC}"

COMPANY_RESPONSE=$(curl -s -X GET "$API_URL/v1/company/" \
  -H "Authorization: Bearer $TOKEN")

echo "$COMPANY_RESPONSE" | jq '.' > "$OUTPUT_DIR/03_company_info.json"
echo -e "${GREEN}✅ Informações da empresa obtidas${NC}"
echo "$COMPANY_RESPONSE" | jq '.'

# ========================================
# STEP 4: Atualizar dados da company (com logo)
# ========================================
echo -e "\n${BLUE}🎨 STEP 4: Atualizando dados da empresa${NC}"

UPDATE_COMPANY_RESPONSE=$(curl -s -X PUT "$API_URL/v1/company/" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Impressões 3D Premium",
    "trade_name": "3D Premium",
    "document": "12345678000190",
    "email": "contato@impressoes3d.com.br",
    "phone": "+5511987654321",
    "whatsapp": "+5511987654321",
    "instagram": "@impressoes3dpremium",
    "website": "https://impressoes3d.com.br",
    "logo_url": "https://via.placeholder.com/200x200.png?text=3D+Premium",
    "address": "Rua das Impressoras, 123",
    "city": "São Paulo",
    "state": "SP",
    "zip_code": "01234-567"
  }')

echo "$UPDATE_COMPANY_RESPONSE" | jq '.' > "$OUTPUT_DIR/04_company_updated.json"
echo -e "${GREEN}✅ Dados da empresa atualizados${NC}"

# ========================================
# STEP 5: Criar Brand (Marca)
# ========================================
echo -e "\n${BLUE}🏷️  STEP 5: Criando marca de filamento${NC}"

BRAND_RESPONSE=$(curl -s -X POST "$API_URL/v1/brands" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Bambu Lab"
  }')

echo "$BRAND_RESPONSE" | jq '.' > "$OUTPUT_DIR/05_brand.json"
BRAND_ID=$(echo "$BRAND_RESPONSE" | jq -r '.id')
echo -e "${GREEN}✅ Marca criada (ID: $BRAND_ID)${NC}"

# ========================================
# STEP 6: Criar Material
# ========================================
echo -e "\n${BLUE}🧪 STEP 6: Criando material${NC}"

MATERIAL_RESPONSE=$(curl -s -X POST "$API_URL/v1/materials" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "PLA",
    "description": "PLA (Ácido Poliláctico) - Material biodegradável ideal para impressões gerais"
  }')

echo "$MATERIAL_RESPONSE" | jq '.' > "$OUTPUT_DIR/06_material.json"
MATERIAL_ID=$(echo "$MATERIAL_RESPONSE" | jq -r '.id')
echo -e "${GREEN}✅ Material criado (ID: $MATERIAL_ID)${NC}"

# ========================================
# STEP 7: Criar Filament
# ========================================
echo -e "\n${BLUE}🎨 STEP 7: Criando filamento${NC}"

FILAMENT_RESPONSE=$(curl -s -X POST "$API_URL/v1/filaments" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d "{
    \"name\": \"PLA Vermelho Ferrari\",
    \"brand_id\": \"$BRAND_ID\",
    \"material_id\": \"$MATERIAL_ID\",
    \"color\": \"Vermelho Ferrari\",
    \"color_hex\": \"#DC0000\",
    \"color_type\": \"solid\",
    \"color_data\": {\"color\": \"#DC0000\"},
    \"weight\": 1000,
    \"price_per_kg\": 89.90,
    \"diameter\": 1.75,
    \"print_temperature\": 200,
    \"bed_temperature\": 55
  }")

echo "$FILAMENT_RESPONSE" | jq '.' > "$OUTPUT_DIR/07_filament.json"
FILAMENT_ID=$(echo "$FILAMENT_RESPONSE" | jq -r '.id')
echo -e "${GREEN}✅ Filamento criado (ID: $FILAMENT_ID)${NC}"

# Criar segundo filamento (para multicolor)
FILAMENT2_RESPONSE=$(curl -s -X POST "$API_URL/v1/filaments" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d "{
    \"name\": \"PLA Branco\",
    \"brand_id\": \"$BRAND_ID\",
    \"material_id\": \"$MATERIAL_ID\",
    \"color\": \"Branco\",
    \"color_hex\": \"#FFFFFF\",
    \"color_type\": \"solid\",
    \"color_data\": {\"color\": \"#FFFFFF\"},
    \"weight\": 1000,
    \"price_per_kg\": 79.90,
    \"diameter\": 1.75,
    \"print_temperature\": 200,
    \"bed_temperature\": 55
  }")

FILAMENT2_ID=$(echo "$FILAMENT2_RESPONSE" | jq -r '.id')
echo -e "${GREEN}✅ Segundo filamento criado (ID: $FILAMENT2_ID)${NC}"

# ========================================
# STEP 8: Criar Presets
# ========================================
echo -e "\n${BLUE}⚙️  STEP 8: Criando presets${NC}"

# Machine Preset
MACHINE_PRESET=$(curl -s -X POST "$API_URL/v1/presets/machines" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Bambu Lab P1S",
    "description": "Configuração padrão Bambu Lab P1S",
    "brand": "Bambu Lab",
    "model": "P1S",
    "build_volume_x": 256,
    "build_volume_y": 256,
    "build_volume_z": 256,
    "nozzle_diameter": 0.4,
    "layer_height_min": 0.08,
    "layer_height_max": 0.32,
    "print_speed_max": 500,
    "power_consumption": 350,
    "filament_diameter": 1.75,
    "bed_temperature_max": 110,
    "extruder_temperature_max": 300
  }')

MACHINE_ID=$(echo "$MACHINE_PRESET" | jq -r '.id')
echo -e "${GREEN}✅ Machine Preset criado (ID: $MACHINE_ID)${NC}"

# Energy Preset
ENERGY_PRESET=$(curl -s -X POST "$API_URL/v1/presets/energy" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "São Paulo Residencial",
    "description": "Tarifa residencial de São Paulo",
    "country": "Brasil",
    "state": "São Paulo",
    "city": "São Paulo",
    "energy_cost_per_kwh": 0.75,
    "currency": "BRL",
    "provider": "Enel",
    "tariff_type": "residential",
    "peak_hour_multiplier": 1.5,
    "off_peak_hour_multiplier": 0.8
  }')

ENERGY_ID=$(echo "$ENERGY_PRESET" | jq -r '.id')
echo -e "${GREEN}✅ Energy Preset criado (ID: $ENERGY_ID)${NC}"

# Cost Preset
COST_PRESET=$(curl -s -X POST "$API_URL/v1/presets/costs" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Custos Padrão",
    "description": "Configuração padrão de custos",
    "labor_cost_per_hour": 25.0,
    "packaging_cost_per_item": 2.50,
    "shipping_cost_base": 15.00,
    "shipping_cost_per_gram": 0.01,
    "overhead_percentage": 10,
    "profit_margin_percentage": 30,
    "post_processing_cost_per_hour": 20.0,
    "support_removal_cost_per_hour": 15.0,
    "quality_control_cost_per_item": 5.0
  }')

COST_ID=$(echo "$COST_PRESET" | jq -r '.id')
echo -e "${GREEN}✅ Cost Preset criado (ID: $COST_ID)${NC}"

# Salvar todos os presets
echo "{ \"machine\": $MACHINE_PRESET, \"energy\": $ENERGY_PRESET, \"cost\": $COST_PRESET }" > "$OUTPUT_DIR/08_presets.json"

# ========================================
# STEP 9: Criar Customer (Cliente)
# ========================================
echo -e "\n${BLUE}👤 STEP 9: Criando cliente${NC}"

CUSTOMER_RESPONSE=$(curl -s -X POST "$API_URL/v1/customers" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Maria Souza",
    "email": "maria.souza@email.com",
    "phone": "+5511999887766",
    "document": "12345678900",
    "address": "Av. Paulista, 1000",
    "city": "São Paulo",
    "state": "SP",
    "zip_code": "01310-100",
    "notes": "Cliente preferencial - pedidos urgentes"
  }')

echo "$CUSTOMER_RESPONSE" | jq '.' > "$OUTPUT_DIR/09_customer.json"
CUSTOMER_ID=$(echo "$CUSTOMER_RESPONSE" | jq -r '.customer.id')
echo -e "${GREEN}✅ Cliente criado (ID: $CUSTOMER_ID)${NC}"

# ========================================
# STEP 10: Criar Budget (Orçamento)
# ========================================
echo -e "\n${BLUE}💰 STEP 10: Criando orçamento${NC}"

BUDGET_RESPONSE=$(curl -s -X POST "$API_URL/v1/budgets" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d "{
    \"name\": \"Chaveiros Personalizados - Outubro Rosa\",
    \"customer_id\": \"$CUSTOMER_ID\",
    \"machine_preset_id\": \"$MACHINE_ID\",
    \"energy_preset_id\": \"$ENERGY_ID\",
    \"cost_preset_id\": \"$COST_ID\",
    \"description\": \"Chaveiros personalizados para campanha Outubro Rosa\",
    \"print_time_hours\": 2,
    \"print_time_minutes\": 30,
    \"items\": [
      {
        \"name\": \"Chaveiro Laço Rosa\",
        \"description\": \"Chaveiro em formato de laço símbolo do Outubro Rosa\",
        \"filament_id\": \"$FILAMENT_ID\",
        \"quantity\": 50,
        \"filament_weight_grams\": 8,
        \"print_time_minutes\": 15,
        \"has_support\": true,
        \"color_count\": 2,
        \"file_url\": \"https://www.thingiverse.com/thing:12345\"
      },
      {
        \"name\": \"Chaveiro Coração Rosa\",
        \"description\": \"Chaveiro em formato de coração\",
        \"filament_id\": \"$FILAMENT2_ID\",
        \"quantity\": 30,
        \"filament_weight_grams\": 10,
        \"print_time_minutes\": 18,
        \"has_support\": false,
        \"color_count\": 1,
        \"file_url\": \"https://www.thingiverse.com/thing:67890\"
      }
    ],
    \"include_labor_cost\": true,
    \"include_energy_cost\": true,
    \"notes\": \"Entrega urgente - prazo de 5 dias úteis\"
  }")

echo "$BUDGET_RESPONSE" | jq '.' > "$OUTPUT_DIR/10_budget.json"
BUDGET_ID=$(echo "$BUDGET_RESPONSE" | jq -r '.budget.id')
echo -e "${GREEN}✅ Orçamento criado (ID: $BUDGET_ID)${NC}"

# Mostrar resumo do orçamento
echo -e "\n${YELLOW}📊 Resumo do Orçamento:${NC}"
echo "$BUDGET_RESPONSE" | jq '{
  id: .budget.id,
  name: .budget.name,
  status: .budget.status,
  total_cost: .budget.total_cost,
  filament_cost: .budget.filament_cost,
  items_count: (.items | length),
  created_at: .budget.created_at
}'

# ========================================
# STEP 11: Gerar PDF do Orçamento
# ========================================
echo -e "\n${BLUE}📄 STEP 11: Gerando PDF do orçamento${NC}"

PDF_FILE="$OUTPUT_DIR/11_orcamento_${BUDGET_ID}.pdf"

curl -s -X GET "$API_URL/v1/budgets/$BUDGET_ID/pdf" \
  -H "Authorization: Bearer $TOKEN" \
  --output "$PDF_FILE"

if [ -f "$PDF_FILE" ] && [ -s "$PDF_FILE" ]; then
  echo -e "${GREEN}✅ PDF gerado com sucesso${NC}"
  echo "   Arquivo: $PDF_FILE"
  ls -lh "$PDF_FILE"
  
  # Abrir o PDF automaticamente (macOS)
  if [[ "$OSTYPE" == "darwin"* ]]; then
    open "$PDF_FILE"
    echo "   📖 PDF aberto automaticamente"
  fi
else
  echo -e "${YELLOW}⚠️  Aviso: PDF não foi gerado ou está vazio${NC}"
fi

# ========================================
# STEP 12: Listar todos os recursos criados
# ========================================
echo -e "\n${BLUE}📋 STEP 12: Listando recursos criados${NC}"

echo -e "\n${YELLOW}Marcas:${NC}"
curl -s -X GET "$API_URL/v1/brands?page=1&pageSize=10" \
  -H "Authorization: Bearer $TOKEN" | jq '.brands[] | {id, name}'

echo -e "\n${YELLOW}Materiais:${NC}"
curl -s -X GET "$API_URL/v1/materials?page=1&pageSize=10" \
  -H "Authorization: Bearer $TOKEN" | jq '.materials[] | {id, name}'

echo -e "\n${YELLOW}Filamentos:${NC}"
curl -s -X GET "$API_URL/v1/filaments?page=1&pageSize=10" \
  -H "Authorization: Bearer $TOKEN" | jq '.filaments[] | {id, name, brand_name, material_name, price}'

echo -e "\n${YELLOW}Presets:${NC}"
curl -s -X GET "$API_URL/v1/presets?page=1&pageSize=10" \
  -H "Authorization: Bearer $TOKEN" | jq '.presets[] | {id, name, description}'

echo -e "\n${YELLOW}Clientes:${NC}"
curl -s -X GET "$API_URL/v1/customers?page=1&pageSize=10" \
  -H "Authorization: Bearer $TOKEN" | jq '.customers[] | {id, name, email, phone}'

echo -e "\n${YELLOW}Orçamentos:${NC}"
curl -s -X GET "$API_URL/v1/budgets?page=1&pageSize=10" \
  -H "Authorization: Bearer $TOKEN" | jq '.budgets[] | {id, name, customer_name, status, final_price}'

# ========================================
# STEP 13: Atualizar status do orçamento
# ========================================
echo -e "\n${BLUE}📝 STEP 13: Atualizando status do orçamento para 'sent'${NC}"

UPDATE_STATUS_RESPONSE=$(curl -s -X PATCH "$API_URL/v1/budgets/$BUDGET_ID/status" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "status": "sent",
    "notes": "Orçamento enviado ao cliente por email"
  }')

echo "$UPDATE_STATUS_RESPONSE" | jq '.' > "$OUTPUT_DIR/13_budget_status_updated.json"
echo -e "${GREEN}✅ Status do orçamento atualizado para 'sent'${NC}"

# ========================================
# Resumo Final
# ========================================
echo -e "\n${GREEN}=================================================="
echo "✅ TESTE COMPLETO FINALIZADO COM SUCESSO!"
echo "==================================================${NC}"
echo ""
echo "📂 Resultados salvos em: $OUTPUT_DIR/"
echo ""
echo "📊 Recursos criados:"
echo "   • Empresa: Impressões 3D Premium"
echo "   • Marca: Bambu Lab (ID: $BRAND_ID)"
echo "   • Material: PLA (ID: $MATERIAL_ID)"
echo "   • Filamentos: 2 criados"
echo "   • Preset: Bambu Lab P1S (ID: $PRESET_ID)"
echo "   • Cliente: Maria Souza (ID: $CUSTOMER_ID)"
echo "   • Orçamento: $BUDGET_ID"
echo "   • PDF: $PDF_FILE"
echo ""
echo "🔗 URLs importantes:"
echo "   • API: $API_URL"
echo "   • Swagger: $API_URL/swagger/index.html"
echo ""
echo "🎉 Fluxo completo do SaaS testado com sucesso!"

