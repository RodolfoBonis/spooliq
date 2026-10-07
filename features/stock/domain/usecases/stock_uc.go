// Package usecases implements the filament stock-movement business logic.
package usecases

import (
	"errors"
	"net/http"
	"time"

	log "github.com/RodolfoBonis/go-otel-agent/logger"
	coreErrors "github.com/RodolfoBonis/spooliq/core/errors"
	"github.com/RodolfoBonis/spooliq/core/helpers"
	"github.com/RodolfoBonis/spooliq/core/validation"
	activityEntities "github.com/RodolfoBonis/spooliq/features/activity/domain/entities"
	activityUc "github.com/RodolfoBonis/spooliq/features/activity/domain/usecases"
	"github.com/RodolfoBonis/spooliq/features/stock/domain/entities"
	"github.com/RodolfoBonis/spooliq/features/stock/domain/repositories"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// IStockUseCase defines the filament stock movement HTTP operations.
type IStockUseCase interface {
	CreateMovement(c *gin.Context)
	ListMovements(c *gin.Context)
}

// StockUseCase implements filament stock movement operations.
type StockUseCase struct {
	repository      repositories.StockRepository
	logger          log.Logger
	activityService activityUc.IActivityService
}

// NewStockUseCase creates a new stock use case.
func NewStockUseCase(repository repositories.StockRepository, logger log.Logger, activityService activityUc.IActivityService) IStockUseCase {
	return &StockUseCase{repository: repository, logger: logger, activityService: activityService}
}

// movementSortWhitelist maps the public sort name to a safe column. created_at is
// the only sortable column; id is the stable tie-breaker.
var movementSortWhitelist = map[string]string{
	"created_at": "m.created_at",
}

// CreateMovement records a manual stock movement for a filament.
// @Summary Create filament stock movement
// @Description Record a manual stock movement (purchase, adjustment or waste) for a filament. Any manual movement enables stock tracking for the filament.
// @Tags Filament Stock
// @Accept json
// @Produce json
// @Param id path string true "Filament ID (UUID)"
// @Param request body entities.CreateMovementRequest true "Stock movement data"
// @Success 201 {object} entities.CreateMovementResponse "Movement recorded and filament balance refreshed"
// @Failure 400 {object} errors.APIError
// @Failure 401 {object} errors.APIError
// @Failure 404 {object} errors.APIError
// @Failure 500 {object} errors.APIError
// @Security BearerAuth
// @Router /filaments/{id}/stock-movements [post]
func (uc *StockUseCase) CreateMovement(c *gin.Context) {
	ctx := c.Request.Context()

	organizationID := helpers.GetOrganizationID(c)
	if organizationID == "" {
		uc.logger.Error(ctx, "Organization ID not found", nil)
		coreErrors.Respond(c, coreErrors.BadRequest("organization_required", "Organização não encontrada no contexto"))
		return
	}

	userID := helpers.GetUserID(c)
	if userID == "" {
		uc.logger.Error(ctx, "User ID not found in context", nil)
		coreErrors.Respond(c, coreErrors.BadRequest("user_required", "Usuário não encontrado no contexto"))
		return
	}

	filamentID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		uc.logger.Error(ctx, "Invalid filament ID", map[string]interface{}{"filament_id": c.Param("id")})
		coreErrors.Respond(c, coreErrors.BadRequest("invalid_filament_id", "ID de filamento inválido"))
		return
	}

	var request entities.CreateMovementRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		uc.logger.Error(ctx, "Invalid stock movement payload", map[string]interface{}{"error": err.Error()})
		coreErrors.Respond(c, err)
		return
	}
	if err := validation.Validate(request); err != nil {
		uc.logger.Error(ctx, "Stock movement validation failed", map[string]interface{}{"error": err.Error()})
		coreErrors.Respond(c, err)
		return
	}

	// Domain validation: type must be user-creatable and grams must satisfy the
	// per-type rule. Normalize returns the SIGNED grams to apply.
	signed, code := request.Normalize()
	if code != "" {
		msg := "Tipo de movimentação inválido"
		if code == entities.CodeInvalidMovementGrams {
			msg = "Quantidade de gramas inválida para esta movimentação"
		}
		uc.logger.Warning(ctx, "Invalid stock movement", map[string]interface{}{"code": code, "type": request.Type})
		coreErrors.Respond(c, coreErrors.BadRequest(code, msg))
		return
	}

	// unit_price_per_kg is meaningful for purchases only; drop it otherwise.
	unitPrice := request.UnitPricePerKg
	if request.Type != entities.MovementPurchase {
		unitPrice = nil
	}

	movement := &entities.StockMovementEntity{
		OrganizationID: organizationID,
		FilamentID:     filamentID,
		Type:           request.Type,
		Grams:          signed,
		UnitPricePerKg: unitPrice,
		Note:           request.Note,
		CreatedBy:      userID,
	}

	stored, summary, err := uc.repository.CreateManualMovement(ctx, movement)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			uc.logger.Warning(ctx, "Filament not found for stock movement", map[string]interface{}{"filament_id": filamentID})
			coreErrors.Respond(c, coreErrors.NotFoundErr("filament_not_found", "Filamento não encontrado"))
			return
		}
		uc.logger.Error(ctx, "Failed to create stock movement", map[string]interface{}{"error": err.Error(), "filament_id": filamentID})
		coreErrors.Respond(c, err)
		return
	}

	response := entities.CreateMovementResponse{
		Movement: toMovementResponse(stored, nil),
		Filament: *summary,
	}

	c.JSON(http.StatusCreated, response)

	uc.activityService.Record(c.Request.Context(), activityEntities.ActivityEntity{
		OrganizationID: organizationID,
		UserID:         userID,
		Action:         activityEntities.ActionUpdated,
		EntityType:     activityEntities.EntityStockMovement,
		EntityID:       filamentID.String(),
		Metadata: map[string]any{
			"type":  string(stored.Type),
			"grams": stored.Grams,
		},
		CreatedAt: time.Now(),
	})
}

// ListMovements returns the paginated movement ledger for a filament.
// @Summary List filament stock movements
// @Description List the stock movement ledger for a filament, newest first, paginated and filterable by type.
// @Tags Filament Stock
// @Accept json
// @Produce json
// @Param id path string true "Filament ID (UUID)"
// @Param page query int false "Page number (1-based)" default(1)
// @Param page_size query int false "Items per page (max 100)" default(20)
// @Param type query string false "Filter by movement type" Enums(purchase, adjustment, waste, consumption)
// @Param sort_by query string false "Sort field" Enums(created_at) default(created_at)
// @Param sort_dir query string false "Sort direction" Enums(asc, desc) default(desc)
// @Success 200 {object} entities.ListMovementsResponse "Paginated movement ledger"
// @Failure 400 {object} errors.APIError
// @Failure 401 {object} errors.APIError
// @Failure 404 {object} errors.APIError
// @Failure 500 {object} errors.APIError
// @Security BearerAuth
// @Router /filaments/{id}/stock-movements [get]
func (uc *StockUseCase) ListMovements(c *gin.Context) {
	ctx := c.Request.Context()

	organizationID := helpers.GetOrganizationID(c)
	if organizationID == "" {
		uc.logger.Error(ctx, "Organization ID not found", nil)
		coreErrors.Respond(c, coreErrors.BadRequest("organization_required", "Organização não encontrada no contexto"))
		return
	}

	filamentID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		uc.logger.Error(ctx, "Invalid filament ID", map[string]interface{}{"filament_id": c.Param("id")})
		coreErrors.Respond(c, coreErrors.BadRequest("invalid_filament_id", "ID de filamento inválido"))
		return
	}

	// 404 when the filament is not in the caller's organization.
	exists, err := uc.repository.FilamentExistsInOrg(ctx, filamentID, organizationID)
	if err != nil {
		uc.logger.Error(ctx, "Failed to verify filament", map[string]interface{}{"error": err.Error()})
		coreErrors.Respond(c, err)
		return
	}
	if !exists {
		coreErrors.Respond(c, coreErrors.NotFoundErr("filament_not_found", "Filamento não encontrado"))
		return
	}

	q := helpers.ParseListQuery(c, helpers.ListQueryOptions{
		DefaultPageSize: 20,
		SortWhitelist:   movementSortWhitelist,
		DefaultSort:     "created_at",
		TieBreaker:      "m.id",
	})

	typeFilter := c.Query("type")
	if typeFilter != "" && !entities.MovementType(typeFilter).IsKnown() {
		coreErrors.Respond(c, coreErrors.BadRequest(entities.CodeInvalidMovementType, "Tipo de movimentação inválido"))
		return
	}

	movements, total, err := uc.repository.ListMovements(ctx, filamentID, organizationID, typeFilter, q.OrderClause(), q.Limit(), q.Offset())
	if err != nil {
		uc.logger.Error(ctx, "Failed to list stock movements", map[string]interface{}{"error": err.Error()})
		coreErrors.Respond(c, err)
		return
	}

	c.JSON(http.StatusOK, helpers.NewPage(movements, total, q))
}

// toMovementResponse maps a stored movement entity to its wire shape. quoteNumber is
// the linked budget's quote number when known (nil for manual movements).
func toMovementResponse(m *entities.StockMovementEntity, quoteNumber *int) entities.StockMovementResponse {
	return entities.StockMovementResponse{
		ID:                m.ID,
		FilamentID:        m.FilamentID,
		Type:              m.Type,
		Grams:             m.Grams,
		UnitPricePerKg:    m.UnitPricePerKg,
		BudgetID:          m.BudgetID,
		BudgetQuoteNumber: quoteNumber,
		Note:              m.Note,
		CreatedBy:         m.CreatedBy,
		CreatedAt:         m.CreatedAt,
	}
}
