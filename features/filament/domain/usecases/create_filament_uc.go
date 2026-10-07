package usecases

import (
	"context"
	"errors"
	"net/http"
	"time"

	coreErrors "github.com/RodolfoBonis/spooliq/core/errors"
	"github.com/RodolfoBonis/spooliq/core/helpers"
	"github.com/RodolfoBonis/spooliq/core/roles"
	"github.com/RodolfoBonis/spooliq/core/validation"
	activityEntities "github.com/RodolfoBonis/spooliq/features/activity/domain/entities"
	filamentEntities "github.com/RodolfoBonis/spooliq/features/filament/domain/entities"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Create handles creating a new filament.
// @Summary Create Filament
// @Description Create a new 3D printing filament
// @Tags Filaments
// @Accept json
// @Produce json
// @Param request body filamentEntities.CreateFilamentRequest true "Filament data"
// @Success 201 {object} filamentEntities.FilamentResponse "Successfully created filament"
// @Failure 400 {object} errors.APIError
// @Failure 401 {object} errors.APIError
// @Failure 409 {object} errors.APIError
// @Failure 500 {object} errors.APIError
// @Router /filaments [post]
// @Security BearerAuth
func (uc *FilamentUseCase) Create(c *gin.Context) {
	ctx := c.Request.Context()

	organizationID := helpers.GetOrganizationID(c)
	if organizationID == "" {
		uc.logger.Error(ctx, "Organization ID not found in context", nil)
		coreErrors.Respond(c, coreErrors.BadRequest("organization_required", "Organização não encontrada no contexto"))
		return
	}

	userIDStr := helpers.GetUserID(c)
	if userIDStr == "" {
		uc.logger.Error(ctx, "User ID not found in context", nil)
		coreErrors.Respond(c, coreErrors.BadRequest("user_required", "Usuário não encontrado no contexto"))
		return
	}

	userRole, _ := c.Get("user_role")
	userRoleStr, _ := userRole.(string)
	isAdmin := userRoleStr == roles.OrgAdminRole

	var request filamentEntities.CreateFilamentRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		uc.logger.Error(ctx, "Invalid filament creation payload", map[string]interface{}{"error": err.Error()})
		coreErrors.Respond(c, err)
		return
	}

	if err := validation.Validate(request); err != nil {
		uc.logger.Error(ctx, "Filament validation failed", map[string]interface{}{"error": err.Error()})
		coreErrors.Respond(c, err)
		return
	}

	// Validate color data if provided
	if request.ColorType != "" && len(request.ColorData) > 0 {
		if !request.ColorType.IsValid() {
			uc.logger.Error(ctx, "Invalid color type", map[string]interface{}{"color_type": request.ColorType})
			coreErrors.Respond(c, coreErrors.BadRequest("invalid_color_type", "Tipo de cor inválido"))
			return
		}
		if _, err := filamentEntities.ParseColorData(request.ColorType, request.ColorData); err != nil {
			uc.logger.Error(ctx, "Invalid color data", map[string]interface{}{"error": err.Error(), "color_type": request.ColorType})
			coreErrors.Respond(c, coreErrors.BadRequest("invalid_color_data", "Dados de cor inválidos"))
			return
		}
	}

	// The referenced brand and material must belong to the caller's organization.
	if err := uc.validateBrandAndMaterial(ctx, organizationID, request.BrandID, request.MaterialID); err != nil {
		uc.logger.Warning(ctx, "Filament references brand/material outside organization", map[string]interface{}{"brand_id": request.BrandID, "material_id": request.MaterialID})
		coreErrors.Respond(c, err)
		return
	}

	exists, err := uc.repository.ExistsByNameAndBrand(ctx, request.Name, request.BrandID, organizationID, nil)
	if err != nil {
		uc.logger.Error(ctx, "Failed to check filament existence", map[string]interface{}{"name": request.Name, "error": err.Error()})
		coreErrors.Respond(c, err)
		return
	}

	if exists {
		uc.logger.Warning(ctx, "Filament creation failed: already exists", map[string]interface{}{"name": request.Name})
		coreErrors.Respond(c, coreErrors.Conflict("filament_name_taken", "Já existe um filamento com este nome para esta marca"))
		return
	}

	filament := &filamentEntities.FilamentEntity{
		ID:               uuid.New(),
		OrganizationID:   organizationID,
		Name:             request.Name,
		Description:      request.Description,
		BrandID:          request.BrandID,
		MaterialID:       request.MaterialID,
		Color:            request.Color,
		ColorHex:         request.ColorHex,
		ColorType:        request.ColorType,
		ColorData:        request.ColorData,
		Diameter:         request.Diameter,
		Weight:           request.Weight,
		PricePerKg:       request.PricePerKg,
		URL:              request.URL,
		PrintTemperature: request.PrintTemperature,
		BedTemperature:   request.BedTemperature,
		// New filaments are available for budgets and slicer suggestions.
		IsActive: true,
	}

	// Only admins may create global (ownerless) filaments; everyone else owns
	// the filaments they create.
	if request.OwnerUserID != nil && isAdmin {
		filament.OwnerUserID = request.OwnerUserID
	} else {
		filament.OwnerUserID = &userIDStr
	}

	// Generate color preview if color data is provided
	if filament.ColorType != "" && len(filament.ColorData) > 0 {
		if colorData, err := filamentEntities.ParseColorData(filament.ColorType, filament.ColorData); err == nil {
			filament.ColorPreview = colorData.GenerateCSS()
			filament.ColorHex = filamentEntities.GenerateLegacyColorHex(filament.ColorType, colorData)
		}
	}

	if err := uc.repository.Create(ctx, filament); err != nil {
		uc.logger.Error(ctx, "Failed to create filament", map[string]interface{}{"name": request.Name, "error": err.Error()})
		coreErrors.Respond(c, err)
		return
	}

	uc.logger.Info(ctx, "Filament created successfully", map[string]interface{}{"filament_id": filament.ID})

	c.JSON(http.StatusCreated, filament)

	uc.activityService.Record(c.Request.Context(), activityEntities.ActivityEntity{
		OrganizationID: organizationID,
		UserID:         helpers.GetUserID(c),
		Action:         activityEntities.ActionCreated,
		EntityType:     activityEntities.EntityFilament,
		EntityID:       filament.ID.String(),
		EntityName:     filament.Name,
		CreatedAt:      time.Now(),
	})
}

// validateBrandAndMaterial ensures the referenced brand and material exist
// within the organization. A missing or foreign id yields a 400 with a stable
// code; any other lookup failure is returned as-is for the 500 path.
func (uc *FilamentUseCase) validateBrandAndMaterial(ctx context.Context, organizationID string, brandID, materialID uuid.UUID) error {
	if _, err := uc.repository.GetBrandInfo(ctx, brandID, organizationID); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return coreErrors.BadRequest("brand_not_found", "Marca não encontrada")
		}
		return err
	}
	if _, err := uc.repository.GetMaterialInfo(ctx, materialID, organizationID); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return coreErrors.BadRequest("material_not_found", "Material não encontrado")
		}
		return err
	}
	return nil
}
