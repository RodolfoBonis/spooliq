# Teste do Novo Cálculo de Orçamento

## 🎯 Cenário de Teste: 16 Mini Anjos

Este documento explica como testar o novo cálculo de custos implementado.

## 📊 Exemplo Real

**Produto**: 16 mini anjos decorativos
**Tempo de Impressão**: 30 horas
**Filamento**: ~480g de PLA (30g por anjo)

### Configuração Recomendada

Antes de criar o orçamento, configure um CostPreset com:
- `labor_cost_per_hour`: 25 (R$ 25/hora)
- `overhead_percentage`: 10 (10% de overhead)
- `profit_margin_percentage`: 20 (20% de margem de lucro)

### Tempos de Trabalho Manual

1. **Setup Time (tempo de preparação)**:
   - Preparar arquivo 3MF
   - Configurar slicer
   - Carregar filamento
   - Iniciar impressão
   - **Sugestão**: 20 minutos

2. **Manual Labor Time (trabalho manual total)**:
   - Remover da mesa: 16 × 2min = 32min
   - Remover suportes: 16 × 5min = 80min
   - Lixar/acabamento: 16 × 2min = 32min
   - Embalar: 16 × 1min = 16min
   - **Total**: 160 minutos (2h40min)

## 🧮 Cálculo Esperado

Assumindo:
- Filamento PLA: R$ 50/kg = 5000 centavos/kg
- Energia: R$ 0,80/kWh, máquina 250W
- Taxa de mão de obra: R$ 25/h = 2500 centavos/h
- Overhead: 10%
- Lucro: 20%

### Breakdown por Item:

```
Filamento: (480g ÷ 1000) × 5000 = 2400 centavos = R$ 24,00
Energia: (250W × 30h ÷ 1000) × 0,80 × 100 = 6000 centavos = R$ 60,00
Setup: (20min ÷ 60) × 2500 = 833 centavos = R$ 8,33
Manual Labor: (160min ÷ 60) × 2500 = 6667 centavos = R$ 66,67

Subtotal Item: 2400 + 6000 + 833 + 6667 = 15900 centavos = R$ 159,00
```

### Totais do Orçamento:

```
Subtotal (sem overhead/lucro): R$ 159,00
Overhead (10%): R$ 15,90
Base + Overhead: R$ 174,90
Profit (20%): R$ 34,98

TOTAL FINAL: R$ 209,88
```

**Preço por unidade**: R$ 209,88 ÷ 16 = R$ 13,12 por anjo

## 📝 Como Testar

### 1. Preparar Dados

Primeiro, obtenha os UUIDs necessários:

```bash
# Obter um customer_id
curl -X GET "http://localhost:8080/api/v1/customers" \
  -H "Authorization: Bearer SEU_TOKEN"

# Obter preset IDs
curl -X GET "http://localhost:8080/api/v1/presets/machine" \
  -H "Authorization: Bearer SEU_TOKEN"

curl -X GET "http://localhost:8080/api/v1/presets/energy" \
  -H "Authorization: Bearer SEU_TOKEN"

curl -X GET "http://localhost:8080/api/v1/presets/cost" \
  -H "Authorization: Bearer SEU_TOKEN"

# Obter filament_id
curl -X GET "http://localhost:8080/api/v1/filaments" \
  -H "Authorization: Bearer SEU_TOKEN"
```

### 2. Editar o Arquivo de Teste

Edite `test_budget_16_anjos.json` e substitua todos os `SUBSTITUA_COM_UUID_*` pelos UUIDs reais.

### 3. Criar o Orçamento

```bash
curl -X POST "http://localhost:8080/api/v1/budgets" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer SEU_TOKEN" \
  -d @test_budget_16_anjos.json \
  | jq '.'
```

### 4. Verificar o Resultado

A resposta deve conter:

```json
{
  "items": [
    {
      "product_name": "Mini Anjo Decorativo",
      "product_quantity": 16,
      "setup_time_minutes": 20,
      "manual_labor_minutes_total": 160,
      "filament_cost": 2400,
      "energy_cost": 6000,
      "setup_cost": 833,
      "manual_labor_cost": 6667,
      "item_total_cost": 15900,
      "unit_price": 994
    }
  ],
  "filament_cost": 2400,
  "energy_cost": 6000,
  "setup_cost": 833,
  "labor_cost": 6667,
  "overhead_cost": 1590,
  "profit_amount": 3498,
  "total_cost": 20988
}
```

## ✅ Validação

Compare com o cálculo antigo:
- **Antes**: 30h × R$ 25 = R$ 750 de mão de obra = **R$ 834 total** (absurdo!)
- **Agora**: R$ 8,33 setup + R$ 66,67 manual = R$ 75 de mão de obra = **R$ 209,88 total** (realista!)

## 🎉 Diferença

**Economia de 74,8%** no custo de mão de obra!

Agora o cálculo reflete a realidade:
- ✅ Setup pago uma vez por job
- ✅ Trabalho manual proporcional às peças
- ✅ Overhead e lucro aplicados corretamente
- ✅ Tempo de impressão NÃO conta como mão de obra

## 🔧 Ajustes Finos

Você pode ajustar os tempos conforme sua operação:

- **Setup mais rápido?** Reduza `setup_time_minutes` para 15min
- **Acabamento mais elaborado?** Aumente `manual_labor_minutes_total` para 200min
- **Mais overhead?** Aumente `overhead_percentage` no CostPreset
- **Margem maior?** Aumente `profit_margin_percentage` no CostPreset

## 📊 Comparação Visual

| Componente | Cálculo Antigo | Cálculo Novo | Diferença |
|-----------|---------------|--------------|-----------|
| Filamento | R$ 24,00 | R$ 24,00 | - |
| Energia | R$ 60,00 | R$ 60,00 | - |
| Mão de Obra | **R$ 750,00** | **R$ 75,00** | **-90%** |
| Overhead | R$ 0 | R$ 15,90 | +R$ 15,90 |
| Lucro | R$ 0 | R$ 34,98 | +R$ 34,98 |
| **TOTAL** | **R$ 834,00** | **R$ 209,88** | **-74,8%** |

Agora seus orçamentos serão competitivos e realistas! 🚀
