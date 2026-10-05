// Package profile wires the print profile HTTP layer.
package profile

import (
	"errors"
	"net/http"

	"github.com/RodolfoBonis/spooliq/core/helpers"
	"github.com/RodolfoBonis/spooliq/core/roles"
	"github.com/RodolfoBonis/spooliq/features/profile/domain/entities"
	"github.com/RodolfoBonis/spooliq/features/profile/domain/usecases"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Handler handles HTTP requests for print profile operations.
type Handler struct {
	useCase *usecases.ProfileUseCase
}

// NewProfileHandler creates a new print profile handler.
func NewProfileHandler(useCase *usecases.ProfileUseCase) *Handler {
	return &Handler{useCase: useCase}
}

// requireOrganizationID extracts organization_id from context, writing a 400 and
// returning ok=false when it is missing.
func requireOrganizationID(c *gin.Context) (string, bool) {
	organizationID := helpers.GetOrganizationID(c)
	if organizationID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Organization ID not found"})
		return "", false
	}
	return organizationID, true
}

// respondError maps domain/repository errors to HTTP status codes. Not-found
// (including cross-organization access) maps to 404 to avoid leaking existence.
func respondError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, gorm.ErrRecordNotFound), errors.Is(err, entities.ErrProfileNotFound):
		c.JSON(http.StatusNotFound, gin.H{"error": "Profile not found"})
	case errors.Is(err, entities.ErrCannotDeleteDefaultProfile),
		errors.Is(err, entities.ErrDefaultConflict):
		c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
	case errors.Is(err, entities.ErrInvalidMachinePreset),
		errors.Is(err, entities.ErrInvalidEnergyPreset),
		errors.Is(err, entities.ErrInvalidCostPreset):
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
	default:
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
	}
}

// Routes configures all print profile routes with authentication middleware.
func Routes(route *gin.RouterGroup, handler *Handler, protectFactory func(handler gin.HandlerFunc, roles ...string) gin.HandlerFunc) {
	profiles := route.Group("/profiles")
	{
		// Read and create: all roles.
		profiles.GET("", protectFactory(handler.List, roles.OwnerRole, roles.OrgAdminRole, roles.UserRole))
		profiles.GET("/:id", protectFactory(handler.Get, roles.OwnerRole, roles.OrgAdminRole, roles.UserRole))
		profiles.POST("", protectFactory(handler.Create, roles.OwnerRole, roles.OrgAdminRole, roles.UserRole))
		profiles.PUT("/:id", protectFactory(handler.Update, roles.OwnerRole, roles.OrgAdminRole, roles.UserRole))
		profiles.POST("/:id/duplicate", protectFactory(handler.Duplicate, roles.OwnerRole, roles.OrgAdminRole, roles.UserRole))

		// Delete and set-default: Owner and OrgAdmin only.
		profiles.DELETE("/:id", protectFactory(handler.Delete, roles.OwnerRole, roles.OrgAdminRole))
		profiles.POST("/:id/default", protectFactory(handler.SetDefault, roles.OwnerRole, roles.OrgAdminRole))
	}
}

// List retrieves all print profiles for the organization.
// @Summary List print profiles
// @Description Retrieve all print profiles for the caller's organization, each embedding the referenced presets' {id, name}.
// @Tags Profiles
// @Accept json
// @Produce json
// @Success 200 {array} entities.ProfileResponse "Profiles"
// @Failure 500 {object} errors.HTTPError "Internal Server Error"
// @Security BearerAuth
// @Router /profiles [get]
func (h *Handler) List(c *gin.Context) {
	organizationID, ok := requireOrganizationID(c)
	if !ok {
		return
	}
	profiles, err := h.useCase.List(organizationID)
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, profiles)
}

// Get retrieves a print profile by ID.
// @Summary Get print profile by ID
// @Description Retrieve a specific print profile by its ID within the caller's organization.
// @Tags Profiles
// @Accept json
// @Produce json
// @Param id path string true "Profile ID (UUID format)"
// @Success 200 {object} entities.ProfileResponse "Profile"
// @Failure 400 {object} errors.HTTPError "Bad Request - Invalid ID format"
// @Failure 404 {object} errors.HTTPError "Not Found - Profile not found"
// @Security BearerAuth
// @Router /profiles/{id} [get]
func (h *Handler) Get(c *gin.Context) {
	organizationID, ok := requireOrganizationID(c)
	if !ok {
		return
	}
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID format"})
		return
	}
	profile, err := h.useCase.Get(id, organizationID)
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, profile)
}

// Create creates a new print profile.
// @Summary Create print profile
// @Description Create a print profile referencing a machine, an energy and an optional cost preset. The referenced presets must belong to the organization and match their type. Name is auto-generated when empty.
// @Tags Profiles
// @Accept json
// @Produce json
// @Param request body entities.CreateProfileRequest true "Profile data"
// @Success 201 {object} entities.ProfileResponse "Profile created"
// @Failure 400 {object} errors.HTTPError "Bad Request - Invalid request or preset reference"
// @Failure 500 {object} errors.HTTPError "Internal Server Error"
// @Security BearerAuth
// @Router /profiles [post]
func (h *Handler) Create(c *gin.Context) {
	organizationID, ok := requireOrganizationID(c)
	if !ok {
		return
	}
	var req entities.CreateProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	profile, err := h.useCase.Create(&req, organizationID, helpers.GetUserID(c))
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusCreated, profile)
}

// Update updates an existing print profile.
// @Summary Update print profile
// @Description Update a print profile. Only provided fields are applied; changed preset references are re-validated.
// @Tags Profiles
// @Accept json
// @Produce json
// @Param id path string true "Profile ID (UUID format)"
// @Param request body entities.UpdateProfileRequest true "Profile update data"
// @Success 200 {object} entities.ProfileResponse "Profile updated"
// @Failure 400 {object} errors.HTTPError "Bad Request - Invalid request or preset reference"
// @Failure 404 {object} errors.HTTPError "Not Found - Profile not found"
// @Failure 500 {object} errors.HTTPError "Internal Server Error"
// @Security BearerAuth
// @Router /profiles/{id} [put]
func (h *Handler) Update(c *gin.Context) {
	organizationID, ok := requireOrganizationID(c)
	if !ok {
		return
	}
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID format"})
		return
	}
	var req entities.UpdateProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	req.ID = id
	profile, err := h.useCase.Update(&req, organizationID)
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, profile)
}

// Delete soft deletes a print profile.
// @Summary Delete print profile
// @Description Soft delete a print profile. The default profile cannot be deleted.
// @Tags Profiles
// @Accept json
// @Produce json
// @Param id path string true "Profile ID (UUID format)"
// @Success 204 "Profile deleted"
// @Failure 400 {object} errors.HTTPError "Bad Request - Invalid ID format"
// @Failure 404 {object} errors.HTTPError "Not Found - Profile not found"
// @Failure 409 {object} errors.HTTPError "Conflict - Default profiles cannot be deleted"
// @Security BearerAuth
// @Router /profiles/{id} [delete]
func (h *Handler) Delete(c *gin.Context) {
	organizationID, ok := requireOrganizationID(c)
	if !ok {
		return
	}
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID format"})
		return
	}
	if err := h.useCase.Delete(id, organizationID); err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusNoContent, nil)
}

// SetDefault marks a print profile as the organization's default.
// @Summary Set print profile as default
// @Description Mark the profile as the single default for the organization; clears any other default.
// @Tags Profiles
// @Accept json
// @Produce json
// @Param id path string true "Profile ID (UUID format)"
// @Success 200 {object} entities.ProfileResponse "Profile set as default"
// @Failure 400 {object} errors.HTTPError "Bad Request - Invalid ID format"
// @Failure 404 {object} errors.HTTPError "Not Found - Profile not found"
// @Security BearerAuth
// @Router /profiles/{id}/default [post]
func (h *Handler) SetDefault(c *gin.Context) {
	organizationID, ok := requireOrganizationID(c)
	if !ok {
		return
	}
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID format"})
		return
	}
	profile, err := h.useCase.SetDefault(id, organizationID)
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, profile)
}

// Duplicate copies a print profile within the organization.
// @Summary Duplicate print profile
// @Description Copy a print profile within the caller's organization. The copy is named "<name> (cópia)" and is not a default.
// @Tags Profiles
// @Accept json
// @Produce json
// @Param id path string true "Profile ID (UUID format)"
// @Success 201 {object} entities.ProfileResponse "Profile duplicated"
// @Failure 400 {object} errors.HTTPError "Bad Request - Invalid ID format"
// @Failure 404 {object} errors.HTTPError "Not Found - Profile not found"
// @Security BearerAuth
// @Router /profiles/{id}/duplicate [post]
func (h *Handler) Duplicate(c *gin.Context) {
	organizationID, ok := requireOrganizationID(c)
	if !ok {
		return
	}
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID format"})
		return
	}
	profile, err := h.useCase.Duplicate(id, organizationID)
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusCreated, profile)
}
