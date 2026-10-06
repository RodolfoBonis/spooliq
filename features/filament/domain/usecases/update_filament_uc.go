package usecases

import (
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

// Update handles updating an existing filament.
// @Summary Update Filament
// @Description Update an existing 3D printing filament
// @Tags Filaments
// @Accept json
// @Produce json
// @Param id path string true "Filament ID (UUID)"
// @Param request body filamentEntities.UpdateFilamentRequest true "Filament update data"
// @Success 200 {object} filamentEntities.FilamentResponse "Successfully updated filament"
// @Failure 400 {object} errors.APIError
// @Failure 401 {object} errors.APIError
// @Failure 403 {object} errors.APIError
// @Failure 404 {object} errors.APIError
// @Failure 409 {object} errors.APIError
// @Failure 500 {object} errors.APIError
// @Router /filaments/{id} [put]
// @Security BearerAuth
func (uc *FilamentUseCase) Update(c *gin.Context) {
	ctx := c.Request.Context()

	organizationID := helpers.GetOrganizationID(c)
	if organizationID == "" {
		uc.logger.Error(ctx, "Organization ID not found", nil)
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

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		uc.logger.Error(ctx, "Invalid filament ID", map[string]interface{}{"filament_id": c.Param("id")})
		coreErrors.Respond(c, coreErrors.BadRequest("invalid_filament_id", "ID de filamento inválido"))
		return
	}

	var request filamentEntities.UpdateFilamentRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		uc.logger.Error(ctx, "Invalid filament update payload", map[string]interface{}{"error": err.Error()})
		coreErrors.Respond(c, err)
		return
	}

	if err := validation.Validate(request); err != nil {
		uc.logger.Error(ctx, "Filament validation failed", map[string]interface{}{"error": err.Error()})
		coreErrors.Respond(c, err)
		return
	}

	// Validate color data if provided
	if request.ColorType != nil && *request.ColorType != "" && request.ColorData != nil && len(*request.ColorData) > 0 {
		if !request.ColorType.IsValid() {
			uc.logger.Error(ctx, "Invalid color type", map[string]interface{}{"color_type": *request.ColorType})
			coreErrors.Respond(c, coreErrors.BadRequest("invalid_color_type", "Tipo de cor inválido"))
			return
		}
		if _, err := filamentEntities.ParseColorData(*request.ColorType, *request.ColorData); err != nil {
			uc.logger.Error(ctx, "Invalid color data", map[string]interface{}{"error": err.Error(), "color_type": *request.ColorType})
			coreErrors.Respond(c, coreErrors.BadRequest("invalid_color_data", "Dados de cor inválidos"))
			return
		}
	}

	existingFilament, err := uc.repository.FindByID(ctx, id, organizationID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			uc.logger.Error(ctx, "Filament not found", map[string]interface{}{"filament_id": id})
			coreErrors.Respond(c, coreErrors.NotFoundErr("filament_not_found", "Filamento não encontrado"))
			return
		}
		uc.logger.Error(ctx, "Failed to retrieve filament", map[string]interface{}{"filament_id": id, "error": err.Error()})
		coreErrors.Respond(c, err)
		return
	}

	// Only the owner or an admin may update a filament.
	if !isAdmin && (existingFilament.OwnerUserID == nil || *existingFilament.OwnerUserID != userIDStr) {
		uc.logger.Warning(ctx, "Access denied to update filament", map[string]interface{}{"filament_id": id, "user_id": userIDStr})
		coreErrors.Respond(c, coreErrors.Forbidden("filament_access_denied", "Você só pode alterar seus próprios filamentos"))
		return
	}

	// If brand or material is being changed, the new reference must belong to
	// the caller's organization.
	if request.BrandID != nil || request.MaterialID != nil {
		brandID := existingFilament.BrandID
		if request.BrandID != nil {
			brandID = *request.BrandID
		}
		materialID := existingFilament.MaterialID
		if request.MaterialID != nil {
			materialID = *request.MaterialID
		}
		if err := uc.validateBrandAndMaterial(ctx, organizationID, brandID, materialID); err != nil {
			uc.logger.Warning(ctx, "Filament update references brand/material outside organization", map[string]interface{}{"brand_id": brandID, "material_id": materialID})
			coreErrors.Respond(c, err)
			return
		}
	}

	// Enforce unique name within the brand when the name changes.
	if request.Name != nil && *request.Name != existingFilament.Name {
		brandID := existingFilament.BrandID
		if request.BrandID != nil {
			brandID = *request.BrandID
		}
		exists, err := uc.repository.ExistsByNameAndBrand(ctx, *request.Name, brandID, organizationID, &id)
		if err != nil {
			uc.logger.Error(ctx, "Failed to check filament existence", map[string]interface{}{"name": *request.Name, "error": err.Error()})
			coreErrors.Respond(c, err)
			return
		}
		if exists {
			uc.logger.Warning(ctx, "Filament update failed: name already exists", map[string]interface{}{"name": *request.Name})
			coreErrors.Respond(c, coreErrors.Conflict("filament_name_taken", "Já existe um filamento com este nome para esta marca"))
			return
		}
	}

	applyFilamentUpdate(existingFilament, &request)

	// Regenerate color preview if color data changed.
	if request.ColorType != nil && request.ColorData != nil && len(*request.ColorData) > 0 {
		if colorData, err := filamentEntities.ParseColorData(*request.ColorType, *request.ColorData); err == nil {
			existingFilament.ColorPreview = colorData.GenerateCSS()
			existingFilament.ColorHex = filamentEntities.GenerateLegacyColorHex(*request.ColorType, colorData)
		}
	}

	if err := uc.repository.Update(ctx, existingFilament); err != nil {
		uc.logger.Error(ctx, "Failed to update filament", map[string]interface{}{"filament_id": id, "error": err.Error()})
		coreErrors.Respond(c, err)
		return
	}

	updatedFilament, err := uc.repository.FindByID(ctx, id, organizationID)
	if err != nil {
		uc.logger.Error(ctx, "Failed to fetch updated filament", map[string]interface{}{"filament_id": id, "error": err.Error()})
		coreErrors.Respond(c, err)
		return
	}

	uc.logger.Info(ctx, "Filament updated successfully", map[string]interface{}{"filament_id": id})

	response := &filamentEntities.FilamentResponse{FilamentEntity: updatedFilament}
	if brandInfo, err := uc.repository.GetBrandInfo(ctx, updatedFilament.BrandID, organizationID); err == nil {
		response.Brand = brandInfo
	}
	if materialInfo, err := uc.repository.GetMaterialInfo(ctx, updatedFilament.MaterialID, organizationID); err == nil {
		response.Material = materialInfo
	}

	c.JSON(http.StatusOK, response)

	uc.activityService.Record(c.Request.Context(), activityEntities.ActivityEntity{
		OrganizationID: organizationID,
		UserID:         helpers.GetUserID(c),
		Action:         activityEntities.ActionUpdated,
		EntityType:     activityEntities.EntityFilament,
		EntityID:       updatedFilament.ID.String(),
		EntityName:     updatedFilament.Name,
		CreatedAt:      time.Now(),
	})
}

// applyFilamentUpdate copies the non-nil fields of the request onto the entity.
func applyFilamentUpdate(f *filamentEntities.FilamentEntity, request *filamentEntities.UpdateFilamentRequest) {
	if request.IsActive != nil {
		f.IsActive = *request.IsActive
	}
	if request.Name != nil {
		f.Name = *request.Name
	}
	if request.Description != nil {
		f.Description = *request.Description
	}
	if request.BrandID != nil {
		f.BrandID = *request.BrandID
	}
	if request.MaterialID != nil {
		f.MaterialID = *request.MaterialID
	}
	if request.Color != nil {
		f.Color = *request.Color
	}
	if request.ColorHex != nil {
		f.ColorHex = *request.ColorHex
	}
	if request.ColorType != nil {
		f.ColorType = *request.ColorType
	}
	if request.ColorData != nil {
		f.ColorData = *request.ColorData
	}
	if request.Diameter != nil {
		f.Diameter = *request.Diameter
	}
	if request.Weight != nil {
		f.Weight = request.Weight
	}
	if request.PricePerKg != nil {
		f.PricePerKg = *request.PricePerKg
	}
	if request.URL != nil {
		f.URL = *request.URL
	}
	if request.PrintTemperature != nil {
		f.PrintTemperature = request.PrintTemperature
	}
	if request.BedTemperature != nil {
		f.BedTemperature = request.BedTemperature
	}
}
