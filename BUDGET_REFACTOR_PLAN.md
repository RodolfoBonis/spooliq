# Refatoração Completa do Módulo de Orçamento

## 🎯 Visão Geral

Refatorar o módulo de orçamento para que:
- **Items = Produtos** (o que o cliente vê e compra)
- **Filamentos estão dentro dos items** (1 produto pode usar N filamentos)
- **Quantidade do item = UNIDADES** (ex: 100 chaveiros)
- **Quantidade de filamento = GRAMAS TOTAL** ⭐ **IMPORTANTE**: não é por unidade!
  - Exemplo: Para imprimir 100 chaveiros → informar 2800g total (não 28g × 100)
  - Motivo: Impressão em lote economiza filamento (menos desperdício, melhor aproveitamento)
- **Tempo de impressão = POR ITEM** (cada item tem seu próprio tempo)
- **Tempo total do orçamento = SOMA dos tempos dos items**

### 💡 Por Que "Quantidade Total" e Não "Por Unidade"?

Na prática, ao imprimir **em lote**:
- ✅ Há economia de filamento (menos purge, menos waste)
- ✅ Melhor aproveitamento do espaço da impressora
- ✅ Otimização de camadas e trajetórias
- ✅ Não há linearidade: 200 unidades ≠ 2× filamento de 100 unidades

Por isso, o usuário informa **quanto filamento vai gastar no total** para aquela quantidade específica.

---

## 📊 Estrutura de Dados Atual vs Nova

### ❌ ANTES (Incorreto)
```
budgets
  - print_time_hours (global)
  - print_time_minutes (global)

budget_items
  - filament_id (1:1) ❌
  - quantity (gramas) ❌
  - product_name
  - product_quantity
  - unit_price (estático)
```

### ✅ DEPOIS (Correto)
```
budgets
  ❌ REMOVER: print_time_hours, print_time_minutes
  ✅ CALCULAR: total_print_time (soma dos items)

budget_items (PRODUTOS)
  - product_name
  - product_quantity (UNIDADES)
  - product_dimensions
  - print_time_hours (deste item)
  - print_time_minutes (deste item)
  - cost_preset_id (preset específico)
  - additional_labor_cost
  - unit_price (CALCULADO)
  ❌ REMOVER: filament_id

budget_item_filaments (NOVA TABELA)
  - budget_item_id
  - filament_id
  - quantity_per_unit (gramas por unidade)
  - order (ordem de aplicação/cor)
```

---

## 🗂️ Fase 1: Database Schema Changes

### 1.1 Criar Nova Tabela: `budget_item_filaments`

**Arquivo:** `features/budget/data/models/budget_item_filament_model.go`

```go
package models

import (
	"time"
	"github.com/google/uuid"
)

type BudgetItemFilamentModel struct {
	ID              uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	BudgetItemID    uuid.UUID `gorm:"type:uuid;not null;index" json:"budget_item_id"`
	FilamentID      uuid.UUID `gorm:"type:uuid;not null;index" json:"filament_id"`
	OrganizationID  string    `gorm:"type:varchar(255);not null;index" json:"organization_id"`
	
	// Quantidade TOTAL de filamento para este item (não por unidade!)
	// Exemplo: Para imprimir 100 chaveiros em lote, usar 2800g de PLA Rosa
	// (economias de escala, menos desperdício, melhor aproveitamento)
	Quantity float64 `gorm:"type:numeric;not null" json:"quantity"` // gramas TOTAL
	
	// Ordem de aplicação (para AMS/multi-cor)
	Order int `gorm:"type:integer;not null;default:1" json:"order"`
	
	CreatedAt time.Time `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt time.Time `gorm:"autoUpdateTime" json:"updated_at"`
}

func (BudgetItemFilamentModel) TableName() string {
	return "budget_item_filaments"
}

// ToEntity converts model to entity
func (m *BudgetItemFilamentModel) ToEntity() *entities.BudgetItemFilamentEntity {
	return &entities.BudgetItemFilamentEntity{
		ID:              m.ID,
		BudgetItemID:    m.BudgetItemID,
		FilamentID:      m.FilamentID,
		QuantityPerUnit: m.QuantityPerUnit,
		Order:           m.Order,
		CreatedAt:       m.CreatedAt,
		UpdatedAt:       m.UpdatedAt,
	}
}

// FromEntity converts entity to model
func (m *BudgetItemFilamentModel) FromEntity(e *entities.BudgetItemFilamentEntity) {
	m.ID = e.ID
	m.BudgetItemID = e.BudgetItemID
	m.FilamentID = e.FilamentID
	m.QuantityPerUnit = e.QuantityPerUnit
	m.Order = e.Order
	m.CreatedAt = e.CreatedAt
	m.UpdatedAt = e.UpdatedAt
}
```

### 1.2 Modificar Tabela: `budget_items`

**Arquivo:** `features/budget/data/models/budget_item_model.go`

```go
type BudgetItemModel struct {
	ID             uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	BudgetID       uuid.UUID `gorm:"type:uuid;not null;index" json:"budget_id"`
	OrganizationID string    `gorm:"type:varchar(255);not null;index" json:"organization_id"`
	
	// ========================================
	// PRODUTO (o que o cliente vê)
	// ========================================
	ProductName        string  `gorm:"type:varchar(255);not null" json:"product_name"`
	ProductDescription *string `gorm:"type:text" json:"product_description,omitempty"`
	ProductQuantity    int     `gorm:"type:integer;not null" json:"product_quantity"` // unidades
	ProductDimensions  *string `gorm:"type:varchar(100)" json:"product_dimensions,omitempty"`
	
	// ========================================
	// TEMPO DE IMPRESSÃO (deste item)
	// ========================================
	PrintTimeHours   int `gorm:"type:integer;not null;default:0" json:"print_time_hours"`
	PrintTimeMinutes int `gorm:"type:integer;not null;default:0" json:"print_time_minutes"`
	
	// ========================================
	// CUSTOS ADICIONAIS (específicos do item)
	// ========================================
	CostPresetID        *uuid.UUID `gorm:"type:uuid" json:"cost_preset_id,omitempty"`
	AdditionalLaborCost *int64     `gorm:"type:bigint" json:"additional_labor_cost,omitempty"` // centavos
	AdditionalNotes     *string    `gorm:"type:text" json:"additional_notes,omitempty"`
	
	// ========================================
	// CUSTOS CALCULADOS (deste item)
	// ========================================
	FilamentCost  int64 `gorm:"type:bigint;default:0" json:"filament_cost"`  // centavos
	WasteCost     int64 `gorm:"type:bigint;default:0" json:"waste_cost"`     // centavos
	EnergyCost    int64 `gorm:"type:bigint;default:0" json:"energy_cost"`    // centavos
	LaborCost     int64 `gorm:"type:bigint;default:0" json:"labor_cost"`     // centavos
	ItemTotalCost int64 `gorm:"type:bigint;default:0" json:"item_total_cost"` // centavos
	
	// Valor unitário final (ItemTotalCost ÷ ProductQuantity)
	UnitPrice int64 `gorm:"type:bigint;default:0" json:"unit_price"` // centavos por unidade
	
	// Ordem de impressão (opcional)
	Order int `gorm:"type:integer;default:0" json:"order"`
	
	// ========================================
	// CAMPOS REMOVIDOS (Migration DROP)
	// ========================================
	// ❌ FilamentID    - movido para budget_item_filaments
	// ❌ Quantity      - substituído por QuantityPerUnit em budget_item_filaments
	// ❌ WasteAmount   - calculado dinamicamente
	// ❌ ItemCost      - renomeado para ItemTotalCost
	
	CreatedAt time.Time `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt time.Time `gorm:"autoUpdateTime" json:"updated_at"`
}
```

### 1.3 Modificar Tabela: `budgets`

**Arquivo:** `features/budget/data/models/budget_model.go`

```go
type BudgetModel struct {
	ID             uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	OrganizationID string    `gorm:"type:varchar(255);not null;index" json:"organization_id"`
	Name           string    `gorm:"type:varchar(255);not null" json:"name"`
	Description    string    `gorm:"type:text" json:"description,omitempty"`
	CustomerID     uuid.UUID `gorm:"type:uuid;not null;index" json:"customer_id"`
	Status         string    `gorm:"type:varchar(50);not null;default:'draft'" json:"status"`
	
	// ========================================
	// PRESETS GLOBAIS
	// ========================================
	MachinePresetID *uuid.UUID `gorm:"type:uuid" json:"machine_preset_id,omitempty"`
	EnergyPresetID  *uuid.UUID `gorm:"type:uuid" json:"energy_preset_id,omitempty"`
	
	// ========================================
	// FLAGS DE INCLUSÃO
	// ========================================
	IncludeEnergyCost bool `gorm:"type:boolean;default:true" json:"include_energy_cost"`
	IncludeWasteCost  bool `gorm:"type:boolean;default:true" json:"include_waste_cost"`
	
	// ========================================
	// CUSTOS CALCULADOS (soma de todos os items)
	// ========================================
	FilamentCost int64 `gorm:"type:bigint;default:0" json:"filament_cost"` // centavos
	WasteCost    int64 `gorm:"type:bigint;default:0" json:"waste_cost"`    // centavos
	EnergyCost   int64 `gorm:"type:bigint;default:0" json:"energy_cost"`   // centavos
	LaborCost    int64 `gorm:"type:bigint;default:0" json:"labor_cost"`    // centavos
	TotalCost    int64 `gorm:"type:bigint;default:0" json:"total_cost"`    // centavos
	
	// ========================================
	// INFORMAÇÕES COMERCIAIS
	// ========================================
	DeliveryDays  int     `gorm:"type:integer" json:"delivery_days"`
	PaymentTerms  string  `gorm:"type:text" json:"payment_terms,omitempty"`
	Notes         string  `gorm:"type:text" json:"notes,omitempty"`
	PDFUrl        *string `gorm:"type:varchar(500)" json:"pdf_url,omitempty"`
	OwnerUserID   string  `gorm:"type:varchar(255);not null" json:"owner_user_id"`
	
	// ========================================
	// CAMPOS REMOVIDOS (Migration DROP)
	// ========================================
	// ❌ PrintTimeHours       - calculado dinamicamente (soma dos items)
	// ❌ PrintTimeMinutes     - calculado dinamicamente (soma dos items)
	// ❌ LaborCostPerHour     - movido para items (via preset ou adicional)
	// ❌ IncludeLaborCost     - implícito (se item tem labor cost, inclui)
	// ❌ CostPresetID         - movido para item level
	
	CreatedAt time.Time `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt time.Time `gorm:"autoUpdateTime" json:"updated_at"`
}
```

### 1.4 Migration SQL

**Arquivo:** `scripts/migrations/budget_refactor_migration.sql`

```sql
-- ================================================
-- MIGRATION: Budget Refactor - Multi-Filament Items
-- ================================================

BEGIN;

-- 1. Criar nova tabela budget_item_filaments
CREATE TABLE IF NOT EXISTS budget_item_filaments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    budget_item_id UUID NOT NULL REFERENCES budget_items(id) ON DELETE CASCADE,
    filament_id UUID NOT NULL REFERENCES filaments(id),
    organization_id VARCHAR(255) NOT NULL,
    
    quantity NUMERIC NOT NULL,  -- gramas TOTAL para este item
    "order" INTEGER NOT NULL DEFAULT 1,
    
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_budget_item_filaments_item ON budget_item_filaments(budget_item_id);
CREATE INDEX idx_budget_item_filaments_filament ON budget_item_filaments(filament_id);
CREATE INDEX idx_budget_item_filaments_org ON budget_item_filaments(organization_id);

-- 2. Adicionar novos campos em budget_items
ALTER TABLE budget_items 
    ADD COLUMN IF NOT EXISTS print_time_hours INTEGER NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS print_time_minutes INTEGER NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS cost_preset_id UUID REFERENCES presets(id),
    ADD COLUMN IF NOT EXISTS additional_labor_cost BIGINT DEFAULT 0,
    ADD COLUMN IF NOT EXISTS additional_notes TEXT,
    ADD COLUMN IF NOT EXISTS energy_cost BIGINT DEFAULT 0,
    ADD COLUMN IF NOT EXISTS labor_cost BIGINT DEFAULT 0,
    ADD COLUMN IF NOT EXISTS item_total_cost BIGINT DEFAULT 0;

-- 3. Renomear campos em budget_items
ALTER TABLE budget_items RENAME COLUMN item_cost TO filament_cost;

-- 4. Migrar dados existentes (se houver)
-- Copiar filament_id + quantity para budget_item_filaments
INSERT INTO budget_item_filaments (budget_item_id, filament_id, organization_id, quantity, "order")
SELECT 
    id,
    filament_id,
    organization_id,
    quantity,  -- quantidade TOTAL (não por unidade)
    "order"
FROM budget_items
WHERE filament_id IS NOT NULL;

-- 5. Remover campos antigos de budget_items
ALTER TABLE budget_items 
    DROP COLUMN IF EXISTS filament_id,
    DROP COLUMN IF EXISTS quantity,
    DROP COLUMN IF EXISTS waste_amount;

-- 6. Remover campos antigos de budgets
ALTER TABLE budgets 
    DROP COLUMN IF EXISTS print_time_hours,
    DROP COLUMN IF EXISTS print_time_minutes,
    DROP COLUMN IF EXISTS labor_cost_per_hour,
    DROP COLUMN IF EXISTS include_labor_cost,
    DROP COLUMN IF EXISTS cost_preset_id;

COMMIT;
```

### 1.5 Adicionar AutoMigrate

**Arquivo:** `core/services/database_service.go`

```go
// Adicionar BudgetItemFilamentModel à lista de migrations
func (s *DatabaseService) RunMigrations() error {
	return s.db.AutoMigrate(
		// ... existing models ...
		&budgetModels.BudgetItemFilamentModel{},
	)
}
```

---

## 🎨 Fase 2: Domain Entities

### 2.1 Nova Entity: `BudgetItemFilamentEntity`

**Arquivo:** `features/budget/domain/entities/budget_item_filament_entity.go`

```go
package entities

import (
	"time"
	"github.com/google/uuid"
)

type BudgetItemFilamentEntity struct {
	ID           uuid.UUID `json:"id"`
	BudgetItemID uuid.UUID `json:"budget_item_id"`
	FilamentID   uuid.UUID `json:"filament_id"`
	Quantity     float64   `json:"quantity"` // gramas TOTAL para este item
	Order        int       `json:"order"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}
```

### 2.2 Atualizar Entity: `BudgetItemEntity`

**Arquivo:** `features/budget/domain/entities/budget_item_entity.go`

```go
package entities

import (
	"time"
	"github.com/google/uuid"
)

type BudgetItemEntity struct {
	ID             uuid.UUID `json:"id"`
	BudgetID       uuid.UUID `json:"budget_id"`
	
	// Produto
	ProductName        string  `json:"product_name"`
	ProductDescription *string `json:"product_description,omitempty"`
	ProductQuantity    int     `json:"product_quantity"` // unidades
	ProductDimensions  *string `json:"product_dimensions,omitempty"`
	
	// Tempo de impressão
	PrintTimeHours   int `json:"print_time_hours"`
	PrintTimeMinutes int `json:"print_time_minutes"`
	
	// Custos adicionais
	CostPresetID        *uuid.UUID `json:"cost_preset_id,omitempty"`
	AdditionalLaborCost *int64     `json:"additional_labor_cost,omitempty"`
	AdditionalNotes     *string    `json:"additional_notes,omitempty"`
	
	// Custos calculados
	FilamentCost  int64 `json:"filament_cost"`
	WasteCost     int64 `json:"waste_cost"`
	EnergyCost    int64 `json:"energy_cost"`
	LaborCost     int64 `json:"labor_cost"`
	ItemTotalCost int64 `json:"item_total_cost"`
	UnitPrice     int64 `json:"unit_price"`
	
	Order int `json:"order"`
	
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
```

### 2.3 Atualizar Entity: `BudgetEntity`

**Arquivo:** `features/budget/domain/entities/budget_entity.go`

```go
type BudgetEntity struct {
	ID             uuid.UUID `json:"id"`
	OrganizationID string    `json:"organization_id"`
	Name           string    `json:"name"`
	Description    string    `json:"description,omitempty"`
	CustomerID     uuid.UUID `json:"customer_id"`
	Status         BudgetStatus `json:"status"`
	
	// Presets globais
	MachinePresetID *uuid.UUID `json:"machine_preset_id,omitempty"`
	EnergyPresetID  *uuid.UUID `json:"energy_preset_id,omitempty"`
	
	// Flags
	IncludeEnergyCost bool `json:"include_energy_cost"`
	IncludeWasteCost  bool `json:"include_waste_cost"`
	
	// Custos (calculados - soma dos items)
	FilamentCost int64 `json:"filament_cost"`
	WasteCost    int64 `json:"waste_cost"`
	EnergyCost   int64 `json:"energy_cost"`
	LaborCost    int64 `json:"labor_cost"`
	TotalCost    int64 `json:"total_cost"`
	
	// Informações comerciais
	DeliveryDays  int     `json:"delivery_days"`
	PaymentTerms  string  `json:"payment_terms,omitempty"`
	Notes         string  `json:"notes,omitempty"`
	PDFUrl        *string `json:"pdf_url,omitempty"`
	OwnerUserID   string  `json:"owner_user_id"`
	
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// Helper: calcula tempo total de impressão (soma dos items)
func (b *BudgetEntity) GetTotalPrintTime(items []*BudgetItemEntity) (hours int, minutes int) {
	totalMinutes := 0
	for _, item := range items {
		totalMinutes += (item.PrintTimeHours * 60) + item.PrintTimeMinutes
	}
	return totalMinutes / 60, totalMinutes % 60
}
```

---

## 📥 Fase 3: Request/Response Entities

### 3.1 Request: `CreateBudgetRequest`

**Arquivo:** `features/budget/domain/entities/budget_request_entity.go`

```go
type BudgetItemFilamentRequest struct {
	FilamentID uuid.UUID `json:"filament_id" validate:"required"`
	Quantity   float64   `json:"quantity" validate:"required,gt=0"` // gramas TOTAL
	Order      int       `json:"order" validate:"gte=1"`
}

type BudgetItemRequest struct {
	// Produto
	ProductName        string  `json:"product_name" validate:"required,min=1,max=255"`
	ProductDescription *string `json:"product_description,omitempty" validate:"omitempty,max=1000"`
	ProductQuantity    int     `json:"product_quantity" validate:"required,gt=0"`
	ProductDimensions  *string `json:"product_dimensions,omitempty" validate:"omitempty,max=100"`
	
	// Tempo de impressão
	PrintTimeHours   int `json:"print_time_hours" validate:"gte=0"`
	PrintTimeMinutes int `json:"print_time_minutes" validate:"gte=0,lt=60"`
	
	// Custos adicionais
	CostPresetID        *uuid.UUID `json:"cost_preset_id,omitempty"`
	AdditionalLaborCost *int64     `json:"additional_labor_cost,omitempty" validate:"omitempty,gte=0"`
	AdditionalNotes     *string    `json:"additional_notes,omitempty" validate:"omitempty,max=500"`
	
	// Filamentos deste item
	Filaments []BudgetItemFilamentRequest `json:"filaments" validate:"required,min=1,dive"`
	
	Order int `json:"order" validate:"gte=0"`
}

type CreateBudgetRequest struct {
	Name        string `json:"name" validate:"required,min=1,max=255"`
	Description string `json:"description,omitempty" validate:"max=1000"`
	CustomerID  uuid.UUID `json:"customer_id" validate:"required"`
	
	// Presets globais
	MachinePresetID *uuid.UUID `json:"machine_preset_id,omitempty"`
	EnergyPresetID  *uuid.UUID `json:"energy_preset_id,omitempty"`
	
	// Flags
	IncludeEnergyCost bool `json:"include_energy_cost"`
	IncludeWasteCost  bool `json:"include_waste_cost"`
	
	// Informações comerciais
	DeliveryDays  int    `json:"delivery_days" validate:"gte=0"`
	PaymentTerms  string `json:"payment_terms,omitempty" validate:"max=500"`
	Notes         string `json:"notes,omitempty" validate:"max=1000"`
	
	// Items (produtos)
	Items []BudgetItemRequest `json:"items" validate:"required,min=1,dive"`
}
```

### 3.2 Response: `BudgetResponse`

**Arquivo:** `features/budget/domain/entities/budget_response_entity.go`

```go
type FilamentUsageInfo struct {
	FilamentID   string  `json:"filament_id"`
	FilamentName string  `json:"filament_name"`
	BrandName    string  `json:"brand_name"`
	MaterialName string  `json:"material_name"`
	Color        string  `json:"color"`
	Quantity     float64 `json:"quantity"` // gramas TOTAL para este item
	Cost         int64   `json:"cost"`     // centavos
	Order        int     `json:"order"`
}

type BudgetItemResponse struct {
	ID                 string  `json:"id"`
	BudgetID           string  `json:"budget_id"`
	ProductName        string  `json:"product_name"`
	ProductDescription *string `json:"product_description,omitempty"`
	ProductQuantity    int     `json:"product_quantity"`
	ProductDimensions  *string `json:"product_dimensions,omitempty"`
	
	// Tempo
	PrintTimeHours   int    `json:"print_time_hours"`
	PrintTimeMinutes int    `json:"print_time_minutes"`
	PrintTimeDisplay string `json:"print_time_display"` // "5h30m"
	
	// Custos adicionais
	CostPresetID        *string `json:"cost_preset_id,omitempty"`
	AdditionalLaborCost *int64  `json:"additional_labor_cost,omitempty"`
	AdditionalNotes     *string `json:"additional_notes,omitempty"`
	
	// Custos calculados
	FilamentCost  int64 `json:"filament_cost"`
	WasteCost     int64 `json:"waste_cost"`
	EnergyCost    int64 `json:"energy_cost"`
	LaborCost     int64 `json:"labor_cost"`
	ItemTotalCost int64 `json:"item_total_cost"`
	UnitPrice     int64 `json:"unit_price"`
	
	// Filamentos usados
	Filaments []FilamentUsageInfo `json:"filaments"`
	
	Order int `json:"order"`
	
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type BudgetResponse struct {
	Budget              *BudgetEntity       `json:"budget"`
	Customer            *CustomerInfo       `json:"customer"`
	Items               []BudgetItemResponse `json:"items"`
	TotalPrintTimeHours int                 `json:"total_print_time_hours"`
	TotalPrintTimeMinutes int               `json:"total_print_time_minutes"`
	TotalPrintTimeDisplay string            `json:"total_print_time_display"` // "14h15m"
}
```

---

## 🔧 Fase 4: Repository Layer

### 4.1 Adicionar Métodos ao Repository

**Arquivo:** `features/budget/domain/repositories/budget_repository.go`

```go
type BudgetRepository interface {
	// ... existing methods ...
	
	// Item Filament operations (NOVO)
	AddItemFilament(ctx context.Context, filament *entities.BudgetItemFilamentEntity) error
	RemoveItemFilament(ctx context.Context, filamentID uuid.UUID) error
	GetItemFilaments(ctx context.Context, itemID uuid.UUID) ([]*entities.BudgetItemFilamentEntity, error)
	DeleteAllItemFilaments(ctx context.Context, itemID uuid.UUID) error
	
	// Filament info with usage (NOVO)
	GetFilamentUsageInfo(ctx context.Context, itemID uuid.UUID) ([]FilamentUsageInfo, error)
}
```

### 4.2 Implementar Repository

**Arquivo:** `features/budget/data/repositories/budget_repository_impl.go`

```go
// AddItemFilament adds a filament to a budget item
func (r *budgetRepositoryImpl) AddItemFilament(ctx context.Context, filament *entities.BudgetItemFilamentEntity) error {
	model := &models.BudgetItemFilamentModel{}
	model.FromEntity(filament)
	
	if err := r.db.WithContext(ctx).Create(model).Error; err != nil {
		return fmt.Errorf("failed to add item filament: %w", err)
	}
	return nil
}

// RemoveItemFilament removes a filament from a budget item
func (r *budgetRepositoryImpl) RemoveItemFilament(ctx context.Context, filamentID uuid.UUID) error {
	if err := r.db.WithContext(ctx).Delete(&models.BudgetItemFilamentModel{}, "id = ?", filamentID).Error; err != nil {
		return fmt.Errorf("failed to remove item filament: %w", err)
	}
	return nil
}

// GetItemFilaments retrieves all filaments for a budget item
func (r *budgetRepositoryImpl) GetItemFilaments(ctx context.Context, itemID uuid.UUID) ([]*entities.BudgetItemFilamentEntity, error) {
	var filaments []*models.BudgetItemFilamentModel
	
	if err := r.db.WithContext(ctx).
		Where("budget_item_id = ?", itemID).
		Order("\"order\" ASC").
		Find(&filaments).Error; err != nil {
		return nil, fmt.Errorf("failed to get item filaments: %w", err)
	}
	
	entities := make([]*entities.BudgetItemFilamentEntity, len(filaments))
	for i, f := range filaments {
		entities[i] = f.ToEntity()
	}
	
	return entities, nil
}

// DeleteAllItemFilaments deletes all filaments for a budget item
func (r *budgetRepositoryImpl) DeleteAllItemFilaments(ctx context.Context, itemID uuid.UUID) error {
	if err := r.db.WithContext(ctx).
		Where("budget_item_id = ?", itemID).
		Delete(&models.BudgetItemFilamentModel{}).Error; err != nil {
		return fmt.Errorf("failed to delete item filaments: %w", err)
	}
	return nil
}

// GetFilamentUsageInfo retrieves detailed filament usage info for an item
func (r *budgetRepositoryImpl) GetFilamentUsageInfo(ctx context.Context, itemID uuid.UUID) ([]entities.FilamentUsageInfo, error) {
	// Get filaments with info
	var results []struct {
		FilamentID   uuid.UUID
		Quantity     float64
		Order        int
		FilamentName string
		BrandName    string
		MaterialName string
		Color        string
		PricePerKg   float64
	}
	
	err := r.db.WithContext(ctx).
		Table("budget_item_filaments bif").
		Select(`
			bif.filament_id,
			bif.quantity,
			bif."order",
			f.name as filament_name,
			b.name as brand_name,
			m.name as material_name,
			f.color,
			f.price_per_kg
		`).
		Joins("JOIN filaments f ON f.id = bif.filament_id").
		Joins("JOIN brands b ON b.id = f.brand_id").
		Joins("JOIN materials m ON m.id = f.material_id").
		Where("bif.budget_item_id = ?", itemID).
		Order("bif.\"order\" ASC").
		Scan(&results).Error
	
	if err != nil {
		return nil, fmt.Errorf("failed to get filament usage info: %w", err)
	}
	
	infos := make([]entities.FilamentUsageInfo, len(results))
	for i, r := range results {
		// Quantity já é o total, não precisa multiplicar!
		cost := int64((r.Quantity / 1000.0) * r.PricePerKg * 100) // to cents
		
		infos[i] = entities.FilamentUsageInfo{
			FilamentID:   r.FilamentID.String(),
			FilamentName: r.FilamentName,
			BrandName:    r.BrandName,
			MaterialName: r.MaterialName,
			Color:        r.Color,
			Quantity:     r.Quantity,
			Cost:         cost,
			Order:        r.Order,
		}
	}
	
	return infos, nil
}
```

---

## 🧮 Fase 5: Lógica de Cálculo Refatorada

### 5.1 Atualizar `CalculateCosts`

**Arquivo:** `features/budget/data/repositories/budget_repository_impl.go`

```go
func (r *budgetRepositoryImpl) CalculateCosts(ctx context.Context, budgetID uuid.UUID) error {
	// Get budget
	var budget models.BudgetModel
	if err := r.db.WithContext(ctx).First(&budget, "id = ?", budgetID).Error; err != nil {
		return fmt.Errorf("failed to get budget: %w", err)
	}
	
	// Get items
	items, err := r.GetItems(ctx, budgetID)
	if err != nil {
		return err
	}
	
	// Get machine and energy presets
	var machinePreset *entities.PresetInfo
	var energyPreset *entities.PresetInfo
	
	if budget.MachinePresetID != nil {
		machinePreset, _ = r.GetPresetInfo(ctx, *budget.MachinePresetID, "machine")
	}
	if budget.EnergyPresetID != nil {
		energyPreset, _ = r.GetPresetInfo(ctx, *budget.EnergyPresetID, "energy")
	}
	
	var totalFilamentCost, totalWasteCost, totalEnergyCost, totalLaborCost int64
	
	// Process each item
	for _, item := range items {
		var itemFilamentCost, itemWasteCost, itemEnergyCost, itemLaborCost int64
		
		// 1. Get filaments for this item
		itemFilaments, err := r.GetItemFilaments(ctx, item.ID)
		if err != nil {
			return fmt.Errorf("failed to get item filaments: %w", err)
		}
		
		// 2. Calculate filament cost
		var totalGrams float64
		var avgPrice float64
		var totalPrice float64
		
		for _, itemFil := range itemFilaments {
			filament, err := r.GetFilamentInfo(ctx, itemFil.FilamentID)
			if err != nil {
				continue
			}
			
			// Quantity já é o total (não precisa multiplicar por ProductQuantity!)
			gramsTotal := itemFil.Quantity
			cost := (gramsTotal / 1000.0) * filament.PricePerKg * 100 // to cents
			itemFilamentCost += int64(cost)
			totalGrams += gramsTotal
			totalPrice += filament.PricePerKg
		}
		
		if len(itemFilaments) > 0 {
			avgPrice = totalPrice / float64(len(itemFilaments))
		}
		
		// 3. Calculate waste cost (AMS multi-color)
		if budget.IncludeWasteCost && len(itemFilaments) > 1 {
			wastePerChange := 15.0 // grams
			numChanges := len(itemFilaments) - 1
			totalWaste := wastePerChange * float64(numChanges)
			itemWasteCost = int64((totalWaste / 1000.0) * avgPrice * 100)
		}
		
		// 4. Calculate energy cost (proportional to this item's print time)
		if budget.IncludeEnergyCost && machinePreset != nil && energyPreset != nil {
			// Get power consumption and energy price from presets
			var powerConsumption, energyPrice float64
			
			r.db.WithContext(ctx).
				Table("presets").
				Select("CAST(value AS FLOAT) as price").
				Where("id = ? AND key = 'price_per_kwh'", energyPreset.ID).
				Scan(&energyPrice)
			
			r.db.WithContext(ctx).
				Table("presets").
				Select("CAST(value AS FLOAT) as power").
				Where("id = ? AND key = 'power_consumption'", machinePreset.ID).
				Scan(&powerConsumption)
			
			itemHours := float64(item.PrintTimeHours) + float64(item.PrintTimeMinutes)/60.0
			kwh := powerConsumption * itemHours / 1000.0
			itemEnergyCost = int64(kwh * energyPrice * 100)
		}
		
		// 5. Calculate labor cost (base + additional)
		itemHours := float64(item.PrintTimeHours) + float64(item.PrintTimeMinutes)/60.0
		
		// Base labor cost from preset
		var laborRate float64
		if item.CostPresetID != nil {
			costPreset, err := r.GetPresetInfo(ctx, *item.CostPresetID, "cost")
			if err == nil {
				r.db.WithContext(ctx).
					Table("presets").
					Select("CAST(value AS FLOAT) as rate").
					Where("id = ? AND key = 'labor_cost_per_hour'", costPreset.ID).
					Scan(&laborRate)
			}
		}
		
		itemLaborCost = int64(itemHours * laborRate * 100)
		
		// Add additional labor cost (pintura, acabamento, etc)
		if item.AdditionalLaborCost != nil {
			itemLaborCost += *item.AdditionalLaborCost
		}
		
		// 6. Calculate item total cost
		item.FilamentCost = itemFilamentCost
		item.WasteCost = itemWasteCost
		item.EnergyCost = itemEnergyCost
		item.LaborCost = itemLaborCost
		item.ItemTotalCost = itemFilamentCost + itemWasteCost + itemEnergyCost + itemLaborCost
		
		// 7. Calculate unit price
		if item.ProductQuantity > 0 {
			item.UnitPrice = item.ItemTotalCost / int64(item.ProductQuantity)
		}
		
		// 8. Update item in database
		if err := r.UpdateItem(ctx, item); err != nil {
			return fmt.Errorf("failed to update item costs: %w", err)
		}
		
		// 9. Sum to budget totals
		totalFilamentCost += itemFilamentCost
		totalWasteCost += itemWasteCost
		totalEnergyCost += itemEnergyCost
		totalLaborCost += itemLaborCost
	}
	
	// Update budget totals
	budget.FilamentCost = totalFilamentCost
	budget.WasteCost = totalWasteCost
	budget.EnergyCost = totalEnergyCost
	budget.LaborCost = totalLaborCost
	budget.TotalCost = totalFilamentCost + totalWasteCost + totalEnergyCost + totalLaborCost
	
	if err := r.db.WithContext(ctx).
		Model(&models.BudgetModel{}).
		Where("id = ?", budgetID).
		Updates(map[string]interface{}{
			"filament_cost": budget.FilamentCost,
			"waste_cost":    budget.WasteCost,
			"energy_cost":   budget.EnergyCost,
			"labor_cost":    budget.LaborCost,
			"total_cost":    budget.TotalCost,
		}).Error; err != nil {
		return fmt.Errorf("failed to update budget costs: %w", err)
	}
	
	return nil
}
```

---

## 🎯 Fase 6: Use Cases Refatorados

### 6.1 Criar Orçamento

**Arquivo:** `features/budget/domain/usecases/create_budget_uc.go`

```go
func (uc *CreateBudgetUseCase) Execute(c *gin.Context) {
	ctx := c.Request.Context()
	
	var request entities.CreateBudgetRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format"})
		return
	}
	
	// Validate
	validate := validator.New()
	if err := validate.Struct(request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	
	organizationID := helpers.GetOrganizationID(c)
	userID := helpers.GetUserID(c)
	
	// Check customer
	_, err := uc.customerRepository.FindByID(ctx, request.CustomerID, organizationID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Customer not found"})
		return
	}
	
	// Create budget
	budget := &entities.BudgetEntity{
		ID:                uuid.New(),
		OrganizationID:    organizationID,
		Name:              request.Name,
		Description:       request.Description,
		CustomerID:        request.CustomerID,
		Status:            entities.StatusDraft,
		MachinePresetID:   request.MachinePresetID,
		EnergyPresetID:    request.EnergyPresetID,
		IncludeEnergyCost: request.IncludeEnergyCost,
		IncludeWasteCost:  request.IncludeWasteCost,
		DeliveryDays:      request.DeliveryDays,
		PaymentTerms:      request.PaymentTerms,
		Notes:             request.Notes,
		OwnerUserID:       userID,
		CreatedAt:         time.Now(),
		UpdatedAt:         time.Now(),
	}
	
	if err := uc.budgetRepository.Create(ctx, budget); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create budget"})
		return
	}
	
	// Create items with filaments
	for _, itemReq := range request.Items {
		item := &entities.BudgetItemEntity{
			ID:                  uuid.New(),
			BudgetID:            budget.ID,
			ProductName:         itemReq.ProductName,
			ProductDescription:  itemReq.ProductDescription,
			ProductQuantity:     itemReq.ProductQuantity,
			ProductDimensions:   itemReq.ProductDimensions,
			PrintTimeHours:      itemReq.PrintTimeHours,
			PrintTimeMinutes:    itemReq.PrintTimeMinutes,
			CostPresetID:        itemReq.CostPresetID,
			AdditionalLaborCost: itemReq.AdditionalLaborCost,
			AdditionalNotes:     itemReq.AdditionalNotes,
			Order:               itemReq.Order,
			CreatedAt:           time.Now(),
			UpdatedAt:           time.Now(),
		}
		
		if err := uc.budgetRepository.AddItem(ctx, item); err != nil {
			uc.budgetRepository.Delete(ctx, budget.ID)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create item"})
			return
		}
		
		// Add filaments to item
		for _, filReq := range itemReq.Filaments {
			filament := &entities.BudgetItemFilamentEntity{
				ID:              uuid.New(),
				BudgetItemID:    item.ID,
				FilamentID:      filReq.FilamentID,
				QuantityPerUnit: filReq.QuantityPerUnit,
				Order:           filReq.Order,
				CreatedAt:       time.Now(),
				UpdatedAt:       time.Now(),
			}
			
			if err := uc.budgetRepository.AddItemFilament(ctx, filament); err != nil {
				uc.budgetRepository.Delete(ctx, budget.ID)
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to add filament"})
				return
			}
		}
	}
	
	// Calculate costs
	if err := uc.budgetRepository.CalculateCosts(ctx, budget.ID); err != nil {
		uc.logger.Error(ctx, "Failed to calculate costs", map[string]interface{}{"error": err.Error()})
	}
	
	// Build response
	response, err := uc.buildBudgetResponse(ctx, budget.ID, organizationID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to build response"})
		return
	}
	
	c.JSON(http.StatusCreated, response)
}

func (uc *CreateBudgetUseCase) buildBudgetResponse(ctx context.Context, budgetID uuid.UUID, organizationID string) (*entities.BudgetResponse, error) {
	budget, err := uc.budgetRepository.FindByID(ctx, budgetID, organizationID)
	if err != nil {
		return nil, err
	}
	
	customer, _ := uc.budgetRepository.GetCustomerInfo(ctx, budget.CustomerID)
	items, _ := uc.budgetRepository.GetItems(ctx, budgetID)
	
	itemResponses := make([]entities.BudgetItemResponse, len(items))
	var totalMinutes int
	
	for i, item := range items {
		filaments, _ := uc.budgetRepository.GetFilamentUsageInfo(ctx, item.ID)
		
		itemMinutes := (item.PrintTimeHours * 60) + item.PrintTimeMinutes
		totalMinutes += itemMinutes
		
		itemResponses[i] = entities.BudgetItemResponse{
			ID:                  item.ID.String(),
			BudgetID:            item.BudgetID.String(),
			ProductName:         item.ProductName,
			ProductDescription:  item.ProductDescription,
			ProductQuantity:     item.ProductQuantity,
			ProductDimensions:   item.ProductDimensions,
			PrintTimeHours:      item.PrintTimeHours,
			PrintTimeMinutes:    item.PrintTimeMinutes,
			PrintTimeDisplay:    fmt.Sprintf("%dh%02dm", item.PrintTimeHours, item.PrintTimeMinutes),
			CostPresetID:        ptrToStr(item.CostPresetID),
			AdditionalLaborCost: item.AdditionalLaborCost,
			AdditionalNotes:     item.AdditionalNotes,
			FilamentCost:        item.FilamentCost,
			WasteCost:           item.WasteCost,
			EnergyCost:          item.EnergyCost,
			LaborCost:           item.LaborCost,
			ItemTotalCost:       item.ItemTotalCost,
			UnitPrice:           item.UnitPrice,
			Filaments:           filaments,
			Order:               item.Order,
			CreatedAt:           item.CreatedAt,
			UpdatedAt:           item.UpdatedAt,
		}
	}
	
	totalHours := totalMinutes / 60
	totalMins := totalMinutes % 60
	
	return &entities.BudgetResponse{
		Budget:                budget,
		Customer:              customer,
		Items:                 itemResponses,
		TotalPrintTimeHours:   totalHours,
		TotalPrintTimeMinutes: totalMins,
		TotalPrintTimeDisplay: fmt.Sprintf("%dh%02dm", totalHours, totalMins),
	}, nil
}
```

---

## 📄 Fase 7: Atualizar PDF Service

### 7.1 Modificar Geração de PDF

**Arquivo:** `core/services/pdf_service.go`

```go
// Modificar addItemsTable para mostrar produtos
func (s *PDFService) addItemsTable(pdf *gofpdf.Fpdf, items []budgetEntities.BudgetItemResponse, branding *companyEntities.CompanyBrandingEntity) {
	pdf.SetFont("Arial", "B", 11)
	r, g, b := s.hexToRGB(branding.SecondaryColor)
	pdf.SetTextColor(r, g, b)
	pdf.Cell(0, 8, s.utf8ToLatin1("Itens do Orçamento"))
	pdf.Ln(6)

	// Table header
	r, g, b = s.hexToRGB(branding.TableHeaderBgColor)
	pdf.SetFillColor(r, g, b)
	r, g, b = s.hexToRGB(branding.HeaderTextColor)
	pdf.SetTextColor(r, g, b)
	pdf.SetFont("Arial", "B", 9)

	pdf.CellFormat(95, 7, s.utf8ToLatin1("Descrição"), "1", 0, "C", true, 0, "")
	pdf.CellFormat(20, 7, s.utf8ToLatin1("Qtd."), "1", 0, "C", true, 0, "")
	pdf.CellFormat(35, 7, s.utf8ToLatin1("Valor Unit. (R$)"), "1", 0, "C", true, 0, "")
	pdf.CellFormat(35, 7, s.utf8ToLatin1("Subtotal (R$)"), "1", 0, "C", true, 0, "")
	pdf.Ln(-1)

	// Table rows
	pdf.SetFont("Arial", "", 8)
	r, g, b = s.hexToRGB(branding.BodyTextColor)
	pdf.SetTextColor(r, g, b)

	for i, item := range items {
		fillColor := i%2 == 0
		if fillColor {
			r, g, b = s.hexToRGB(branding.TableRowAltBgColor)
			pdf.SetFillColor(r, g, b)
		}

		// Product description
		description := item.ProductName
		if item.ProductDimensions != nil && *item.ProductDimensions != "" {
			description += " - " + *item.ProductDimensions
		}
		
		// Add additional notes if present
		if item.AdditionalNotes != nil && *item.AdditionalNotes != "" {
			description += "\n*" + *item.AdditionalNotes
		}
		
		// Truncate if too long
		if len(description) > 70 {
			description = description[:67] + "..."
		}

		// Calculate subtotal
		subtotal := float64(item.ItemTotalCost) / 100.0

		pdf.CellFormat(95, 6, s.utf8ToLatin1(description), "1", 0, "L", fillColor, 0, "")
		pdf.CellFormat(20, 6, fmt.Sprintf("%d", item.ProductQuantity), "1", 0, "C", fillColor, 0, "")
		pdf.CellFormat(35, 6, fmt.Sprintf("%.2f", float64(item.UnitPrice)/100.0), "1", 0, "R", fillColor, 0, "")
		pdf.CellFormat(35, 6, fmt.Sprintf("%.2f", subtotal), "1", 0, "R", fillColor, 0, "")
		pdf.Ln(-1)
	}

	pdf.Ln(4)
}

// Adicionar resumo de tempo de impressão
func (s *PDFService) addPrintTimeSummary(pdf *gofpdf.Fpdf, totalHours, totalMinutes int) {
	pdf.SetFont("Arial", "", 10)
	pdf.SetTextColor(64, 64, 64)
	pdf.Cell(0, 6, s.utf8ToLatin1(fmt.Sprintf("Tempo Total de Impressão: %dh%02dm", totalHours, totalMinutes)))
	pdf.Ln(5)
}
```

---

## 📚 Fase 8: Documentação

### 8.1 Atualizar Swagger

Adicionar annotations em todos os use cases atualizados.

### 8.2 Atualizar Frontend Specs

**Arquivo:** `FRONTEND_SPECS.md`

Atualizar todos os endpoints, requests e responses para refletir a nova estrutura.

### 8.3 Atualizar Testes

**Arquivo:** `test_full_flow.sh`

Atualizar script de teste para usar a nova estrutura de items com múltiplos filamentos.

---

## 🚀 Fase 9: Plano de Execução

### Etapa 1: Preparação (Day 1)
- [ ] Revisar e aprovar este plano
- [ ] Fazer backup do banco de dados
- [ ] Criar branch `feature/budget-refactor`
- [ ] Documentar dados existentes (se houver)

### Etapa 2: Database (Day 1-2)
- [ ] Criar `BudgetItemFilamentModel`
- [ ] Atualizar `BudgetItemModel` (adicionar campos)
- [ ] Atualizar `BudgetModel` (adicionar campos)
- [ ] Criar migration SQL
- [ ] Testar migration em ambiente local
- [ ] Adicionar AutoMigrate

### Etapa 3: Entities (Day 2)
- [ ] Criar `BudgetItemFilamentEntity`
- [ ] Atualizar `BudgetItemEntity`
- [ ] Atualizar `BudgetEntity`
- [ ] Criar requests/responses atualizados
- [ ] Adicionar validations

### Etapa 4: Repository (Day 2-3)
- [ ] Adicionar métodos de `budget_item_filaments`
- [ ] Implementar `GetFilamentUsageInfo`
- [ ] Refatorar `CalculateCosts` completo
- [ ] Atualizar `GetItems` para incluir filaments
- [ ] Testes unitários de repository

### Etapa 5: Use Cases (Day 3-4)
- [ ] Refatorar `CreateBudgetUseCase`
- [ ] Refatorar `UpdateBudgetUseCase`
- [ ] Refatorar `FindByIdBudgetUseCase`
- [ ] Refatorar `FindAllBudgetUseCase`
- [ ] Refatorar `GeneratePDFUseCase`
- [ ] Atualizar `UpdateStatusUseCase`

### Etapa 6: PDF Service (Day 4)
- [ ] Atualizar `addItemsTable`
- [ ] Adicionar `addPrintTimeSummary`
- [ ] Testar geração de PDF

### Etapa 7: Testing (Day 5)
- [ ] Atualizar `test_full_flow.sh`
- [ ] Testes de integração completos
- [ ] Testar cenários:
  - Item com 1 filamento
  - Item com múltiplos filamentos
  - Item com custos adicionais
  - Cálculo de custos
  - PDF generation

### Etapa 8: Documentation (Day 5)
- [ ] Atualizar Swagger annotations
- [ ] Atualizar `FRONTEND_SPECS.md`
- [ ] Atualizar `.cursorrules-frontend`
- [ ] Criar guia de migração

### Etapa 9: Cleanup (Day 5)
- [ ] Remover código deprecated
- [ ] Remover imports não utilizados
- [ ] Run linter
- [ ] Code review

### Etapa 10: Deploy (Day 6)
- [ ] Merge para main
- [ ] Deploy em staging
- [ ] Testes em staging
- [ ] Deploy em produção
- [ ] Monitoramento

---

## ⚠️ Breaking Changes

### API Changes

1. **Request: `CreateBudgetRequest`**
   - ❌ Removido: `print_time_hours`, `print_time_minutes` (global)
   - ❌ Removido: `labor_cost_per_hour`
   - ❌ Removido: `include_labor_cost`
   - ✅ Adicionado: `items[].print_time_hours`, `items[].print_time_minutes`
   - ✅ Adicionado: `items[].filaments[]` (array de filamentos)
   - ✅ Adicionado: `items[].cost_preset_id`
   - ✅ Adicionado: `items[].additional_labor_cost`

2. **Response: `BudgetResponse`**
   - ✅ Adicionado: `total_print_time_hours`, `total_print_time_minutes`, `total_print_time_display`
   - ✅ Adicionado: `items[].filaments[]` (detalhes de cada filamento)
   - ✅ Adicionado: `items[].print_time_*`

3. **Database Schema**
   - Nova tabela: `budget_item_filaments`
   - Colunas removidas de `budgets`: `print_time_hours`, `print_time_minutes`, `labor_cost_per_hour`, etc.
   - Colunas removidas de `budget_items`: `filament_id`, `quantity`, `waste_amount`
   - Colunas adicionadas em `budget_items`: `print_time_*`, `cost_preset_id`, `additional_*`, custos individuais

---

## ⚠️ IMPORTANTE: Quantidade de Filamento

### Como Funciona

```json
{
  "product_name": "Chaveiro Rosa/Branco",
  "product_quantity": 100,  // unidades do produto
  
  "filaments": [
    {
      "filament_id": "uuid",
      "quantity": 2800.0,  // ✅ TOTAL em gramas para 100 unidades
      "order": 1
    }
  ]
}
```

### ❌ NÃO é assim:
```json
{
  "quantity_per_unit": 28.0,  // ❌ Não usa "por unidade"
  // Depois multiplica: 28g × 100 = 2800g
}
```

### ✅ É assim:
```json
{
  "quantity": 2800.0,  // ✅ Informa diretamente o total
  // Não há multiplicação! Usuário decide quanto vai gastar
}
```

### Por Quê?

- Imprimir 100 chaveiros de uma vez ≠ 100× imprimir 1 chaveiro
- Há economias de escala, otimizações, menos waste
- Usuário tem mais controle e precisão

---

## 📝 Exemplo Completo de Request/Response

### Request

```json
POST /v1/budgets
{
  "name": "Orçamento Outubro Rosa",
  "description": "Chaveiros e miniaturas personalizadas",
  "customer_id": "uuid",
  "machine_preset_id": "uuid",
  "energy_preset_id": "uuid",
  "include_energy_cost": true,
  "include_waste_cost": true,
  "delivery_days": 7,
  "payment_terms": "50% entrada, 50% na entrega",
  "notes": "Embalagem personalizada",
  "items": [
    {
      "product_name": "Chaveiro Laço Rosa/Branco",
      "product_description": "Chaveiro dupla-cor Outubro Rosa",
      "product_quantity": 100,
      "product_dimensions": "26×48×9 mm",
      "print_time_hours": 5,
      "print_time_minutes": 30,
      "filaments": [
        {
          "filament_id": "uuid-pla-rosa",
          "quantity": 2800.0,  // TOTAL para os 100 chaveiros
          "order": 1
        },
        {
          "filament_id": "uuid-pla-branco",
          "quantity": 1900.0,  // TOTAL para os 100 chaveiros
          "order": 2
        }
      ],
      "order": 1
    },
    {
      "product_name": "Miniatura Unicórnio (4 cores)",
      "product_description": "Miniatura colorida com pintura manual",
      "product_quantity": 50,
      "product_dimensions": "80×50×120 mm",
      "print_time_hours": 8,
      "print_time_minutes": 45,
      "cost_preset_id": "uuid-acabamento-premium",
      "additional_labor_cost": 15000,
      "additional_notes": "Inclui pintura manual detalhada",
      "filaments": [
        {
          "filament_id": "uuid-pla-branco",
          "quantity": 3800.0,  // TOTAL para as 50 miniaturas
          "order": 1
        },
        {
          "filament_id": "uuid-pla-rosa",
          "quantity": 950.0,  // TOTAL para as 50 miniaturas
          "order": 2
        },
        {
          "filament_id": "uuid-pla-azul",
          "quantity": 720.0,  // TOTAL para as 50 miniaturas
          "order": 3
        },
        {
          "filament_id": "uuid-pla-amarelo",
          "quantity": 480.0,  // TOTAL para as 50 miniaturas
          "order": 4
        }
      ],
      "order": 2
    }
  ]
}
```

### Response

```json
{
  "budget": {
    "id": "uuid",
    "name": "Orçamento Outubro Rosa",
    "total_cost": 116000,
    "status": "draft"
  },
  "customer": {
    "id": "uuid",
    "name": "Maria Souza"
  },
  "items": [
    {
      "id": "uuid",
      "product_name": "Chaveiro Laço Rosa/Branco",
      "product_quantity": 100,
      "print_time_hours": 5,
      "print_time_minutes": 30,
      "print_time_display": "5h30m",
      "filament_cost": 44950,
      "waste_cost": 1347,
      "energy_cost": 1935,
      "labor_cost": 2750,
      "item_total_cost": 50982,
      "unit_price": 509,
      "filaments": [
        {
          "filament_id": "uuid",
          "filament_name": "PLA Rosa Ferrari",
          "brand_name": "Creality",
          "material_name": "PLA",
          "color": "Rosa Ferrari",
          "quantity": 2800.0,  // TOTAL para os 100 chaveiros
          "cost": 26970,
          "order": 1
        },
        {
          "filament_id": "uuid",
          "filament_name": "PLA Branco",
          "brand_name": "Creality",
          "material_name": "PLA",
          "color": "Branco",
          "quantity": 1900.0,  // TOTAL para os 100 chaveiros
          "cost": 17980,
          "order": 2
        }
      ]
    },
    {
      "id": "uuid",
      "product_name": "Miniatura Unicórnio (4 cores)",
      "product_quantity": 50,
      "print_time_hours": 8,
      "print_time_minutes": 45,
      "print_time_display": "8h45m",
      "additional_labor_cost": 15000,
      "additional_notes": "Inclui pintura manual detalhada",
      "filament_cost": 35955,
      "waste_cost": 4042,
      "energy_cost": 3082,
      "labor_cost": 19375,
      "item_total_cost": 62454,
      "unit_price": 1249,
      "filaments": [...]
    }
  ],
  "total_print_time_hours": 14,
  "total_print_time_minutes": 15,
  "total_print_time_display": "14h15m"
}
```

---

## ✅ Checklist Final

- [ ] Todas as migrations criadas e testadas
- [ ] Todos os models atualizados
- [ ] Todas as entities atualizadas
- [ ] Repository completo e testado
- [ ] Use cases refatorados
- [ ] PDF service atualizado
- [ ] Swagger documentado
- [ ] Frontend specs atualizados
- [ ] Testes completos (unit + integration)
- [ ] Script `test_full_flow.sh` atualizado
- [ ] Lint passing
- [ ] Code review aprovado
- [ ] Deploy em staging OK
- [ ] Deploy em produção OK

---

## 🎯 Resultado Esperado

Após esta refatoração:

1. ✅ **Items = Produtos** (o que o cliente vê)
2. ✅ **Múltiplos filamentos por item** (suporte a multi-cor)
3. ✅ **Quantidade em unidades** (ex: 100 chaveiros)
4. ✅ **Filamentos em gramas por unidade** (ex: 30g/chaveiro)
5. ✅ **Tempo por item** (cada produto tem seu tempo)
6. ✅ **Custos distribuídos corretamente**
7. ✅ **PDF profissional** mostrando produtos e valores finais
8. ✅ **Unit price calculado** incluindo todos os custos

---

**Tempo Estimado Total: 5-6 dias de desenvolvimento**

**Pronto para começar?** 🚀

