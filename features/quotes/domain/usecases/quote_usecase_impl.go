package usecases

import (
	"net/http"
	"strconv"
	"time"

	"github.com/RodolfoBonis/spooliq/core/entities"
	"github.com/RodolfoBonis/spooliq/core/errors"
	"github.com/RodolfoBonis/spooliq/core/logger"
	calculationEntities "github.com/RodolfoBonis/spooliq/features/calculation/domain/entities"
	calculationServices "github.com/RodolfoBonis/spooliq/features/calculation/domain/services"
	"github.com/RodolfoBonis/spooliq/features/quotes/data/mappers"
	quotesEntities "github.com/RodolfoBonis/spooliq/features/quotes/domain/entities"
	"github.com/RodolfoBonis/spooliq/features/quotes/domain/repositories"
	"github.com/RodolfoBonis/spooliq/features/quotes/domain/services"
	"github.com/RodolfoBonis/spooliq/features/quotes/presentation/dto"
	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
)

type quoteUseCaseImpl struct {
	quoteRepo             repositories.QuoteRepository
	calculationService    calculationServices.CalculationService
	snapshotService       services.SnapshotService
	energyProfileService  services.EnergyProfileService
	machineProfileService services.MachineProfileService
	costProfileService    services.CostProfileService
	marginProfileService  services.MarginProfileService
	logger                logger.Logger
	validator             *validator.Validate
}

// NewQuoteUseCase creates a new instance of QuoteUseCase with the provided dependencies.
func NewQuoteUseCase(
	quoteRepo repositories.QuoteRepository,
	calculationService calculationServices.CalculationService,
	snapshotService services.SnapshotService,
	energyProfileService services.EnergyProfileService,
	machineProfileService services.MachineProfileService,
	costProfileService services.CostProfileService,
	marginProfileService services.MarginProfileService,
	logger logger.Logger,
) QuoteUseCase {
	return &quoteUseCaseImpl{
		quoteRepo:             quoteRepo,
		calculationService:    calculationService,
		snapshotService:       snapshotService,
		energyProfileService:  energyProfileService,
		machineProfileService: machineProfileService,
		costProfileService:    costProfileService,
		marginProfileService:  marginProfileService,
		logger:                logger,
		validator:             validator.New(),
	}
}

func (uc *quoteUseCaseImpl) CreateQuote(c *gin.Context) {
	var req dto.CreateQuoteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		appError := errors.NewAppError(entities.ErrEntity, "Formato de requisição inválido", nil, err)
		httpError := appError.ToHTTPError()
		uc.logger.LogError(c.Request.Context(), "Failed to bind create quote request", appError)
		c.JSON(httpError.StatusCode, httpError)
		return
	}

	if err := uc.validator.Struct(&req); err != nil {
		appError := errors.NewAppError(entities.ErrEntity, "Falha na validação dos dados", nil, err)
		httpError := appError.ToHTTPError()
		uc.logger.LogError(c.Request.Context(), "Validation failed for create quote", appError)
		c.JSON(httpError.StatusCode, httpError)
		return
	}

	// Get user ID from context (set by auth middleware)
	userID := c.GetString("user_id")
	if userID == "" {
		appError := errors.NewAppError(entities.ErrUnauthorized, "Usuário não autenticado", nil, nil)
		httpError := appError.ToHTTPError()
		uc.logger.LogError(c.Request.Context(), "User not authenticated", appError)
		c.JSON(httpError.StatusCode, httpError)
		return
	}

	// Convert DTO to entity (without filament lines and energy profile)
	quote := mappers.CreateRequestToEntity(&req, userID)

	// Process filament lines with snapshot service
	if len(req.FilamentLines) > 0 {
		quote.FilamentLines = make([]quotesEntities.QuoteFilamentLine, 0, len(req.FilamentLines))
		for _, lineReq := range req.FilamentLines {
			// Use SnapshotService to create filament line (handles both automatic and manual snapshots)
			line, err := uc.snapshotService.CreateFilamentSnapshot(c.Request.Context(), &lineReq, userID)
			if err != nil {
				appError := errors.NewAppError(entities.ErrEntity, "Erro ao criar snapshot de filamento", nil, err)
				httpError := appError.ToHTTPError()
				uc.logger.LogError(c.Request.Context(), "Failed to create filament snapshot", appError)
				c.JSON(httpError.StatusCode, httpError)
				return
			}

			quote.FilamentLines = append(quote.FilamentLines, *line)
		}
	}

	// Process energy profile with energy profile service (handles presets and custom data)
	if req.EnergyProfile != nil {
		energyProfile, err := uc.energyProfileService.CreateEnergyProfileFromRequest(c.Request.Context(), req.EnergyProfile, userID)
		if err != nil {
			appError := errors.NewAppError(entities.ErrEntity, "Erro ao criar perfil de energia", nil, err)
			httpError := appError.ToHTTPError()
			uc.logger.LogError(c.Request.Context(), "Failed to create energy profile", appError)
			c.JSON(httpError.StatusCode, httpError)
			return
		}
		quote.EnergyProfile = energyProfile
	}

	// Process machine profile with machine profile service (handles presets and custom data)
	if req.MachineProfile != nil {
		machineProfile, err := uc.machineProfileService.CreateMachineProfileFromRequest(c.Request.Context(), req.MachineProfile)
		if err != nil {
			appError := errors.NewAppError(entities.ErrEntity, "Erro ao criar perfil de máquina", nil, err)
			httpError := appError.ToHTTPError()
			uc.logger.LogError(c.Request.Context(), "Failed to create machine profile", appError)
			c.JSON(httpError.StatusCode, httpError)
			return
		}
		quote.MachineProfile = machineProfile
	}

	// Process cost profile with cost profile service (handles presets and custom data)
	if req.CostProfile != nil {
		costProfile, err := uc.costProfileService.CreateCostProfileFromRequest(c.Request.Context(), req.CostProfile)
		if err != nil {
			appError := errors.NewAppError(entities.ErrEntity, "Erro ao criar perfil de custos", nil, err)
			httpError := appError.ToHTTPError()
			uc.logger.LogError(c.Request.Context(), "Failed to create cost profile", appError)
			c.JSON(httpError.StatusCode, httpError)
			return
		}
		quote.CostProfile = costProfile
	}

	// Process margin profile with margin profile service (handles presets and custom data)
	if req.MarginProfile != nil {
		marginProfile, err := uc.marginProfileService.CreateMarginProfileFromRequest(c.Request.Context(), req.MarginProfile)
		if err != nil {
			appError := errors.NewAppError(entities.ErrEntity, "Erro ao criar perfil de margens", nil, err)
			httpError := appError.ToHTTPError()
			uc.logger.LogError(c.Request.Context(), "Failed to create margin profile", appError)
			c.JSON(httpError.StatusCode, httpError)
			return
		}
		quote.MarginProfile = marginProfile
	}

	// Validate business rules
	if !quote.IsValid() {
		appError := errors.NewAppError(entities.ErrEntity, "Dados do orçamento inválidos", nil, nil)
		httpError := appError.ToHTTPError()
		uc.logger.LogError(c.Request.Context(), "Quote business validation failed", appError)
		c.JSON(httpError.StatusCode, httpError)
		return
	}

	// Create quote
	if err := uc.quoteRepo.Create(c.Request.Context(), quote); err != nil {
		appError := errors.NewAppError(entities.ErrRepository, "Erro ao criar orçamento", nil, err)
		httpError := appError.ToHTTPError()
		uc.logger.LogError(c.Request.Context(), "Failed to create quote", appError)
		c.JSON(httpError.StatusCode, httpError)
		return
	}

	// Convert to response DTO
	response := dto.ToQuoteResponse(quote)

	uc.logger.Info(c.Request.Context(), "Quote created successfully", map[string]interface{}{
		"quote_id": quote.ID,
		"user_id":  userID,
	})

	c.JSON(http.StatusCreated, response)
}

func (uc *quoteUseCaseImpl) GetQuote(c *gin.Context) {
	idParam := c.Param("id")
	id, err := strconv.ParseUint(idParam, 10, 32)
	if err != nil {
		appError := errors.NewAppError(entities.ErrEntity, "ID inválido", nil, err)
		httpError := appError.ToHTTPError()
		uc.logger.LogError(c.Request.Context(), "Invalid quote ID", appError)
		c.JSON(httpError.StatusCode, httpError)
		return
	}

	userID := c.GetString("user_id")
	if userID == "" {
		appError := errors.NewAppError(entities.ErrUnauthorized, "Usuário não autenticado", nil, nil)
		httpError := appError.ToHTTPError()
		uc.logger.LogError(c.Request.Context(), "User not authenticated", appError)
		c.JSON(httpError.StatusCode, httpError)
		return
	}

	quote, err := uc.quoteRepo.GetByID(c.Request.Context(), uint(id), userID)
	if err != nil {
		appError := errors.NewAppError(entities.ErrNotFound, "Orçamento não encontrado", nil, err)
		httpError := appError.ToHTTPError()
		uc.logger.LogError(c.Request.Context(), "Failed to get quote", appError)
		c.JSON(httpError.StatusCode, httpError)
		return
	}

	// Convert to response DTO
	response := dto.ToQuoteResponse(quote)

	c.JSON(http.StatusOK, response)
}

func (uc *quoteUseCaseImpl) GetUserQuotes(c *gin.Context) {
	userID := c.GetString("user_id")
	if userID == "" {
		appError := errors.NewAppError(entities.ErrUnauthorized, "Usuário não autenticado", nil, nil)
		httpError := appError.ToHTTPError()
		uc.logger.LogError(c.Request.Context(), "User not authenticated", appError)
		c.JSON(httpError.StatusCode, httpError)
		return
	}

	quotes, err := uc.quoteRepo.GetByUser(c.Request.Context(), userID)
	if err != nil {
		appError := errors.NewAppError(entities.ErrRepository, "Erro ao buscar orçamentos do usuário", nil, err)
		httpError := appError.ToHTTPError()
		uc.logger.LogError(c.Request.Context(), "Failed to get user quotes", appError)
		c.JSON(httpError.StatusCode, httpError)
		return
	}

	// Convert to response DTOs
	responses := make([]*dto.QuoteResponse, 0, len(quotes))
	for _, quote := range quotes {
		responses = append(responses, dto.ToQuoteResponse(quote))
	}

	c.JSON(http.StatusOK, gin.H{"quotes": responses})
}

func (uc *quoteUseCaseImpl) UpdateQuote(c *gin.Context) {
	// Implementation similar to CreateQuote but with update logic
	// ... (truncated for brevity)
	c.JSON(http.StatusNotImplemented, gin.H{"message": "UpdateQuote not implemented yet"})
}

func (uc *quoteUseCaseImpl) DeleteQuote(c *gin.Context) {
	idParam := c.Param("id")
	id, err := strconv.ParseUint(idParam, 10, 32)
	if err != nil {
		appError := errors.NewAppError(entities.ErrEntity, "ID inválido", nil, err)
		httpError := appError.ToHTTPError()
		uc.logger.LogError(c.Request.Context(), "Invalid quote ID", appError)
		c.JSON(httpError.StatusCode, httpError)
		return
	}

	userID := c.GetString("user_id")
	if userID == "" {
		appError := errors.NewAppError(entities.ErrUnauthorized, "Usuário não autenticado", nil, nil)
		httpError := appError.ToHTTPError()
		uc.logger.LogError(c.Request.Context(), "User not authenticated", appError)
		c.JSON(httpError.StatusCode, httpError)
		return
	}

	if err := uc.quoteRepo.Delete(c.Request.Context(), uint(id), userID); err != nil {
		appError := errors.NewAppError(entities.ErrRepository, "Erro ao deletar orçamento", nil, err)
		httpError := appError.ToHTTPError()
		uc.logger.LogError(c.Request.Context(), "Failed to delete quote", appError)
		c.JSON(httpError.StatusCode, httpError)
		return
	}

	uc.logger.Info(c.Request.Context(), "Quote deleted successfully", map[string]interface{}{
		"quote_id": id,
		"user_id":  userID,
	})

	c.JSON(http.StatusNoContent, nil)
}

func (uc *quoteUseCaseImpl) DuplicateQuote(c *gin.Context) {
	// Implementation for duplicating quotes
	// ... (truncated for brevity)
	c.JSON(http.StatusNotImplemented, gin.H{"message": "DuplicateQuote not implemented yet"})
}

func (uc *quoteUseCaseImpl) CalculateQuote(c *gin.Context) {
	idParam := c.Param("id")
	id, err := strconv.ParseUint(idParam, 10, 32)
	if err != nil {
		appError := errors.NewAppError(entities.ErrEntity, "ID inválido", nil, err)
		httpError := appError.ToHTTPError()
		uc.logger.LogError(c.Request.Context(), "Invalid quote ID", appError)
		c.JSON(httpError.StatusCode, httpError)
		return
	}

	var req dto.CalculateQuoteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		appError := errors.NewAppError(entities.ErrEntity, "Formato de requisição inválido", nil, err)
		httpError := appError.ToHTTPError()
		uc.logger.LogError(c.Request.Context(), "Failed to bind calculate request", appError)
		c.JSON(httpError.StatusCode, httpError)
		return
	}

	if err := uc.validator.Struct(&req); err != nil {
		appError := errors.NewAppError(entities.ErrEntity, "Falha na validação dos dados", nil, err)
		httpError := appError.ToHTTPError()
		uc.logger.LogError(c.Request.Context(), "Validation failed for calculate request", appError)
		c.JSON(httpError.StatusCode, httpError)
		return
	}

	userID := c.GetString("user_id")
	if userID == "" {
		appError := errors.NewAppError(entities.ErrUnauthorized, "Usuário não autenticado", nil, nil)
		httpError := appError.ToHTTPError()
		uc.logger.LogError(c.Request.Context(), "User not authenticated", appError)
		c.JSON(httpError.StatusCode, httpError)
		return
	}

	// Get quote with all data
	quote, err := uc.quoteRepo.GetWithFilamentLines(c.Request.Context(), uint(id), userID)
	if err != nil {
		appError := errors.NewAppError(entities.ErrNotFound, "Orçamento não encontrado", nil, err)
		httpError := appError.ToHTTPError()
		uc.logger.LogError(c.Request.Context(), "Failed to get quote for calculation", appError)
		c.JSON(httpError.StatusCode, httpError)
		return
	}

	// Prepare calculation input
	calculationInput := calculationEntities.CostBreakdown{}

	// Convert filament lines to calculation entities
	for _, line := range quote.FilamentLines {
		filament := calculationEntities.FilamentLineInput{
			Label:      line.FilamentSnapshotName + " " + line.FilamentSnapshotColor,
			Grams:      line.WeightGrams,
			PricePerKg: line.FilamentSnapshotPricePerKg,
		}
		calculationInput.Filaments = append(calculationInput.Filaments, filament)
	}

	// Set machine input
	if quote.MachineProfile != nil {
		calculationInput.Machine = calculationEntities.MachineInput{
			Name:         quote.MachineProfile.Name,
			Watt:         quote.MachineProfile.Watt,
			IdleFactor:   quote.MachineProfile.IdleFactor,
			HoursDecimal: req.PrintTimeHours,
		}
	}

	// Set energy input
	if quote.EnergyProfile != nil {
		calculationInput.Energy = calculationEntities.EnergyInput{
			BaseTariff:    quote.EnergyProfile.BaseTariff,
			FlagSurcharge: quote.EnergyProfile.FlagSurcharge,
		}
	}

	// Set cost input
	calculationInput.Costs = calculationEntities.CostInput{
		WearPct:        quote.CostProfile.WearPercentage,
		Overhead:       quote.CostProfile.OverheadAmount,
		OpRatePerHour:  quote.MarginProfile.OperatorRatePerHour,
		OpMinutes:      req.OperatorMinutes,
		CadRatePerHour: quote.MarginProfile.ModelerRatePerHour,
		CadMinutes:     req.ModelerMinutes,
	}

	// Set margin input based on service type
	appliedMargin := 0.0
	if quote.MarginProfile != nil {
		appliedMargin = quote.MarginProfile.GetMarginByServiceType(req.ServiceType)
	}

	calculationInput.Margins = calculationEntities.MarginInput{
		OnlyPrintPct:     appliedMargin,
		LightAdjustPct:   appliedMargin,
		FullModelPct:     appliedMargin,
		ExtraCadLightMin: 0.0,
		ExtraCadFullMin:  0.0,
	}

	// Perform calculation
	result, err := uc.calculationService.Calculate(c.Request.Context(), calculationInput)
	if err != nil {
		appError := errors.NewAppError(entities.ErrService, "Erro ao calcular orçamento", nil, err)
		httpError := appError.ToHTTPError()
		uc.logger.LogError(c.Request.Context(), "Failed to calculate quote", appError)
		c.JSON(httpError.StatusCode, httpError)
		return
	}

	// Get margin based on service type (already calculated above)

	// Convert to response DTO
	response := dto.CalculationResult{
		MaterialCost:    result.Results.MaterialsCost,
		EnergyCost:      result.Results.EnergyCost,
		WearCost:        result.Results.WearCost,
		LaborCost:       result.Results.LaborCost,
		DirectCost:      result.Results.DirectCost,
		FinalPrice:      result.Results.DirectCost * (1 + appliedMargin/100),
		PrintTimeHours:  req.PrintTimeHours,
		OperatorMinutes: req.OperatorMinutes,
		ModelerMinutes:  req.ModelerMinutes,
		ServiceType:     req.ServiceType,
		AppliedMargin:   appliedMargin,
	}

	// Save calculation result to quote
	now := time.Now()
	quote.CalculationMaterialCost = &response.MaterialCost
	quote.CalculationEnergyCost = &response.EnergyCost
	quote.CalculationWearCost = &response.WearCost
	quote.CalculationLaborCost = &response.LaborCost
	quote.CalculationDirectCost = &response.DirectCost
	quote.CalculationFinalPrice = &response.FinalPrice
	quote.CalculationPrintTimeHours = &response.PrintTimeHours
	quote.CalculationOperatorMinutes = &response.OperatorMinutes
	quote.CalculationModelerMinutes = &response.ModelerMinutes
	quote.CalculationServiceType = &response.ServiceType
	quote.CalculationAppliedMargin = &response.AppliedMargin
	quote.CalculationCalculatedAt = &now

	// Update quote in database with calculation results
	if err := uc.quoteRepo.Update(c.Request.Context(), quote, userID); err != nil {
		uc.logger.Error(c.Request.Context(), "Failed to save calculation results", map[string]interface{}{
			"quote_id": id,
			"error":    err.Error(),
		})
		// Don't fail the request if we can't save - just log the error
	}

	uc.logger.Info(c.Request.Context(), "Quote calculated successfully", map[string]interface{}{
		"quote_id":     id,
		"user_id":      userID,
		"final_price":  response.FinalPrice,
		"service_type": req.ServiceType,
	})

	c.JSON(http.StatusOK, response)
}
