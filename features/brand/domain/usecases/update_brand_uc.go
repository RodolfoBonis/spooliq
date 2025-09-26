package usecases

import (
	"errors"
	"strings"

	coreErrors "github.com/RodolfoBonis/spooliq/core/errors"
	"github.com/RodolfoBonis/spooliq/core/logger"
	"github.com/RodolfoBonis/spooliq/features/brand/domain/entities"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"go.opentelemetry.io/otel"
)

// Update handles updating an existing brand.
// @Summary Update Brand
// @Schemes
// @Description Update an existing filament brand
// @Tags Brands
// @Accept json
// @Produce json
// @Param id path string true "Brand ID" format(uuid)
// @Param request body entities.UpsertBrandRequestEntity true "Brand data"
// @Success 200 {object} entities.BrandEntity "Successfully updated brand"
// @Failure 400 {object} errors.HTTPError
// @Failure 401 {object} errors.HTTPError
// @Failure 404 {object} errors.HTTPError
// @Failure 409 {object} errors.HTTPError
// @Failure 500 {object} errors.HTTPError
// @Router /brands/{id} [put]
// @Security Bearer
func (uc *BrandUseCase) Update(c *gin.Context) {
	// Create custom span for brand update
	tracer := otel.Tracer("brand-service")
	ctx, span := logger.StartSpanWithLogger(c.Request.Context(), tracer, "brand.update", uc.logger)
	var spanErr error
	defer func() {
		logger.EndSpanWithLogger(span, uc.logger, spanErr)
	}()

	idParam := c.Param("id")
	id, err := uuid.Parse(idParam)

	if err != nil {
		spanErr = err

		// Add trace context to log
		fields := logger.AddTraceToContext(ctx)
		fields["brand_id"] = idParam
		fields["error"] = err.Error()
		uc.logger.Error(ctx, "Invalid brand ID", fields)

		appError := coreErrors.UsecaseError(err.Error())
		httpError := appError.ToHTTPError()
		c.AbortWithStatusJSON(httpError.StatusCode, httpError)
		return
	}

	var request entities.UpsertBrandRequestEntity

	if err := c.BindJSON(&request); err != nil {
		appError := coreErrors.UsecaseError(err.Error())
		httpError := appError.ToHTTPError()
		uc.logger.LogError(ctx, httpError.Message, err)
		c.AbortWithStatusJSON(httpError.StatusCode, httpError)
		return
	}

	if err := uc.validator.Struct(request); err != nil {
		appError := coreErrors.UsecaseError(err.Error())
		httpError := appError.ToHTTPError()
		uc.logger.LogError(ctx, httpError.Message, err)
		c.AbortWithStatusJSON(httpError.StatusCode, httpError)
		return
	}

	brand, err := uc.repository.FindByID(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) || strings.Contains(err.Error(), "not found") {
			appError := coreErrors.UsecaseError("Brand not found")
			httpError := appError.ToHTTPError()
			uc.logger.LogError(ctx, httpError.Message, err)
			c.AbortWithStatusJSON(httpError.StatusCode, httpError)
			return
		}
		uc.logger.Error(ctx, "Failed to get brand for update", map[string]interface{}{
			"brand_id": id,
			"error":    err.Error(),
		})
		appError := coreErrors.UsecaseError(err.Error())
		httpError := appError.ToHTTPError()
		uc.logger.LogError(ctx, httpError.Message, err)
		c.AbortWithStatusJSON(httpError.StatusCode, httpError)
		return
	}

	// Check if another brand with the same name exists (excluding current brand)
	if request.Name != brand.Name {
		exists, err := uc.repository.Exists(request.Name)
		if err != nil {
			uc.logger.Error(ctx, "Failed to check brand name existence", map[string]interface{}{
				"name":  request.Name,
				"error": err.Error(),
			})
			appError := coreErrors.UsecaseError(err.Error())
			httpError := appError.ToHTTPError()
			c.AbortWithStatusJSON(httpError.StatusCode, httpError)
			return
		}

		if exists {
			appError := coreErrors.UsecaseError("Brand with this name already exists")
			httpError := appError.ToHTTPError()
			uc.logger.LogError(ctx, httpError.Message, nil)
			c.AbortWithStatusJSON(httpError.StatusCode, httpError)
			return
		}
	}

	// Update brand fields
	brand.Name = request.Name
	brand.Description = request.Description

	// Save updated brand
	if err := uc.repository.Update(brand); err != nil {
		uc.logger.Error(ctx, "Failed to update brand", map[string]interface{}{
			"brand_id": id,
			"error":    err.Error(),
		})
		appError := coreErrors.UsecaseError("Failed to update brand")
		httpError := appError.ToHTTPError()
		uc.logger.LogError(ctx, httpError.Message, err)
		c.AbortWithStatusJSON(httpError.StatusCode, httpError)
		return
	}

	// Log successful update with trace context
	fields := logger.AddTraceToContext(ctx)
	fields["brand_id"] = brand.ID
	fields["brand_name"] = brand.Name
	uc.logger.Info(ctx, "Brand updated successfully", fields)

	c.JSON(200, brand)
}
