package usecases

import (
	"net/http"

	coreErrors "github.com/RodolfoBonis/spooliq/core/errors"
	"github.com/RodolfoBonis/spooliq/core/helpers"
	"github.com/RodolfoBonis/spooliq/features/budget/domain/entities"
	"github.com/gin-gonic/gin"
)

// Preview calculates a budget's full cost breakdown WITHOUT persisting anything.
// It accepts the same body shape as Create (customer optional, items required),
// validates every referenced preset/filament against the organization, runs the
// pure pricing engine via the repository's stateless computation, and returns the
// per-item breakdown (costs + sale values) together with the budget totals.
//
// @Summary Preview budget costs
// @Description Calculate a budget's full cost breakdown without saving it. Useful for live quoting in the UI.
// @Tags budgets
// @Accept json
// @Produce json
// @Param request body entities.PreviewBudgetRequest true "Budget preview request"
// @Success 200 {object} entities.BudgetResponse
// @Failure 400 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /v1/budgets/preview [post]
// @Security BearerAuth
func (uc *BudgetUseCase) Preview(c *gin.Context) {
	ctx := c.Request.Context()

	organizationID := helpers.GetOrganizationID(c)
	if organizationID == "" {
		uc.logger.Error(ctx, "Organization ID not found", nil)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Organization ID required"})
		return
	}

	var request entities.PreviewBudgetRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		uc.logger.Error(ctx, "Failed to bind preview request", map[string]interface{}{
			"error": err.Error(),
		})
		appError := coreErrors.BadRequestError("Não foi possível interpretar os dados do orçamento. Verifique o formato e tente novamente.")
		c.JSON(appError.HTTPStatus(), gin.H{"error": appError.Message})
		return
	}

	if err := uc.validator.Struct(request); err != nil {
		uc.logger.Error(ctx, "Preview validation failed", map[string]interface{}{
			"error": err.Error(),
		})
		appError := coreErrors.BadRequestError("Dados do orçamento inválidos: informe ao menos um item, e cada item deve ter ao menos um filamento.")
		c.JSON(appError.HTTPStatus(), gin.H{"error": appError.Message})
		return
	}

	for i, item := range request.Items {
		if len(item.Filaments) == 0 {
			uc.logger.Error(ctx, "Preview item has no filaments", map[string]interface{}{
				"item_index": i,
			})
			appError := coreErrors.BadRequestError("Cada item do orçamento deve ter ao menos um filamento.")
			c.JSON(appError.HTTPStatus(), gin.H{"error": appError.Message})
			return
		}
	}

	// Resolve the machine/energy/cost presets exactly as Create does (request >
	// profile > default profile > org default preset). The resolver validates the
	// profile and every resolved preset against the organization.
	resolved, err := uc.resolvePresets(c, organizationID, PresetResolutionInput{
		ProfileID:       request.ProfileID,
		MachinePresetID: request.MachinePresetID,
		EnergyPresetID:  request.EnergyPresetID,
		CostPresetID:    request.CostPresetID,
	})
	if err != nil {
		uc.logger.Error(ctx, "Failed to resolve preview presets", map[string]interface{}{
			"error": err.Error(),
		})
		appError := coreErrors.BadRequestError("Referências inválidas no orçamento: " + err.Error())
		c.JSON(appError.HTTPStatus(), gin.H{"error": appError.Message})
		return
	}

	// Validate the remaining references (item-level cost presets + filaments).
	if err := uc.validateReferences(ctx, organizationID, nil, nil, nil, request.Items); err != nil {
		uc.logger.Error(ctx, "Invalid preview references", map[string]interface{}{
			"error": err.Error(),
		})
		appError := coreErrors.BadRequestError("Referências inválidas no orçamento: " + err.Error())
		c.JSON(appError.HTTPStatus(), gin.H{"error": appError.Message})
		return
	}

	// Assemble the persistence-agnostic pricing input from the request items.
	specs := make([]entities.PricingItemSpec, len(request.Items))
	for i, item := range request.Items {
		filaments := make([]entities.PricingFilamentSpec, len(item.Filaments))
		for j, f := range item.Filaments {
			filaments[j] = entities.PricingFilamentSpec{FilamentID: f.FilamentID, Quantity: f.Quantity}
		}
		specs[i] = entities.PricingItemSpec{
			ProductQuantity:         item.ProductQuantity,
			PrintTimeHours:          item.PrintTimeHours,
			PrintTimeMinutes:        item.PrintTimeMinutes,
			SetupTimeMinutes:        item.SetupTimeMinutes,
			ManualLaborMinutesTotal: item.ManualLaborMinutesTotal,
			CostPresetID:            item.CostPresetID,
			Filaments:               filaments,
		}
	}

	result, err := uc.budgetRepository.ComputeBudgetPricing(ctx, entities.PricingComputationInput{
		OrganizationID:     organizationID,
		IncludeEnergyCost:  request.IncludeEnergyCost,
		IncludeWasteCost:   request.IncludeWasteCost,
		MachinePresetID:    resolved.MachinePresetID,
		EnergyPresetID:     resolved.EnergyPresetID,
		BudgetCostPresetID: resolved.CostPresetID,
		Items:              specs,
	})
	if err != nil {
		uc.logger.Error(ctx, "Failed to compute budget preview", map[string]interface{}{
			"error": err.Error(),
		})
		appError := coreErrors.BadRequestError("Não foi possível calcular o orçamento: " + err.Error())
		c.JSON(appError.HTTPStatus(), gin.H{"error": appError.Message})
		return
	}

	// Build the transient (never-persisted) budget + item responses.
	previewBudget := &entities.BudgetEntity{
		OrganizationID:    organizationID,
		Name:              request.Name,
		Description:       request.Description,
		Status:            entities.StatusDraft,
		ProfileID:         resolved.ProfileID,
		MachinePresetID:   resolved.MachinePresetID,
		EnergyPresetID:    resolved.EnergyPresetID,
		CostPresetID:      resolved.CostPresetID,
		IncludeEnergyCost: request.IncludeEnergyCost,
		IncludeWasteCost:  request.IncludeWasteCost,
		DeliveryDays:      request.DeliveryDays,
		PaymentTerms:      request.PaymentTerms,
		Notes:             request.Notes,
		FilamentCost:      result.FilamentCost,
		WasteCost:         result.WasteCost,
		EnergyCost:        result.EnergyCost,
		SetupCost:         result.SetupCost,
		LaborCost:         result.LaborCost,
		OverheadCost:      result.Overhead,
		ProfitAmount:      result.Profit,
		TotalCost:         result.Total,
	}

	var customerInfo *entities.CustomerInfo
	if request.CustomerID != nil {
		previewBudget.CustomerID = *request.CustomerID
		// Best-effort: enrich the response with the customer when it resolves inside
		// the org. A missing/other-tenant customer does not fail a preview.
		customerInfo, _ = uc.budgetRepository.GetCustomerInfo(ctx, *request.CustomerID, organizationID)
	}

	itemResponses := make([]entities.BudgetItemResponse, len(request.Items))
	presetNameCache := make(map[string]string)
	var totalPrintMinutes int
	for i, item := range request.Items {
		res := result.Items[i]

		var costPresetIDStr *string
		var costPresetRef *entities.CostPresetRef
		if item.CostPresetID != nil {
			s := item.CostPresetID.String()
			costPresetIDStr = &s
			costPresetRef = &entities.CostPresetRef{ID: s}
			if name, ok := presetNameCache[s]; ok {
				costPresetRef.Name = name
			} else if info, perr := uc.budgetRepository.GetPresetInfo(ctx, *item.CostPresetID, "cost", organizationID); perr == nil && info != nil {
				costPresetRef.Name = info.Name
				presetNameCache[s] = info.Name
			}
		}

		filaments := make([]entities.FilamentUsageInfo, len(item.Filaments))
		for j, f := range item.Filaments {
			filaments[j] = entities.FilamentUsageInfo{
				FilamentID: f.FilamentID.String(),
				Quantity:   f.Quantity,
				Order:      f.Order,
			}
		}

		totalPrintMinutes += (item.PrintTimeHours * 60) + item.PrintTimeMinutes

		itemResponses[i] = entities.BudgetItemResponse{
			ProductName:             item.ProductName,
			ProductDescription:      item.ProductDescription,
			ProductQuantity:         item.ProductQuantity,
			ProductDimensions:       item.ProductDimensions,
			PrintTimeHours:          item.PrintTimeHours,
			PrintTimeMinutes:        item.PrintTimeMinutes,
			PrintTimeDisplay:        formatPrintTime(item.PrintTimeHours, item.PrintTimeMinutes),
			CostPresetID:            costPresetIDStr,
			CostPreset:              costPresetRef,
			SetupTimeMinutes:        item.SetupTimeMinutes,
			ManualLaborMinutesTotal: item.ManualLaborMinutesTotal,
			AdditionalNotes:         item.AdditionalNotes,
			FilamentCost:            res.FilamentCost,
			WasteCost:               res.WasteCost,
			EnergyCost:              res.EnergyCost,
			SetupCost:               res.SetupCost,
			ManualLaborCost:         res.ManualLaborCost,
			ItemTotalCost:           res.ItemTotalCost,
			UnitPrice:               res.UnitCost,
			SaleTotal:               res.SaleTotal,
			SaleUnitPrice:           res.SaleUnitPrice,
			Filaments:               filaments,
			Order:                   item.Order,
		}
	}

	totalHours := totalPrintMinutes / 60
	totalMins := totalPrintMinutes % 60

	profileRef, costRef := uc.budgetLevelRefs(ctx, previewBudget)

	response := entities.BudgetResponse{
		BudgetEntity:          previewBudget,
		Customer:              customerInfo,
		Items:                 itemResponses,
		Profile:               profileRef,
		CostPreset:            costRef,
		TotalPrintTimeHours:   totalHours,
		TotalPrintTimeMinutes: totalMins,
		TotalPrintTimeDisplay: formatPrintTime(totalHours, totalMins),
	}

	uc.logger.Info(ctx, "Budget preview computed successfully", map[string]interface{}{
		"items": len(request.Items),
		"total": result.Total,
	})

	c.JSON(http.StatusOK, response)
}
