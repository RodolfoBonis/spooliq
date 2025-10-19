package plans

import (
	"net/http"

	"github.com/RodolfoBonis/spooliq/core/errors"
	"github.com/RodolfoBonis/spooliq/core/logger"
	"github.com/RodolfoBonis/spooliq/features/subscriptions/domain/entities"
	"github.com/RodolfoBonis/spooliq/features/subscriptions/domain/repositories"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// CreatePlanRequest represents the request to create a plan
type CreatePlanRequest struct {
	Name        string                 `json:"name" binding:"required"`
	Slug        string                 `json:"slug" binding:"required"`
	Description string                 `json:"description"`
	Price       float64                `json:"price" binding:"required,min=0"`
	Currency    string                 `json:"currency"`
	Interval    string                 `json:"interval" binding:"required,oneof=MONTHLY YEARLY"`
	Active      bool                   `json:"active"`
	Popular     bool                   `json:"popular"`
	Recommended bool                   `json:"recommended"`
	SortOrder   int                    `json:"sort_order"`
	Features    []CreateFeatureRequest `json:"features"`
}

// CreateFeatureRequest represents a feature in the create request
type CreateFeatureRequest struct {
	Name        string `json:"name" binding:"required"`
	Key         string `json:"key" binding:"required"`
	Description string `json:"description"`
	Value       string `json:"value" binding:"required"`
	ValueType   string `json:"value_type" binding:"required,oneof=number boolean text"`
	Available   bool   `json:"available"`
	SortOrder   int    `json:"sort_order"`
}

// CreatePlanUseCase handles plan creation by admin
type CreatePlanUseCase struct {
	planRepository repositories.PlanRepository
	logger         logger.Logger
}

// NewCreatePlanUseCase creates a new instance
func NewCreatePlanUseCase(
	planRepository repositories.PlanRepository,
	logger logger.Logger,
) *CreatePlanUseCase {
	return &CreatePlanUseCase{
		planRepository: planRepository,
		logger:         logger,
	}
}

// Execute creates a new plan
func (uc *CreatePlanUseCase) Execute(c *gin.Context) {
	ctx := c.Request.Context()
	var req CreatePlanRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		uc.logger.Error(ctx, "Invalid request body", map[string]interface{}{
			"error": err.Error(),
		})
		c.JSON(http.StatusBadRequest, errors.NewHTTPError(http.StatusBadRequest, "Dados inválidos: "+err.Error()))
		return
	}

	// Set default currency if not provided
	if req.Currency == "" {
		req.Currency = "BRL"
	}

	// Create plan entity
	plan := &entities.PlanEntity{
		ID:          uuid.New(),
		Name:        req.Name,
		Slug:        req.Slug,
		Description: req.Description,
		Price:       req.Price,
		Currency:    req.Currency,
		Interval:    req.Interval,
		Active:      req.Active,
		Popular:     req.Popular,
		Recommended: req.Recommended,
		SortOrder:   req.SortOrder,
	}

	// Add features
	features := make([]entities.PlanFeature, len(req.Features))
	for i, f := range req.Features {
		features[i] = entities.PlanFeature{
			ID:          uuid.New(),
			Name:        f.Name,
			Key:         f.Key,
			Description: f.Description,
			Value:       f.Value,
			ValueType:   f.ValueType,
			Available:   f.Available,
			SortOrder:   f.SortOrder,
		}
	}
	plan.Features = features

	// Save to database
	if err := uc.planRepository.Create(ctx, plan); err != nil {
		uc.logger.Error(ctx, "Failed to create plan", map[string]interface{}{
			"error": err.Error(),
			"slug":  req.Slug,
		})
		c.JSON(http.StatusInternalServerError, errors.NewHTTPError(http.StatusInternalServerError, "Erro ao criar plano"))
		return
	}

	uc.logger.Info(ctx, "Plan created successfully", map[string]interface{}{
		"plan_id": plan.ID,
		"slug":    plan.Slug,
	})

	c.JSON(http.StatusCreated, plan)
}
