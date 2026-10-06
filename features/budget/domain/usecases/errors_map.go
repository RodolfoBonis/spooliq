package usecases

import (
	"errors"

	coreErrors "github.com/RodolfoBonis/spooliq/core/errors"
	"github.com/RodolfoBonis/spooliq/features/budget/domain/entities"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// Stable, snake_case error codes surfaced by the budget feature. They are the
// machine-readable contract the web app branches on (replacing the previous
// English-message matching). The HTTP status of each is fixed by the constructor
// used in budgetAPIError below.
const (
	// CodeBudgetNotFound (404) — the budget does not exist in the organization.
	CodeBudgetNotFound = "budget_not_found"
	// CodeBudgetNotEditable (409) — the budget is not a draft and cannot be edited/recalculated.
	CodeBudgetNotEditable = "budget_not_editable"
	// CodeBudgetStatusConflict (409) — the status changed concurrently (optimistic guard).
	CodeBudgetStatusConflict = "budget_status_conflict"
	// CodeBudgetNotDeletable (409) — printing/completed budgets cannot be deleted.
	CodeBudgetNotDeletable = "budget_not_deletable"
	// CodeInvalidStatusTransition (400) — the requested status transition is not allowed.
	CodeInvalidStatusTransition = "invalid_status_transition"
	// CodeInvalidBudgetID (400) — the path budget ID is not a valid UUID.
	CodeInvalidBudgetID = "invalid_budget_id"
	// CodeInvalidCustomerID (400) — the customer ID (path or filter) is not a valid UUID.
	CodeInvalidCustomerID = "invalid_customer_id"
	// CodeInvalidPresetReference (400) — a referenced preset is missing/wrong-type/cross-tenant.
	CodeInvalidPresetReference = "invalid_preset_reference"
	// CodeProfileNotFound (400) — the referenced print profile is not in the organization.
	CodeProfileNotFound = "profile_not_found"
	// CodeFilamentNotFound (400) — a referenced filament is not in the organization.
	CodeFilamentNotFound = "filament_not_found"
	// CodeCustomerNotFound (404) — the referenced customer is not in the organization.
	CodeCustomerNotFound = "customer_not_found"
	// CodeInvalidStatusFilter (400) — the list status filter is not a known status.
	CodeInvalidStatusFilter = "invalid_status_filter"
	// CodeInvalidDate (400) — a from/to date filter could not be parsed.
	CodeInvalidDate = "invalid_date"
	// CodeInvalidRequest (400) — the request body could not be decoded.
	CodeInvalidRequest = "invalid_request"
	// CodeOrganizationRequired (400) — the organization ID is missing from the context.
	CodeOrganizationRequired = "organization_required"
	// CodeUserRequired (400) — the user ID is missing from the context.
	CodeUserRequired = "user_required"
)

// budgetAPIError maps a domain error to a stable *APIError (code + pt-BR message +
// HTTP status). It returns nil when err is not a recognized budget domain error, so
// callers can fall back to coreErrors.Respond (which logs and returns a generic 500
// for unexpected errors, and still maps gorm/validation errors on its own).
func budgetAPIError(err error) *coreErrors.APIError {
	switch {
	case err == nil:
		return nil
	case errors.Is(err, entities.ErrBudgetNotFound), errors.Is(err, gorm.ErrRecordNotFound):
		return coreErrors.NotFoundErr(CodeBudgetNotFound, "Orçamento não encontrado")
	case errors.Is(err, entities.ErrBudgetNotEditable):
		return coreErrors.Conflict(CodeBudgetNotEditable, "Apenas orçamentos em rascunho podem ser editados")
	case errors.Is(err, entities.ErrBudgetStatusConflict):
		return coreErrors.Conflict(CodeBudgetStatusConflict, "O status do orçamento foi alterado por outra requisição. Recarregue e tente novamente.")
	case errors.Is(err, entities.ErrBudgetNotDeletable):
		return coreErrors.Conflict(CodeBudgetNotDeletable, "Não é possível excluir orçamentos em impressão ou concluídos")
	case errors.Is(err, entities.ErrInvalidTransition):
		return coreErrors.BadRequest(CodeInvalidStatusTransition, "Transição de status inválida")
	case errors.Is(err, entities.ErrProfileNotFound):
		return coreErrors.BadRequest(CodeProfileNotFound, "O perfil de impressão informado não pertence à sua organização")
	case errors.Is(err, entities.ErrInvalidPresetReference), errors.Is(err, entities.ErrPresetNotFound):
		return coreErrors.BadRequest(CodeInvalidPresetReference, "Uma ou mais referências de preset são inválidas ou não pertencem à sua organização")
	case errors.Is(err, entities.ErrFilamentNotFound):
		return coreErrors.BadRequest(CodeFilamentNotFound, "Um ou mais filamentos informados não pertencem à sua organização")
	case errors.Is(err, entities.ErrCustomerNotFound):
		return coreErrors.NotFoundErr(CodeCustomerNotFound, "Cliente não encontrado")
	case errors.Is(err, entities.ErrInvalidStatusFilter):
		return coreErrors.BadRequest(CodeInvalidStatusFilter, "Status de filtro inválido")
	default:
		return nil
	}
}

// respondBudgetError writes the standard error envelope for err. Recognized budget
// domain errors get their stable code + pt-BR message; anything else is delegated to
// coreErrors.Respond, which maps gorm/validation/binding errors and logs unexpected
// failures as a generic 500 (never leaking internals).
func respondBudgetError(c *gin.Context, err error) {
	if apiErr := budgetAPIError(err); apiErr != nil {
		coreErrors.Respond(c, apiErr)
		return
	}
	coreErrors.Respond(c, err)
}
