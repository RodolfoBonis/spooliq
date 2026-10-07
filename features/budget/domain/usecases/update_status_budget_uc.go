package usecases

import (
	"errors"
	"net/http"
	"time"

	coreErrors "github.com/RodolfoBonis/spooliq/core/errors"
	"github.com/RodolfoBonis/spooliq/core/helpers"
	"github.com/RodolfoBonis/spooliq/core/validation"
	activityEntities "github.com/RodolfoBonis/spooliq/features/activity/domain/entities"
	"github.com/RodolfoBonis/spooliq/features/budget/domain/entities"
	budgetRepo "github.com/RodolfoBonis/spooliq/features/budget/domain/repositories"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// UpdateStatus updates the status of a budget
// @Summary Update budget status
// @Description Update the status of a budget with validation
// @Tags budgets
// @Accept json
// @Produce json
// @Param id path string true "Budget ID"
// @Param request body entities.UpdateStatusRequest true "Update status request"
// @Success 200 {object} entities.BudgetResponse
// @Failure 400 {object} errors.HTTPError
// @Failure 404 {object} errors.HTTPError
// @Failure 409 {object} errors.HTTPError
// @Failure 500 {object} errors.HTTPError
// @Router /budgets/{id}/status [patch]
// @Security BearerAuth
func (uc *BudgetUseCase) UpdateStatus(c *gin.Context) {
	ctx := c.Request.Context()

	organizationID := helpers.GetOrganizationID(c)
	if organizationID == "" {
		uc.logger.Error(ctx, "Organization ID not found", nil)
		coreErrors.Respond(c, coreErrors.BadRequest(CodeOrganizationRequired, "Organização não identificada"))
		return
	}

	userID := helpers.GetUserID(c)
	if userID == "" {
		uc.logger.Error(ctx, "User ID not found", nil)
		coreErrors.Respond(c, coreErrors.BadRequest(CodeUserRequired, "Usuário não identificado"))
		return
	}

	// Parse budget ID
	budgetID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		uc.logger.Error(ctx, "Invalid budget ID", map[string]interface{}{"error": err.Error()})
		coreErrors.Respond(c, coreErrors.BadRequest(CodeInvalidBudgetID, "ID de orçamento inválido"))
		return
	}

	var request entities.UpdateStatusRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		uc.logger.Error(ctx, "Failed to bind request", map[string]interface{}{"error": err.Error()})
		coreErrors.Respond(c, err)
		return
	}

	if err := validation.Validate(&request); err != nil {
		uc.logger.Error(ctx, "Validation failed", map[string]interface{}{"error": err.Error()})
		coreErrors.Respond(c, err)
		return
	}

	// Get existing budget
	budget, err := uc.budgetRepository.FindByID(ctx, budgetID, organizationID)
	if err != nil {
		uc.logger.Error(ctx, "Failed to retrieve budget", map[string]interface{}{
			"error":     err.Error(),
			"budget_id": budgetID,
		})
		respondBudgetError(c, err)
		return
	}

	// Check if transition is valid (allowed-transition validation is preserved).
	if !budget.IsValidTransition(request.Status) {
		uc.logger.Error(ctx, "Invalid status transition", map[string]interface{}{
			"budget_id":        budgetID,
			"current_status":   budget.Status,
			"requested_status": request.Status,
		})
		coreErrors.Respond(c, coreErrors.BadRequest(CodeInvalidStatusTransition, "Transição de status inválida"))
		return
	}

	// Capture the status we transitioned from; it is the optimistic guard for the
	// write (UpdateStatus only matches rows still in this status).
	previousStatus := budget.Status

	history := &entities.BudgetStatusHistoryEntity{
		ID:             uuid.New(),
		BudgetID:       budget.ID,
		OrganizationID: organizationID,
		PreviousStatus: previousStatus,
		NewStatus:      request.Status,
		ChangedBy:      userID,
		Notes:          request.Notes,
		CreatedAt:      time.Now(),
	}

	// Do the status update and the history insert atomically in ONE transaction so
	// a history row is never recorded for a transition that didn't actually apply
	// (and vice-versa). The status write is guarded by the expected current status;
	// if a concurrent change already moved the budget off previousStatus the write
	// matches no row and we surface a 409 Conflict.
	// Resolve the quote-validity side effects of this transition:
	//   - becoming "sent" with no valid_until set => set valid_until to now + default
	//     company validity days;
	//   - reopening to "draft" => clear valid_until so the next send recomputes it.
	now := time.Now()
	var validUntilToSet *time.Time
	setValidUntil := false
	clearValidUntil := false
	if request.Status == entities.StatusSent && budget.ValidUntil == nil {
		validityDays, _, derr := uc.budgetRepository.GetCompanyQuoteDefaults(ctx, organizationID)
		if derr != nil {
			uc.logger.Error(ctx, "Failed to load company quote defaults", map[string]interface{}{"error": derr.Error()})
			respondBudgetError(c, derr)
			return
		}
		computed := computeValidUntil(now, validityDays)
		validUntilToSet = &computed
		setValidUntil = true
	}
	if request.Status == entities.StatusDraft {
		clearValidUntil = true
	}

	if err := uc.budgetRepository.WithTransaction(ctx, func(repo budgetRepo.BudgetRepository) error {
		if err := repo.UpdateStatus(ctx, budget.ID, organizationID, previousStatus, request.Status); err != nil {
			return err
		}
		if setValidUntil {
			if err := repo.SetValidUntil(ctx, budget.ID, organizationID, validUntilToSet); err != nil {
				return err
			}
		}
		if clearValidUntil {
			if err := repo.SetValidUntil(ctx, budget.ID, organizationID, nil); err != nil {
				return err
			}
		}
		if err := repo.AddStatusHistory(ctx, history); err != nil {
			return err
		}
		// On completion, deduct filament stock in the SAME transaction so the status
		// change and the consumption movements are atomic. The deduction is idempotent
		// (partial unique index), so a retry never double-counts.
		if request.Status == entities.StatusCompleted && uc.stockDeductor != nil {
			if err := uc.stockDeductor.DeductForCompletedBudget(ctx, repo.UnderlyingTx(), budget.ID, organizationID, userID); err != nil {
				return err
			}
		}
		return nil
	}); err != nil {
		if errors.Is(err, entities.ErrBudgetStatusConflict) {
			uc.logger.Warning(ctx, "Budget status changed concurrently", map[string]interface{}{
				"budget_id":       budgetID,
				"expected_status": previousStatus,
			})
			coreErrors.Respond(c, coreErrors.Conflict(CodeBudgetStatusConflict, "O status do orçamento foi alterado por outra requisição. Recarregue e tente novamente."))
			return
		}
		uc.logger.Error(ctx, "Failed to update budget status", map[string]interface{}{"error": err.Error()})
		respondBudgetError(c, err)
		return
	}

	// Build response from the freshly stored state.
	response, err := uc.buildBudgetResponse(ctx, budgetID, organizationID)
	if err != nil {
		respondBudgetError(c, err)
		return
	}

	uc.logger.Info(ctx, "Budget status updated successfully", map[string]interface{}{
		"budget_id":  budget.ID,
		"new_status": request.Status,
	})

	c.JSON(http.StatusOK, response)

	// Record activity (fire-and-forget)
	uc.activityService.Record(c.Request.Context(), activityEntities.ActivityEntity{
		OrganizationID: organizationID,
		UserID:         userID,
		Action:         activityEntities.ActionStatusChanged,
		EntityType:     activityEntities.EntityBudget,
		EntityID:       budget.ID.String(),
		EntityName:     budget.Name,
		Metadata: map[string]any{
			"previous_status": string(previousStatus),
			"new_status":      string(request.Status),
		},
		CreatedAt: time.Now(),
	})
}
