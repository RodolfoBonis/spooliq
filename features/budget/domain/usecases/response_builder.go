package usecases

import (
	"context"
	"fmt"

	"github.com/RodolfoBonis/spooliq/features/budget/domain/entities"
	"github.com/RodolfoBonis/spooliq/features/budget/domain/repositories"
	pricing "github.com/RodolfoBonis/spooliq/features/budget/domain/services"
	"github.com/google/uuid"
)

// buildBudgetItemResponses maps stored budget items into API responses and is the
// single source of truth for the per-item SALE values (SaleTotal / SaleUnitPrice):
// it distributes the budget-wide overhead+profit markup (budgetTotal - sum of item
// costs) across items proportionally to each item's direct cost via
// pricing.DistributeMarkup, so the shares sum EXACTLY to budgetTotal. The same
// values feed both the API and the PDF.
//
// It also resolves each item's cost_preset {id, name} (cached so a repeated preset
// is looked up once) and computes the print-time totals. It returns the item
// responses plus the summed print time (hours, minutes).
func buildBudgetItemResponses(ctx context.Context, repo repositories.BudgetRepository, items []*entities.BudgetItemEntity, budgetTotal int64, organizationID string) ([]entities.BudgetItemResponse, int, int) {
	itemCosts := make([]int64, len(items))
	for i, item := range items {
		itemCosts[i] = item.ItemTotalCost
	}
	saleTotals := pricing.DistributeMarkup(itemCosts, budgetTotal)

	// Resolve cost-preset names once per distinct preset.
	presetNameCache := make(map[uuid.UUID]*entities.CostPresetRef)
	resolveCostPreset := func(id *uuid.UUID) *entities.CostPresetRef {
		if id == nil {
			return nil
		}
		if ref, ok := presetNameCache[*id]; ok {
			return ref
		}
		ref := &entities.CostPresetRef{ID: id.String()}
		if info, err := repo.GetPresetInfo(ctx, *id, "cost", organizationID); err == nil && info != nil {
			ref.Name = info.Name
		}
		presetNameCache[*id] = ref
		return ref
	}

	responses := make([]entities.BudgetItemResponse, len(items))
	var totalPrintMinutes int

	for i, item := range items {
		filaments, _ := repo.GetFilamentUsageInfo(ctx, item.ID, organizationID)

		printTimeDisplay := ""
		if item.PrintTimeHours > 0 {
			printTimeDisplay = fmt.Sprintf("%dh%02dm", item.PrintTimeHours, item.PrintTimeMinutes)
		} else {
			printTimeDisplay = fmt.Sprintf("%dm", item.PrintTimeMinutes)
		}
		totalPrintMinutes += (item.PrintTimeHours * 60) + item.PrintTimeMinutes

		var costPresetIDStr *string
		if item.CostPresetID != nil {
			s := item.CostPresetID.String()
			costPresetIDStr = &s
		}

		saleTotal := saleTotals[i]
		responses[i] = entities.BudgetItemResponse{
			ID:                      item.ID.String(),
			BudgetID:                item.BudgetID.String(),
			ProductName:             item.ProductName,
			ProductDescription:      item.ProductDescription,
			ProductQuantity:         item.ProductQuantity,
			ProductDimensions:       item.ProductDimensions,
			PrintTimeHours:          item.PrintTimeHours,
			PrintTimeMinutes:        item.PrintTimeMinutes,
			PrintTimeDisplay:        printTimeDisplay,
			CostPresetID:            costPresetIDStr,
			CostPreset:              resolveCostPreset(item.CostPresetID),
			SetupTimeMinutes:        item.SetupTimeMinutes,
			ManualLaborMinutesTotal: item.ManualLaborMinutesTotal,
			AdditionalNotes:         item.AdditionalNotes,
			FilamentCost:            item.FilamentCost,
			WasteCost:               item.WasteCost,
			EnergyCost:              item.EnergyCost,
			SetupCost:               item.SetupCost,
			ManualLaborCost:         item.ManualLaborCost,
			ItemTotalCost:           item.ItemTotalCost,
			UnitPrice:               item.UnitPrice,
			SaleTotal:               saleTotal,
			SaleUnitPrice:           pricing.UnitPriceCents(saleTotal, item.ProductQuantity),
			Filaments:               filaments,
			Order:                   item.Order,
			CreatedAt:               item.CreatedAt,
			UpdatedAt:               item.UpdatedAt,
		}
	}

	return responses, totalPrintMinutes / 60, totalPrintMinutes % 60
}

// formatPrintTime renders a total print time as "5h30m" (or "30m" when under an
// hour), matching the long-standing response format.
func formatPrintTime(hours, minutes int) string {
	if hours > 0 {
		return fmt.Sprintf("%dh%02dm", hours, minutes)
	}
	return fmt.Sprintf("%dm", minutes)
}

// budgetLevelRefs resolves the budget's {id, name} references to its print profile
// and its budget-level cost preset. Both are best-effort (a missing name degrades
// to empty rather than failing the response) and either may be nil.
func (uc *BudgetUseCase) budgetLevelRefs(ctx context.Context, budget *entities.BudgetEntity) (*entities.ProfileRef, *entities.CostPresetRef) {
	var profileRef *entities.ProfileRef
	if budget.ProfileID != nil {
		profileRef = &entities.ProfileRef{ID: budget.ProfileID.String()}
		if uc.profileProvider != nil {
			if p, err := uc.profileProvider.ProfileByID(ctx, *budget.ProfileID, budget.OrganizationID); err == nil && p != nil {
				profileRef.Name = p.Name
			}
		}
	}

	var costRef *entities.CostPresetRef
	if budget.CostPresetID != nil {
		costRef = &entities.CostPresetRef{ID: budget.CostPresetID.String()}
		if info, err := uc.budgetRepository.GetPresetInfo(ctx, *budget.CostPresetID, "cost", budget.OrganizationID); err == nil && info != nil {
			costRef.Name = info.Name
		}
	}

	return profileRef, costRef
}

// buildBudgetResponse builds a complete budget response (budget + customer + items
// with sale values + print-time totals + budget-level profile/cost refs) for the
// given stored budget. It is the single builder used by every read/mutation
// endpoint that returns a BudgetResponse.
func (uc *BudgetUseCase) buildBudgetResponse(ctx context.Context, budgetID uuid.UUID, organizationID string) (*entities.BudgetResponse, error) {
	budget, err := uc.budgetRepository.FindByID(ctx, budgetID, organizationID)
	if err != nil {
		return nil, err
	}

	customerInfo, _ := uc.budgetRepository.GetCustomerInfo(ctx, budget.CustomerID, organizationID)
	items, _ := uc.budgetRepository.GetItems(ctx, budget.ID)

	itemResponses, totalHours, totalMins := buildBudgetItemResponses(ctx, uc.budgetRepository, items, budget.TotalCost, organizationID)
	profileRef, costRef := uc.budgetLevelRefs(ctx, budget)

	return &entities.BudgetResponse{
		BudgetEntity:          budget,
		Customer:              customerInfo,
		Items:                 itemResponses,
		Profile:               profileRef,
		CostPreset:            costRef,
		TotalPrintTimeHours:   totalHours,
		TotalPrintTimeMinutes: totalMins,
		TotalPrintTimeDisplay: formatPrintTime(totalHours, totalMins),
	}, nil
}
