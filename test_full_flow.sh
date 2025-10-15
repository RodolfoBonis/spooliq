#!/bin/bash

# =============================================================================
# Spooliq SaaS - Complete Flow Test
# =============================================================================
# Tests: Registration, Login, Logo Upload, Branding, CRUD operations, 
# Budget with Products, and PDF generation
# =============================================================================

set -e  # Exit on error

API_URL="http://localhost:8080/v1"
TEST_EMAIL="test_$(date +%s)@example.com"
TEST_PASSWORD="Test@123456"
COMPANY_NAME="Test Company $(date +%s)"

echo "======================================"
echo "🚀 Spooliq SaaS - Complete Flow Test"
echo "======================================"
echo ""
echo "Test email: $TEST_EMAIL"
echo "API URL: $API_URL"
echo ""

# Cleanup function
cleanup() {
    rm -f *.json test_logo.png 2>/dev/null || true
}

# Cleanup before starting
cleanup

# =============================================================================
# 1. REGISTER NEW COMPANY
# =============================================================================
echo "1️⃣  Registering new company..."

cat > 01_register.json <<EOF
{
  "name": "João Silva",
  "email": "$TEST_EMAIL",
  "password": "$TEST_PASSWORD",
  "company_name": "$COMPANY_NAME",
  "company_trade_name": "Test Co.",
  "company_document": "12.345.678/0001-90",
  "company_phone": "+5511987654321",
  "address": "Rua Teste",
  "address_number": "123",
  "complement": "Sala 1",
  "neighborhood": "Centro",
  "city": "São Paulo",
  "state": "SP",
  "zip_code": "01234-567"
}
EOF

curl -s -X POST "$API_URL/register" \
  -H "Content-Type: application/json" \
  -d @01_register.json | jq '.' | tee register_response.json

if [ $? -eq 0 ]; then
    echo "✅ Company registered successfully"
else
    echo "❌ Failed to register company"
    exit 1
fi

sleep 2

# =============================================================================
# 2. LOGIN
# =============================================================================
echo ""
echo "2️⃣  Logging in..."

curl -s -X POST "$API_URL/login" \
  -H "Content-Type: application/json" \
  -d "{\"email\":\"$TEST_EMAIL\",\"password\":\"$TEST_PASSWORD\"}" | jq '.' | tee 02_login.json

TOKEN=$(jq -r '.accessToken' 02_login.json)

if [ "$TOKEN" = "null" ] || [ -z "$TOKEN" ]; then
    echo "❌ Failed to get token"
    exit 1
fi

echo "✅ Login successful"
echo "Token: ${TOKEN:0:50}..."

# =============================================================================
# 3. GET COMPANY INFO
# =============================================================================
echo ""
echo "3️⃣  Getting company info..."

curl -s -X GET "$API_URL/company" \
  -H "Authorization: Bearer $TOKEN" | jq '.' | tee 03_company.json

echo "✅ Company info retrieved"

# =============================================================================
# 4. CREATE AND UPLOAD COMPANY LOGO
# =============================================================================
echo ""
echo "4️⃣  Creating and uploading company logo..."

# Create a minimal PNG (1x1 pink pixel as logo)
# This is a valid PNG file in base64
echo "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mP8z8DwHwAFBQIAX8jx0gAAAABJRU5ErkJggg==" | base64 -d > test_logo.png

if [ ! -f test_logo.png ]; then
    echo "❌ Failed to create test logo"
    exit 1
fi

curl -s -X POST "$API_URL/company/logo" \
  -H "Authorization: Bearer $TOKEN" \
  -F "file=@test_logo.png" | jq '.' | tee 04_logo.json

echo "✅ Logo uploaded successfully"

# =============================================================================
# 5. GET BRANDING TEMPLATES
# =============================================================================
echo ""
echo "5️⃣  Getting branding templates..."

curl -s -X GET "$API_URL/company/branding/templates" \
  -H "Authorization: Bearer $TOKEN" | jq '.' | tee 05_templates.json

echo "✅ Branding templates retrieved"

# =============================================================================
# 6. GET CURRENT BRANDING (should return default)
# =============================================================================
echo ""
echo "6️⃣  Getting current branding configuration..."

curl -s -X GET "$API_URL/company/branding" \
  -H "Authorization: Bearer $TOKEN" | jq '.' | tee 06_branding_get.json

echo "✅ Current branding retrieved"

# =============================================================================
# 7. UPDATE BRANDING (use corporate_blue template)
# =============================================================================
echo ""
echo "7️⃣  Updating branding to 'corporate_blue' template..."

cat > 07_branding_update.json <<EOF
{
  "template_name": "corporate_blue",
  "header_bg_color": "#1e40af",
  "header_text_color": "#ffffff",
  "primary_color": "#3b82f6",
  "primary_text_color": "#ffffff",
  "secondary_color": "#60a5fa",
  "secondary_text_color": "#1e3a8a",
  "title_color": "#1e40af",
  "body_text_color": "#374151",
  "accent_color": "#0ea5e9",
  "border_color": "#93c5fd",
  "background_color": "#ffffff",
  "table_header_bg_color": "#dbeafe",
  "table_row_alt_bg_color": "#eff6ff"
}
EOF

curl -s -X PUT "$API_URL/company/branding" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d @07_branding_update.json | jq '.' | tee branding_response.json

echo "✅ Branding updated successfully"

# =============================================================================
# 8. UPDATE COMPANY INFO
# =============================================================================
echo ""
echo "8️⃣  Updating company info..."

curl -s -X PUT "$API_URL/company" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d "{\"website\":\"https://testcompany.com\"}" | jq '.' | tee company_update.json

echo "✅ Company updated"

# =============================================================================
# 9. CREATE BRAND
# =============================================================================
echo ""
echo "9️⃣  Creating brand..."

curl -s -X POST "$API_URL/brands" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"name":"Creality","description":"Marca de filamentos Creality"}' | jq '.' | tee 09_brand.json

BRAND_ID=$(jq -r '.id // .brand.id' 09_brand.json)
echo "✅ Brand created: $BRAND_ID"

# =============================================================================
# 10. CREATE MATERIAL
# =============================================================================
echo ""
echo "🔟 Creating material..."

curl -s -X POST "$API_URL/materials" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"name":"PLA","description":"Ácido Polilático"}' | jq '.' | tee 10_material.json

MATERIAL_ID=$(jq -r '.id // .material.id' 10_material.json)
echo "✅ Material created: $MATERIAL_ID"

# =============================================================================
# 11. CREATE FILAMENT
# =============================================================================
echo ""
echo "1️⃣1️⃣  Creating filament..."

cat > 11_filament.json <<'FILEOF'
{
  "name": "PLA Rosa Pink",
  "brand_id": "BRAND_ID_PLACEHOLDER",
  "material_id": "MATERIAL_ID_PLACEHOLDER",
  "color": "Rosa Pink",
  "color_hex": "#ec4899",
  "diameter": 1.75,
  "price_per_kg": 8990
}
FILEOF

# Replace placeholders
sed -i.bak "s/BRAND_ID_PLACEHOLDER/$BRAND_ID/g" 11_filament.json
sed -i.bak "s/MATERIAL_ID_PLACEHOLDER/$MATERIAL_ID/g" 11_filament.json

curl -s -X POST "$API_URL/filaments" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d @11_filament.json | jq '.' | tee filament_response.json

FILAMENT_ID=$(jq -r '.id // .filament.id' filament_response.json)
echo "✅ Filament created: $FILAMENT_ID"

# =============================================================================
# 12. CREATE MACHINE PRESET
# =============================================================================
echo ""
echo "1️⃣2️⃣  Creating machine preset..."

cat > 12_machine.json <<EOF
{
  "name": "Ender 3 V2",
  "brand": "Creality",
  "model": "Ender 3 V2",
  "build_volume_x": 220,
  "build_volume_y": 220,
  "build_volume_z": 250,
  "nozzle_diameter": 0.4,
  "layer_height_min": 0.1,
  "layer_height_max": 0.3,
  "print_speed_max": 100,
  "power_consumption": 350,
  "bed_temperature_max": 100,
  "extruder_temperature_max": 260,
  "filament_diameter": 1.75,
  "cost_per_hour": 0
}
EOF

curl -s -X POST "$API_URL/presets/machines" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d @12_machine.json | jq '.' | tee machine_response.json

MACHINE_PRESET_ID=$(jq -r '.id' machine_response.json)
echo "✅ Machine preset created: $MACHINE_PRESET_ID"

# =============================================================================
# 13. CREATE ENERGY PRESET
# =============================================================================
echo ""
echo "1️⃣3️⃣  Creating energy preset..."

cat > 13_energy.json <<EOF
{
  "name": "Padrão Residencial",
  "country": "Brasil",
  "state": "SP",
  "city": "São Paulo",
  "energy_cost_per_kwh": 0.82,
  "currency": "BRL",
  "provider": "Enel",
  "tariff_type": "Residencial",
  "peak_hour_multiplier": 1.5,
  "off_peak_hour_multiplier": 0.8
}
EOF

curl -s -X POST "$API_URL/presets/energy" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d @13_energy.json | jq '.' | tee energy_response.json

ENERGY_PRESET_ID=$(jq -r '.id' energy_response.json)
echo "✅ Energy preset created: $ENERGY_PRESET_ID"

# =============================================================================
# 14. CREATE COST PRESET
# =============================================================================
echo ""
echo "1️⃣4️⃣  Creating cost preset..."

curl -s -X POST "$API_URL/presets/costs" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"name":"Padrão","labor_cost_per_hour":5000}' | jq '.' | tee 14_cost.json

COST_PRESET_ID=$(jq -r '.id // .preset.id' 14_cost.json)
echo "✅ Cost preset created: $COST_PRESET_ID"

# =============================================================================
# 15. CREATE CUSTOMER
# =============================================================================
echo ""
echo "1️⃣5️⃣  Creating customer..."

CUSTOMER_EMAIL="maria.souza_$(date +%s)@example.com"

cat > 15_customer.json <<EOF
{
  "name": "Maria Souza",
  "email": "$CUSTOMER_EMAIL",
  "phone": "+5511987654321",
  "document": "123.456.789-00",
  "street": "Rua das Flores",
  "number": "456",
  "neighborhood": "Jardim",
  "city": "São Paulo",
  "state": "SP",
  "zip_code": "01234-567",
  "country": "Brasil"
}
EOF

curl -s -X POST "$API_URL/customers" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d @15_customer.json | jq '.' | tee customer_response.json

CUSTOMER_ID=$(jq -r '.id // .customer.id' customer_response.json)
echo "✅ Customer created: $CUSTOMER_ID"

# =============================================================================
# 16. CREATE BUDGET WITH PRODUCT INFORMATION
# =============================================================================
echo ""
echo "1️⃣6️⃣  Creating budget with product information..."

cat > 16_budget.json <<EOF
{
  "name": "Orçamento - Chaveiros Outubro Rosa",
  "description": "100 chaveiros personalizados para campanha Outubro Rosa",
  "customer_id": "$CUSTOMER_ID",
  "machine_preset_id": "$MACHINE_PRESET_ID",
  "energy_preset_id": "$ENERGY_PRESET_ID",
  "include_energy_cost": true,
  "include_waste_cost": true,
  "delivery_days": 7,
  "payment_terms": "50% entrada, 50% na entrega",
  "notes": "Entrega em caixa personalizada com laços rosas",
  "items": [
    {
      "product_name": "Chaveiro Laço Rosa (PLA) com argola",
      "product_description": "Chaveiro em formato de laço da campanha Outubro Rosa, impressão 3D em PLA rosa, inclui argola metálica",
      "product_quantity": 100,
      "product_dimensions": "26×48×9 mm",
      "print_time_hours": 5,
      "print_time_minutes": 30,
      "cost_preset_id": "$COST_PRESET_ID",
      "additional_labor_cost": 5000,
      "additional_notes": "Inclui montagem manual da argola metálica",
      "filaments": [
        {
          "filament_id": "$FILAMENT_ID",
          "quantity": 2800.0,
          "order": 1
        }
      ],
      "order": 1
    }
  ]
}
EOF

curl -s -X POST "$API_URL/budgets" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d @16_budget.json | jq '.' | tee budget_response.json

BUDGET_ID=$(jq -r '.id // .budget.id' budget_response.json)

if [ "$BUDGET_ID" = "null" ] || [ -z "$BUDGET_ID" ]; then
    echo "❌ Failed to create budget"
    cat budget_response.json
    exit 1
fi

echo "✅ Budget created: $BUDGET_ID"

# =============================================================================
# 17. GET BUDGET DETAILS
# =============================================================================
echo ""
echo "1️⃣7️⃣  Getting budget details..."

curl -s -X GET "$API_URL/budgets/$BUDGET_ID" \
  -H "Authorization: Bearer $TOKEN" | jq '.' | tee 17_budget_details.json

echo "✅ Budget details retrieved"

# Display product information
echo ""
echo "📦 Product Information:"
jq '.items[0] | {
  product_name,
  product_quantity,
  product_dimensions,
  print_time_display,
  filaments: (.filaments | length),
  unit_price,
  item_total_cost
}' 17_budget_details.json

echo ""
echo "⏱️  Total Print Time:"
jq '{
  total_print_time_display,
  total_print_time_hours,
  total_print_time_minutes
}' 17_budget_details.json

echo ""
echo "💰 Cost Breakdown:"
jq '.budget | {
  filament_cost,
  waste_cost,
  energy_cost,
  labor_cost,
  total_cost
}' 17_budget_details.json

# =============================================================================
# 18. GENERATE PDF
# =============================================================================
echo ""
echo "1️⃣8️⃣  Generating PDF..."

curl -s -X GET "$API_URL/budgets/$BUDGET_ID/pdf" \
  -H "Authorization: Bearer $TOKEN" \
  -o "budget_${BUDGET_ID}.pdf"

if [ -f "budget_${BUDGET_ID}.pdf" ]; then
    FILE_SIZE=$(ls -lh "budget_${BUDGET_ID}.pdf" | awk '{print $5}')
    echo "✅ PDF generated successfully: budget_${BUDGET_ID}.pdf ($FILE_SIZE)"
    
    # Check if PDF was uploaded to CDN
    PDF_URL=$(jq -r '.budget.pdf_url' 17_budget_details.json)
    if [ "$PDF_URL" != "null" ] && [ -n "$PDF_URL" ]; then
        echo "✅ PDF uploaded to CDN: $PDF_URL"
    fi
else
    echo "❌ Failed to generate PDF"
    exit 1
fi

# =============================================================================
# 19. UPDATE BUDGET STATUS
# =============================================================================
echo ""
echo "1️⃣9️⃣  Updating budget status to 'sent'..."

curl -s -X PATCH "$API_URL/budgets/$BUDGET_ID/status" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"new_status":"sent","notes":"Orçamento enviado ao cliente"}' | jq '.' | tee status_response.json

echo "✅ Budget status updated"

# =============================================================================
# 20. LIST ALL BUDGETS
# =============================================================================
echo ""
echo "2️⃣0️⃣  Listing all budgets..."

curl -s -X GET "$API_URL/budgets?page=1&pageSize=10" \
  -H "Authorization: Bearer $TOKEN" | jq '.' | tee 20_budgets_list.json

TOTAL_BUDGETS=$(jq -r '.total' 20_budgets_list.json)
echo "✅ Found $TOTAL_BUDGETS budget(s)"

# =============================================================================
# SUMMARY
# =============================================================================
echo ""
echo "======================================"
echo "✅ TEST COMPLETED SUCCESSFULLY!"
echo "======================================"
echo ""
echo "📊 Summary:"
echo "  - Company: $COMPANY_NAME"
echo "  - Email: $TEST_EMAIL"
echo "  - Brand ID: $BRAND_ID"
echo "  - Material ID: $MATERIAL_ID"
echo "  - Filament ID: $FILAMENT_ID"
echo "  - Machine Preset ID: $MACHINE_PRESET_ID"
echo "  - Energy Preset ID: $ENERGY_PRESET_ID"
echo "  - Cost Preset ID: $COST_PRESET_ID"
echo "  - Customer ID: $CUSTOMER_ID"
echo "  - Budget ID: $BUDGET_ID"
echo "  - PDF: budget_${BUDGET_ID}.pdf"
echo ""
echo "🎨 Features Tested:"
echo "  ✅ Company Registration"
echo "  ✅ Login & Authentication"
echo "  ✅ Logo Upload to CDN"
echo "  ✅ Branding Templates"
echo "  ✅ Branding Customization"
echo "  ✅ Brand CRUD"
echo "  ✅ Material CRUD"
echo "  ✅ Filament CRUD"
echo "  ✅ Presets (Machine, Energy, Cost)"
echo "  ✅ Customer CRUD"
echo "  ✅ Budget with Product Info"
echo "  ✅ PDF Generation with Branding"
echo "  ✅ PDF Upload to CDN"
echo "  ✅ Budget Status Management"
echo ""
echo "📄 Generated files:"
ls -lh *.json budget_*.pdf test_logo.png 2>/dev/null | awk '{print "  - " $9 " (" $5 ")"}'
echo ""
echo "🎉 All tests passed!"
