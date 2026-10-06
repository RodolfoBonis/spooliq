package usecases

import (
	"context"
	"fmt"

	"github.com/RodolfoBonis/spooliq/features/budget/domain/entities"
	"github.com/RodolfoBonis/spooliq/features/budget/domain/repositories"
	pricing "github.com/RodolfoBonis/spooliq/features/budget/domain/services"
	"github.com/google/uuid"
)

// assembleItemResponse maps a single stored budget item into its API response,
// given the item's already-resolved sale total, filament usage and cost-preset ref.
// It is the single source of truth for the item response shape, shared by the
// per-budget (detail/PDF) path and the batched list path so both emit identical
// fields. It also returns the item's print time in minutes so callers can total it.
func assembleItemResponse(item *entities.BudgetItemEntity, saleTotal int64, filaments []entities.FilamentUsageInfo, costPresetRef *entities.CostPresetRef) (entities.BudgetItemResponse, int) {
	printTimeDisplay := ""
	if item.PrintTimeHours > 0 {
		printTimeDisplay = fmt.Sprintf("%dh%02dm", item.PrintTimeHours, item.PrintTimeMinutes)
	} else {
		printTimeDisplay = fmt.Sprintf("%dm", item.PrintTimeMinutes)
	}
	printMinutes := (item.PrintTimeHours * 60) + item.PrintTimeMinutes

	var costPresetIDStr *string
	if item.CostPresetID != nil {
		s := item.CostPresetID.String()
		costPresetIDStr = &s
	}

	if filaments == nil {
		filaments = []entities.FilamentUsageInfo{}
	}

	return entities.BudgetItemResponse{
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
		CostPreset:              costPresetRef,
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
	}, printMinutes
}

// saleTotalsFor distributes the budget-wide overhead+profit markup
// (budgetTotal - sum of item costs) across items proportionally to each item's
// direct cost, so the per-item shares sum EXACTLY to budgetTotal.
func saleTotalsFor(items []*entities.BudgetItemEntity, budgetTotal int64) []int64 {
	itemCosts := make([]int64, len(items))
	for i, item := range items {
		itemCosts[i] = item.ItemTotalCost
	}
	return pricing.DistributeMarkup(itemCosts, budgetTotal)
}

// buildBudgetItemResponses maps stored budget items into API responses for a SINGLE
// budget, resolving each item's filament usage and cost-preset name directly from
// the repository. It is used by the detail endpoint and the PDF generator (one
// budget at a time); the list endpoint uses the batched path instead. It returns the
// item responses plus the summed print time (hours, minutes).
func buildBudgetItemResponses(ctx context.Context, repo repositories.BudgetRepository, items []*entities.BudgetItemEntity, budgetTotal int64, organizationID string) ([]entities.BudgetItemResponse, int, int) {
	saleTotals := saleTotalsFor(items, budgetTotal)

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
		resp, minutes := assembleItemResponse(item, saleTotals[i], filaments, resolveCostPreset(item.CostPresetID))
		responses[i] = resp
		totalPrintMinutes += minutes
	}

	return responses, totalPrintMinutes / 60, totalPrintMinutes % 60
}

// assembleItemResponsesFromMaps builds the item responses for a budget from already
// batch-loaded data (filament usage keyed by item ID, cost-preset names keyed by
// preset ID), performing NO per-item queries. It is the list-path counterpart of
// buildBudgetItemResponses and emits an identical response shape.
func assembleItemResponsesFromMaps(items []*entities.BudgetItemEntity, budgetTotal int64, filamentsByItem map[uuid.UUID][]entities.FilamentUsageInfo, costPresetNames map[uuid.UUID]string) ([]entities.BudgetItemResponse, int, int) {
	saleTotals := saleTotalsFor(items, budgetTotal)

	responses := make([]entities.BudgetItemResponse, len(items))
	var totalPrintMinutes int
	for i, item := range items {
		var costPresetRef *entities.CostPresetRef
		if item.CostPresetID != nil {
			costPresetRef = &entities.CostPresetRef{
				ID:   item.CostPresetID.String(),
				Name: costPresetNames[*item.CostPresetID],
			}
		}
		resp, minutes := assembleItemResponse(item, saleTotals[i], filamentsByItem[item.ID], costPresetRef)
		responses[i] = resp
		totalPrintMinutes += minutes
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
// endpoint that returns a BudgetResponse for ONE budget.
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

// buildBudgetListResponses assembles the API responses for a PAGE of budgets using a
// constant number of queries, independent of the page size. It batch-loads the
// customers, the items for all budgets, the filament usage for all items, and the
// cost-preset/profile names for every referenced ID, then builds each response from
// those in-memory maps (no per-budget or per-item query). The response shape matches
// buildBudgetResponse, including the budget-level profile/cost_preset refs.
func (uc *BudgetUseCase) buildBudgetListResponses(ctx context.Context, budgets []*entities.BudgetEntity, organizationID string) ([]entities.BudgetResponse, error) {
	responses := make([]entities.BudgetResponse, 0, len(budgets))
	if len(budgets) == 0 {
		return responses, nil
	}

	customerIDs := make([]uuid.UUID, 0, len(budgets))
	budgetIDs := make([]uuid.UUID, 0, len(budgets))
	profileIDs := make([]uuid.UUID, 0)
	costPresetIDs := make([]uuid.UUID, 0)
	for _, budget := range budgets {
		customerIDs = append(customerIDs, budget.CustomerID)
		budgetIDs = append(budgetIDs, budget.ID)
		if budget.ProfileID != nil {
			profileIDs = append(profileIDs, *budget.ProfileID)
		}
		if budget.CostPresetID != nil {
			costPresetIDs = append(costPresetIDs, *budget.CostPresetID)
		}
	}

	customers, err := uc.budgetRepository.GetCustomersInfo(ctx, customerIDs, organizationID)
	if err != nil {
		return nil, err
	}

	itemsByBudget, err := uc.budgetRepository.GetItemsByBudgetIDs(ctx, budgetIDs, organizationID)
	if err != nil {
		return nil, err
	}

	itemIDs := make([]uuid.UUID, 0)
	for _, items := range itemsByBudget {
		for _, item := range items {
			itemIDs = append(itemIDs, item.ID)
			if item.CostPresetID != nil {
				costPresetIDs = append(costPresetIDs, *item.CostPresetID)
			}
		}
	}

	filamentsByItem, err := uc.budgetRepository.GetFilamentUsageInfoByItemIDs(ctx, itemIDs, organizationID)
	if err != nil {
		return nil, err
	}

	costPresetNames, err := uc.budgetRepository.GetCostPresetNames(ctx, costPresetIDs, organizationID)
	if err != nil {
		return nil, err
	}

	profileNames, err := uc.budgetRepository.GetProfileNames(ctx, profileIDs, organizationID)
	if err != nil {
		return nil, err
	}

	for _, budget := range budgets {
		items := itemsByBudget[budget.ID]
		itemResponses, totalHours, totalMins := assembleItemResponsesFromMaps(items, budget.TotalCost, filamentsByItem, costPresetNames)

		var profileRef *entities.ProfileRef
		if budget.ProfileID != nil {
			profileRef = &entities.ProfileRef{ID: budget.ProfileID.String(), Name: profileNames[*budget.ProfileID]}
		}
		var costRef *entities.CostPresetRef
		if budget.CostPresetID != nil {
			costRef = &entities.CostPresetRef{ID: budget.CostPresetID.String(), Name: costPresetNames[*budget.CostPresetID]}
		}

		responses = append(responses, entities.BudgetResponse{
			BudgetEntity:          budget,
			Customer:              customers[budget.CustomerID],
			Items:                 itemResponses,
			Profile:               profileRef,
			CostPreset:            costRef,
			TotalPrintTimeHours:   totalHours,
			TotalPrintTimeMinutes: totalMins,
			TotalPrintTimeDisplay: formatPrintTime(totalHours, totalMins),
		})
	}

	return responses, nil
}
