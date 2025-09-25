package usecases

import (
	"net/http"

	"github.com/RodolfoBonis/spooliq/core/errors"
	"github.com/RodolfoBonis/spooliq/core/logger"
	"github.com/RodolfoBonis/spooliq/features/brand/domain/entities"
	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/otel"
)

// Create handles creating a new brand.
// @Summary Create Brand
// @Schemes
// @Description Create a new filament brand
// @Tags Brands
// @Accept json
// @Produce json
// @Param request body entities.UpsertBrandRequestEntity true "Brand data"
// @Success 201 {object} entities.BrandEntity "Successfully created brand"
// @Failure 400 {object} errors.HTTPError
// @Failure 401 {object} errors.HTTPError
// @Failure 409 {object} errors.HTTPError
// @Failure 500 {object} errors.HTTPError
// @Router /brands [post]
// @Security Bearer
func (uc *BrandUseCase) Create(c *gin.Context) {
	// Create custom span for brand creation
	tracer := otel.Tracer("brand-service")
	ctx, span := logger.StartSpanWithLogger(c.Request.Context(), tracer, "brand.create", uc.logger)
	var spanErr error
	defer func() {
		logger.EndSpanWithLogger(span, uc.logger, spanErr)
	}()

	var request entities.UpsertBrandRequestEntity
	if err := c.ShouldBindJSON(&request); err != nil {
		spanErr = err
		appError := errors.UsecaseError(err.Error())
		httpError := appError.ToHTTPError()

		// Add trace context to log
		fields := logger.AddTraceToContext(ctx)
		fields["error"] = err.Error()
		uc.logger.Error(ctx, httpError.Message, fields)

		c.AbortWithStatusJSON(httpError.StatusCode, httpError)
		return
	}

	if err := uc.validator.Struct(request); err != nil {
		spanErr = err
		appError := errors.UsecaseError(err.Error())
		httpError := appError.ToHTTPError()

		// Add trace context to log
		fields := logger.AddTraceToContext(ctx)
		fields["error"] = err.Error()
		fields["validation_failed"] = true
		uc.logger.Error(ctx, httpError.Message, fields)

		c.AbortWithStatusJSON(httpError.StatusCode, httpError)
		return
	}

	exists, err := uc.repository.Exists(request.Name)
	if err != nil {
		spanErr = err

		// Add trace context to log
		fields := logger.AddTraceToContext(ctx)
		fields["name"] = request.Name
		fields["error"] = err.Error()
		fields["operation"] = "check_brand_existence"
		uc.logger.Error(ctx, "Failed to check brand existence", fields)

		appError := errors.UsecaseError(err.Error())
		httpError := appError.ToHTTPError()
		c.AbortWithStatusJSON(httpError.StatusCode, httpError)
		return
	}

	if exists {
		httpError := errors.NewHTTPError(http.StatusConflict, "Brand with this name already exists")

		// Add trace context to log
		fields := logger.AddTraceToContext(ctx)
		fields["name"] = request.Name
		fields["conflict"] = true
		uc.logger.Warning(ctx, httpError.Message, fields)

		c.AbortWithStatusJSON(httpError.StatusCode, httpError)
		return
	}

	brand := &entities.BrandEntity{
		Name:        request.Name,
		Description: request.Description,
	}

	if err := uc.repository.Create(brand); err != nil {
		spanErr = err

		// Add trace context to log
		fields := logger.AddTraceToContext(ctx)
		fields["name"] = request.Name
		fields["error"] = err.Error()
		fields["operation"] = "create_brand"
		uc.logger.Error(ctx, "Failed to create brand", fields)

		httpError := errors.NewHTTPError(http.StatusInternalServerError, "Failed to create brand")
		c.AbortWithStatusJSON(httpError.StatusCode, httpError)
		return
	}

	// Log successful creation with trace context
	fields := logger.AddTraceToContext(ctx)
	fields["brand_id"] = brand.ID
	fields["brand_name"] = brand.Name
	uc.logger.Info(ctx, "Brand created successfully", fields)

	c.JSON(http.StatusCreated, brand)
}
