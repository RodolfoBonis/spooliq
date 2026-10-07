package usecases

import (
	"crypto/rand"
	"encoding/base64"
	"net/http"
	"time"

	coreErrors "github.com/RodolfoBonis/spooliq/core/errors"
	"github.com/RodolfoBonis/spooliq/core/helpers"
	activityEntities "github.com/RodolfoBonis/spooliq/features/activity/domain/entities"
	"github.com/RodolfoBonis/spooliq/features/budget/domain/entities"
	budgetRepo "github.com/RodolfoBonis/spooliq/features/budget/domain/repositories"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// shareableStatuses are the budget statuses from which a public share link may be
// created. cancelled, printing and completed are intentionally excluded.
var shareableStatuses = map[entities.BudgetStatus]bool{
	entities.StatusDraft:    true,
	entities.StatusSent:     true,
	entities.StatusApproved: true,
	entities.StatusRejected: true,
	entities.StatusExpired:  true,
}

// generatePublicToken returns a URL-safe, unpadded base64 token derived from 32
// bytes of crypto/rand entropy (43 characters). Used as the public share token.
func generatePublicToken() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}

// shareResponse is the body returned by POST /budgets/:id/share.
type shareResponse struct {
	PublicToken string     `json:"public_token"`
	Status      string     `json:"status"`
	ValidUntil  *time.Time `json:"valid_until"`
}

// Share creates (or returns the existing) public share token for a budget.
// @Summary Share budget publicly
// @Description Create or return the public share token for a budget. Idempotent:
// @Description repeated calls return the same token. A draft is transitioned to sent
// @Description (with the same valid_until side effect as UpdateStatus).
// @Tags budgets
// @Accept json
// @Produce json
// @Param id path string true "Budget ID"
// @Success 200 {object} map[string]interface{} "public_token, status, valid_until"
// @Failure 400 {object} errors.HTTPError
// @Failure 404 {object} errors.HTTPError
// @Failure 409 {object} errors.HTTPError
// @Failure 500 {object} errors.HTTPError
// @Router /budgets/{id}/share [post]
// @Security BearerAuth
func (uc *BudgetUseCase) Share(c *gin.Context) {
	ctx := c.Request.Context()

	organizationID := helpers.GetOrganizationID(c)
	if organizationID == "" {
		coreErrors.Respond(c, coreErrors.BadRequest(CodeOrganizationRequired, "Organização não identificada"))
		return
	}
	userID := helpers.GetUserID(c)

	budgetID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		coreErrors.Respond(c, coreErrors.BadRequest(CodeInvalidBudgetID, "ID de orçamento inválido"))
		return
	}

	budget, err := uc.budgetRepository.FindByID(ctx, budgetID, organizationID)
	if err != nil {
		respondBudgetError(c, err)
		return
	}

	if !shareableStatuses[budget.Status] {
		coreErrors.Respond(c, coreErrors.Conflict(CodeBudgetNotShareable, "Este orçamento não pode ser compartilhado no status atual"))
		return
	}

	now := time.Now()
	needsTransition := budget.Status == entities.StatusDraft

	var tokenToSet string
	if budget.PublicToken == nil || *budget.PublicToken == "" {
		tokenToSet, err = generatePublicToken()
		if err != nil {
			uc.logger.Error(ctx, "Failed to generate public token", map[string]interface{}{"error": err.Error()})
			coreErrors.Respond(c, coreErrors.Internal())
			return
		}
	}

	// A draft becoming sent gets a valid_until when it has none, mirroring UpdateStatus.
	var validUntilToSet *time.Time
	if needsTransition && budget.ValidUntil == nil {
		validityDays, _, derr := uc.budgetRepository.GetCompanyQuoteDefaults(ctx, organizationID)
		if derr != nil {
			respondBudgetError(c, derr)
			return
		}
		computed := computeValidUntil(now, validityDays)
		validUntilToSet = &computed
	}

	var history *entities.BudgetStatusHistoryEntity
	if needsTransition {
		history = &entities.BudgetStatusHistoryEntity{
			ID:             uuid.New(),
			BudgetID:       budget.ID,
			OrganizationID: organizationID,
			PreviousStatus: entities.StatusDraft,
			NewStatus:      entities.StatusSent,
			ChangedBy:      userID,
			CreatedAt:      now,
		}
	}

	if err := uc.budgetRepository.WithTransaction(ctx, func(repo budgetRepo.BudgetRepository) error {
		if needsTransition {
			if err := repo.UpdateStatus(ctx, budget.ID, organizationID, entities.StatusDraft, entities.StatusSent); err != nil {
				return err
			}
			if validUntilToSet != nil {
				if err := repo.SetValidUntil(ctx, budget.ID, organizationID, validUntilToSet); err != nil {
					return err
				}
			}
			if err := repo.AddStatusHistory(ctx, history); err != nil {
				return err
			}
		}
		if tokenToSet != "" {
			if err := repo.SetShareToken(ctx, budget.ID, organizationID, tokenToSet, now); err != nil {
				return err
			}
		}
		return nil
	}); err != nil {
		respondBudgetError(c, err)
		return
	}

	// Re-read the authoritative state (status, valid_until, token).
	updated, err := uc.budgetRepository.FindByID(ctx, budgetID, organizationID)
	if err != nil {
		respondBudgetError(c, err)
		return
	}

	token := ""
	if updated.PublicToken != nil {
		token = *updated.PublicToken
	}

	c.JSON(http.StatusOK, shareResponse{
		PublicToken: token,
		Status:      string(updated.Status),
		ValidUntil:  updated.ValidUntil,
	})

	if needsTransition {
		uc.activityService.Record(c.Request.Context(), activityEntities.ActivityEntity{
			OrganizationID: organizationID,
			UserID:         userID,
			Action:         activityEntities.ActionStatusChanged,
			EntityType:     activityEntities.EntityBudget,
			EntityID:       budget.ID.String(),
			EntityName:     budget.Name,
			Metadata: map[string]any{
				"previous_status": string(entities.StatusDraft),
				"new_status":      string(entities.StatusSent),
			},
			CreatedAt: now,
		})
	}
}

// RevokeShare revokes the public share token for a budget.
// @Summary Revoke budget public share
// @Description Revoke the public share token for a budget (sets it to NULL).
// @Tags budgets
// @Produce json
// @Param id path string true "Budget ID"
// @Success 204 "No Content"
// @Failure 400 {object} errors.HTTPError
// @Failure 404 {object} errors.HTTPError
// @Failure 500 {object} errors.HTTPError
// @Router /budgets/{id}/share [delete]
// @Security BearerAuth
func (uc *BudgetUseCase) RevokeShare(c *gin.Context) {
	ctx := c.Request.Context()

	organizationID := helpers.GetOrganizationID(c)
	if organizationID == "" {
		coreErrors.Respond(c, coreErrors.BadRequest(CodeOrganizationRequired, "Organização não identificada"))
		return
	}

	budgetID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		coreErrors.Respond(c, coreErrors.BadRequest(CodeInvalidBudgetID, "ID de orçamento inválido"))
		return
	}

	if err := uc.budgetRepository.RevokeShareToken(ctx, budgetID, organizationID); err != nil {
		respondBudgetError(c, err)
		return
	}

	c.Status(http.StatusNoContent)
}
