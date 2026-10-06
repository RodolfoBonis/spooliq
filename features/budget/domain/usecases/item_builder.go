package usecases

import (
	"time"

	"github.com/RodolfoBonis/spooliq/features/budget/domain/entities"
	"github.com/google/uuid"
)

// builtBudgetItem groups a budget item entity with the filament entities that
// belong to it, both fully populated and ready to be persisted.
type builtBudgetItem struct {
	Item      *entities.BudgetItemEntity
	Filaments []*entities.BudgetItemFilamentEntity
}

// buildBudgetItems constructs budget item entities (and their filament entities)
// from request items, ensuring the fields that create/update/duplicate used to
// set inconsistently are always populated:
//
//   - OrganizationID on both the item and each filament (multi-tenancy),
//   - BudgetID on the item,
//   - the legacy primary FilamentID / Quantity columns on the item (kept
//     populated for backward compatibility with the NOT NULL budget_items
//     columns — the first filament is used as the primary, Quantity is the sum
//     of all filament grams).
//
// Callers are expected to validate filament ownership before persisting.
func buildBudgetItems(budgetID uuid.UUID, organizationID string, reqs []entities.BudgetItemRequest) []builtBudgetItem {
	now := time.Now()
	built := make([]builtBudgetItem, 0, len(reqs))

	for _, req := range reqs {
		itemID := uuid.New()

		var primaryFilamentID uuid.UUID
		var totalGrams float64
		filaments := make([]*entities.BudgetItemFilamentEntity, 0, len(req.Filaments))

		for i, filReq := range req.Filaments {
			if i == 0 {
				primaryFilamentID = filReq.FilamentID
			}
			totalGrams += filReq.Quantity
			filaments = append(filaments, &entities.BudgetItemFilamentEntity{
				ID:             uuid.New(),
				BudgetItemID:   itemID,
				FilamentID:     filReq.FilamentID,
				OrganizationID: organizationID,
				Quantity:       filReq.Quantity,
				Order:          filReq.Order,
				CreatedAt:      now,
				UpdatedAt:      now,
			})
		}

		item := &entities.BudgetItemEntity{
			ID:                      itemID,
			BudgetID:                budgetID,
			FilamentID:              primaryFilamentID,
			OrganizationID:          organizationID,
			Quantity:                totalGrams,
			Order:                   req.Order,
			ProductName:             req.ProductName,
			ProductDescription:      req.ProductDescription,
			ProductQuantity:         req.ProductQuantity,
			ProductDimensions:       req.ProductDimensions,
			PrintTimeHours:          req.PrintTimeHours,
			PrintTimeMinutes:        req.PrintTimeMinutes,
			SetupTimeMinutes:        req.SetupTimeMinutes,
			ManualLaborMinutesTotal: req.ManualLaborMinutesTotal,
			CostPresetID:            req.CostPresetID,
			AdditionalNotes:         req.AdditionalNotes,
			Model3DID:               req.Model3DID,
			CreatedAt:               now,
			UpdatedAt:               now,
		}

		built = append(built, builtBudgetItem{Item: item, Filaments: filaments})
	}

	return built
}

// collectFilamentIDs returns every filament ID referenced by the request items.
func collectFilamentIDs(reqs []entities.BudgetItemRequest) []uuid.UUID {
	ids := make([]uuid.UUID, 0)
	for _, req := range reqs {
		for _, fil := range req.Filaments {
			ids = append(ids, fil.FilamentID)
		}
	}
	return ids
}

// collectModel3DIDs returns every distinct, non-nil 3D model ID referenced by the
// request items, so references can be validated against the organization in one go.
func collectModel3DIDs(reqs []entities.BudgetItemRequest) []uuid.UUID {
	seen := make(map[uuid.UUID]struct{})
	ids := make([]uuid.UUID, 0)
	for _, req := range reqs {
		if req.Model3DID == nil {
			continue
		}
		if _, ok := seen[*req.Model3DID]; ok {
			continue
		}
		seen[*req.Model3DID] = struct{}{}
		ids = append(ids, *req.Model3DID)
	}
	return ids
}
