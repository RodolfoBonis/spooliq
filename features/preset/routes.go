package preset

import (
	stderrors "errors"
	"net/http"
	"strings"
	"time"

	coreerrors "github.com/RodolfoBonis/spooliq/core/errors"
	"github.com/RodolfoBonis/spooliq/core/helpers"
	"github.com/RodolfoBonis/spooliq/core/roles"
	activityEntities "github.com/RodolfoBonis/spooliq/features/activity/domain/entities"
	activityUc "github.com/RodolfoBonis/spooliq/features/activity/domain/usecases"
	"github.com/RodolfoBonis/spooliq/features/preset/domain/entities"
	"github.com/RodolfoBonis/spooliq/features/preset/domain/usecases"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Handler handles HTTP requests for preset operations
type Handler struct {
	createUC        *usecases.CreatePresetUseCase
	findUC          *usecases.FindPresetUseCase
	updateUC        *usecases.UpdatePresetUseCase
	deleteUC        *usecases.DeletePresetUseCase
	manageUC        *usecases.ManagePresetUseCase
	activityService activityUc.IActivityService
}

// NewPresetHandler creates a new preset handler
func NewPresetHandler(
	createUC *usecases.CreatePresetUseCase,
	findUC *usecases.FindPresetUseCase,
	updateUC *usecases.UpdatePresetUseCase,
	deleteUC *usecases.DeletePresetUseCase,
	manageUC *usecases.ManagePresetUseCase,
	activityService activityUc.IActivityService,
) *Handler {
	return &Handler{
		createUC:        createUC,
		findUC:          findUC,
		updateUC:        updateUC,
		deleteUC:        deleteUC,
		manageUC:        manageUC,
		activityService: activityService,
	}
}

// presetListOptions is the single source of truth for how preset list endpoints
// interpret pagination, search and sort. Default page size is 100; free-text
// search (q) matches the preset name; sorting is restricted to name/created_at.
func presetListOptions() helpers.ListQueryOptions {
	return helpers.ListQueryOptions{
		DefaultPageSize: 100,
		SortWhitelist: map[string]string{
			"name":       "presets.name",
			"created_at": "presets.created_at",
		},
		DefaultSort: "created_at",
	}
}

// requireOrganizationID extracts the organization_id from context, writing a
// 400 response and returning ok=false when it is missing.
func requireOrganizationID(c *gin.Context) (string, bool) {
	organizationID := helpers.GetOrganizationID(c)
	if organizationID == "" {
		coreerrors.Respond(c, coreerrors.BadRequest("organization_id_missing", "Organização não encontrada no contexto"))
		return "", false
	}
	return organizationID, true
}

// authenticatedUserID returns the authenticated user's ID from context as a
// *uuid.UUID, or nil when it is absent or not a valid UUID. The request body is
// never trusted for ownership.
func authenticatedUserID(c *gin.Context) *uuid.UUID {
	userIDStr := helpers.GetUserID(c)
	if userIDStr == "" {
		return nil
	}
	parsed, err := uuid.Parse(userIDStr)
	if err != nil {
		return nil
	}
	return &parsed
}

// respondPresetError maps domain/repository errors to the standard error
// envelope. Not-found (including cross-organization access and wrong-type reads)
// maps to 404 to avoid leaking the existence of other tenants' presets. Anything
// unmapped goes through coreerrors.Respond, which logs and returns a clean 500.
func respondPresetError(c *gin.Context, err error) {
	switch {
	case stderrors.Is(err, gorm.ErrRecordNotFound), stderrors.Is(err, entities.ErrPresetNotFound):
		coreerrors.Respond(c, coreerrors.NotFoundErr("preset_not_found", "Preset não encontrado"))
	case stderrors.Is(err, entities.ErrInvalidPresetType):
		// Requesting a resource of the wrong kind is treated as not found.
		coreerrors.Respond(c, coreerrors.NotFoundErr("preset_not_found", "Preset não encontrado"))
	case stderrors.Is(err, entities.ErrCannotDeleteDefaultPreset):
		coreerrors.Respond(c, coreerrors.Conflict("default_preset_cannot_be_deleted", "Presets padrão não podem ser excluídos"))
	case stderrors.Is(err, entities.ErrPresetInUseByProfile):
		coreerrors.Respond(c, coreerrors.Conflict("preset_in_use_by_profile", err.Error()))
	case stderrors.Is(err, entities.ErrDefaultConflict):
		coreerrors.Respond(c, coreerrors.Conflict("default_conflict", err.Error()))
	case stderrors.Is(err, entities.ErrPresetNameRequired):
		coreerrors.Respond(c, coreerrors.BadRequest("preset_name_required", "Nome do preset é obrigatório"))
	default:
		coreerrors.Respond(c, err)
	}
}

// SetupRoutes configures the preset routes with authentication middleware
func SetupRoutes(router *gin.RouterGroup, handler *Handler, protectFactory func(handler gin.HandlerFunc, roles ...string) gin.HandlerFunc) {
	presets := router.Group("/presets")
	{
		// Base preset routes - all users can view, create, and update; only Owner and OrgAdmin can delete
		presets.GET("", protectFactory(handler.GetPresets, roles.OwnerRole, roles.OrgAdminRole, roles.UserRole))
		presets.GET("/:id", protectFactory(handler.GetPresetByID, roles.OwnerRole, roles.OrgAdminRole, roles.UserRole))
		presets.DELETE("/:id", protectFactory(handler.DeletePreset, roles.OwnerRole, roles.OrgAdminRole))

		// Name suggestion (pure helper so the web app can prefill the name field).
		presets.POST("/suggest-name", protectFactory(handler.SuggestName, roles.OwnerRole, roles.OrgAdminRole, roles.UserRole))

		// Templates catalog (static) and instantiation.
		presets.GET("/templates", protectFactory(handler.GetTemplates, roles.OwnerRole, roles.OrgAdminRole, roles.UserRole))
		presets.POST("/from-template/:key", protectFactory(handler.CreateFromTemplate, roles.OwnerRole, roles.OrgAdminRole, roles.UserRole))

		// Default and duplicate actions.
		presets.POST("/:id/default", protectFactory(handler.SetDefault, roles.OwnerRole, roles.OrgAdminRole))
		presets.POST("/:id/duplicate", protectFactory(handler.Duplicate, roles.OwnerRole, roles.OrgAdminRole, roles.UserRole))

		// Machine preset routes
		machines := presets.Group("/machines")
		{
			machines.POST("", protectFactory(handler.CreateMachinePreset, roles.OwnerRole, roles.OrgAdminRole, roles.UserRole))
			machines.GET("", protectFactory(handler.GetMachinePresets, roles.OwnerRole, roles.OrgAdminRole, roles.UserRole))
			machines.GET("/:id", protectFactory(handler.GetMachinePresetByID, roles.OwnerRole, roles.OrgAdminRole, roles.UserRole))
			machines.PUT("/:id", protectFactory(handler.UpdateMachinePreset, roles.OwnerRole, roles.OrgAdminRole, roles.UserRole))
			machines.GET("/brand/:brand", protectFactory(handler.GetMachinePresetsByBrand, roles.OwnerRole, roles.OrgAdminRole, roles.UserRole))
		}

		// Energy preset routes
		energy := presets.Group("/energy")
		{
			energy.POST("", protectFactory(handler.CreateEnergyPreset, roles.OwnerRole, roles.OrgAdminRole, roles.UserRole))
			energy.GET("", protectFactory(handler.GetEnergyPresets, roles.OwnerRole, roles.OrgAdminRole, roles.UserRole))
			energy.GET("/:id", protectFactory(handler.GetEnergyPresetByID, roles.OwnerRole, roles.OrgAdminRole, roles.UserRole))
			energy.PUT("/:id", protectFactory(handler.UpdateEnergyPreset, roles.OwnerRole, roles.OrgAdminRole, roles.UserRole))
			energy.GET("/location", protectFactory(handler.GetEnergyPresetsByLocation, roles.OwnerRole, roles.OrgAdminRole, roles.UserRole))
			energy.GET("/currency/:currency", protectFactory(handler.GetEnergyPresetsByCurrency, roles.OwnerRole, roles.OrgAdminRole, roles.UserRole))
		}

		// Cost preset routes
		costs := presets.Group("/costs")
		{
			costs.POST("", protectFactory(handler.CreateCostPreset, roles.OwnerRole, roles.OrgAdminRole, roles.UserRole))
			costs.GET("", protectFactory(handler.GetCostPresets, roles.OwnerRole, roles.OrgAdminRole, roles.UserRole))
			costs.GET("/:id", protectFactory(handler.GetCostPresetByID, roles.OwnerRole, roles.OrgAdminRole, roles.UserRole))
			costs.PUT("/:id", protectFactory(handler.UpdateCostPreset, roles.OwnerRole, roles.OrgAdminRole, roles.UserRole))
		}
	}
}

// GetPresets retrieves presets with optional filters, paginated with the standard envelope.
// @Summary Get presets with filters
// @Description Retrieve presets with optional, combinable filters including type, active status, default status, global status, and user ID. All filters are applied together within the caller's organization scope. Results are paginated with the standard envelope.
// @Tags Presets
// @Accept json
// @Produce json
// @Param type query string false "Preset type filter (machine, energy, cost)"
// @Param active query boolean false "Filter only active presets"
// @Param default query boolean false "Filter only default presets"
// @Param global query boolean false "Filter only global presets"
// @Param user_id query string false "Filter presets by user ID (UUID format)"
// @Param page query int false "Page number (default 1)"
// @Param page_size query int false "Items per page (default 100, max 100)"
// @Param q query string false "Free-text search on name"
// @Param sort_by query string false "Sort field: name or created_at"
// @Param sort_dir query string false "Sort direction: asc or desc"
// @Success 200 {object} helpers.Page[entities.PresetEntity] "Paginated presets"
// @Failure 400 {object} errors.HTTPError "Bad Request - Invalid user ID format"
// @Failure 500 {object} errors.HTTPError "Internal Server Error"
// @Security BearerAuth
// @Router /presets [get]
func (h *Handler) GetPresets(c *gin.Context) {
	organizationID, ok := requireOrganizationID(c)
	if !ok {
		return
	}

	presetType := c.Query("type")
	activeOnly := c.Query("active") == "true"
	defaultOnly := c.Query("default") == "true"
	globalOnly := c.Query("global") == "true"
	userIDStr := c.Query("user_id")

	filters := entities.PresetFilters{
		ActiveOnly:  activeOnly,
		DefaultOnly: defaultOnly,
		GlobalOnly:  globalOnly,
	}

	if presetType != "" {
		pt := entities.PresetType(presetType)
		filters.Type = &pt
	}

	if userIDStr != "" {
		userID, parseErr := uuid.Parse(userIDStr)
		if parseErr != nil {
			coreerrors.Respond(c, coreerrors.BadRequest("invalid_user_id", "ID de usuário inválido"))
			return
		}
		filters.UserID = &userID
	}

	// Preserve the previous default of returning only active presets when the
	// caller supplies no filters at all.
	if !activeOnly && !defaultOnly && !globalOnly && presetType == "" && userIDStr == "" {
		filters.ActiveOnly = true
	}

	q := helpers.ParseListQuery(c, presetListOptions())
	presets, total, err := h.findUC.FindPresets(organizationID, filters, q)
	if err != nil {
		coreerrors.Respond(c, err)
		return
	}

	c.JSON(http.StatusOK, helpers.NewPage(presets, total, q))
}

// GetPresetByID retrieves a preset by ID
// @Summary Get preset by ID
// @Description Retrieve a specific preset by its unique identifier within the caller's organization
// @Tags Presets
// @Accept json
// @Produce json
// @Param id path string true "Preset ID (UUID format)"
// @Success 200 {object} entities.PresetEntity "Successfully retrieved preset"
// @Failure 400 {object} errors.HTTPError "Bad Request - Invalid ID format"
// @Failure 404 {object} errors.HTTPError "Not Found - Preset not found"
// @Security BearerAuth
// @Router /presets/{id} [get]
func (h *Handler) GetPresetByID(c *gin.Context) {
	organizationID, ok := requireOrganizationID(c)
	if !ok {
		return
	}

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		coreerrors.Respond(c, coreerrors.BadRequest("invalid_preset_id", "ID de preset inválido"))
		return
	}

	preset, err := h.findUC.FindByID(id, organizationID)
	if err != nil {
		respondPresetError(c, err)
		return
	}

	c.JSON(http.StatusOK, preset)
}

// DeletePreset deletes a preset by ID
// @Summary Delete preset
// @Description Delete a preset by its unique identifier within the caller's organization. Default presets cannot be deleted.
// @Tags Presets
// @Accept json
// @Produce json
// @Param id path string true "Preset ID (UUID format)"
// @Success 204 "Preset deleted successfully"
// @Failure 400 {object} errors.HTTPError "Bad Request - Invalid ID format"
// @Failure 404 {object} errors.HTTPError "Not Found - Preset not found"
// @Failure 409 {object} errors.HTTPError "Conflict - Default presets cannot be deleted"
// @Failure 500 {object} errors.HTTPError "Internal Server Error"
// @Security BearerAuth
// @Router /presets/{id} [delete]
func (h *Handler) DeletePreset(c *gin.Context) {
	organizationID, ok := requireOrganizationID(c)
	if !ok {
		return
	}

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		coreerrors.Respond(c, coreerrors.BadRequest("invalid_preset_id", "ID de preset inválido"))
		return
	}

	// Fetch preset before deleting to get name for activity (organization-scoped)
	preset, _ := h.findUC.FindByID(id, organizationID)

	if err := h.deleteUC.Execute(id, organizationID); err != nil {
		respondPresetError(c, err)
		return
	}

	c.JSON(http.StatusNoContent, nil)

	// Record activity (fire-and-forget)
	presetName := ""
	if preset != nil {
		presetName = preset.Name
	}
	h.activityService.Record(c.Request.Context(), activityEntities.ActivityEntity{
		OrganizationID: organizationID,
		UserID:         helpers.GetUserID(c),
		Action:         activityEntities.ActionDeleted,
		EntityType:     activityEntities.EntityPreset,
		EntityID:       id.String(),
		EntityName:     presetName,
		CreatedAt:      time.Now(),
	})
}

// CreateMachinePreset creates a new machine preset
// @Summary Create machine preset
// @Description Create a new machine preset with specifications like build volume, nozzle diameter, and power consumption
// @Tags Machine Presets
// @Accept json
// @Produce json
// @Param request body usecases.CreateMachinePresetRequest true "Machine preset creation data"
// @Success 201 {object} entities.PresetEntity "Machine preset created successfully"
// @Failure 400 {object} errors.HTTPError "Bad Request - Invalid request data"
// @Failure 500 {object} errors.HTTPError "Internal Server Error"
// @Security BearerAuth
// @Router /presets/machines [post]
func (h *Handler) CreateMachinePreset(c *gin.Context) {
	var req usecases.CreateMachinePresetRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		coreerrors.Respond(c, err)
		return
	}

	organizationID, ok := requireOrganizationID(c)
	if !ok {
		return
	}

	preset, err := h.createUC.CreateMachinePreset(&req, organizationID, authenticatedUserID(c))
	if err != nil {
		respondPresetError(c, err)
		return
	}

	c.JSON(http.StatusCreated, preset)

	// Record activity (fire-and-forget)
	h.activityService.Record(c.Request.Context(), activityEntities.ActivityEntity{
		OrganizationID: organizationID,
		UserID:         helpers.GetUserID(c),
		Action:         activityEntities.ActionCreated,
		EntityType:     activityEntities.EntityPreset,
		EntityID:       preset.ID.String(),
		EntityName:     preset.Name,
		CreatedAt:      time.Now(),
	})
}

// GetMachinePresets retrieves machine presets, paginated with the standard envelope.
// @Summary Get machine presets
// @Description Retrieve machine presets with their specifications, paginated with the standard envelope.
// @Tags Machine Presets
// @Accept json
// @Produce json
// @Param page query int false "Page number (default 1)"
// @Param page_size query int false "Items per page (default 100, max 100)"
// @Param q query string false "Free-text search on name"
// @Param sort_by query string false "Sort field: name or created_at"
// @Param sort_dir query string false "Sort direction: asc or desc"
// @Success 200 {object} helpers.Page[usecases.MachinePresetResponse] "Paginated machine presets"
// @Failure 500 {object} errors.HTTPError "Internal Server Error"
// @Security BearerAuth
// @Router /presets/machines [get]
func (h *Handler) GetMachinePresets(c *gin.Context) {
	organizationID, ok := requireOrganizationID(c)
	if !ok {
		return
	}

	q := helpers.ParseListQuery(c, presetListOptions())
	presets, total, err := h.findUC.FindAllMachinePresets(organizationID, q)
	if err != nil {
		coreerrors.Respond(c, err)
		return
	}

	c.JSON(http.StatusOK, helpers.NewPage(presets, total, q))
}

// GetMachinePresetByID retrieves a machine preset with full details
// @Summary Get machine preset by ID
// @Description Retrieve a specific machine preset with complete specifications by its ID
// @Tags Machine Presets
// @Accept json
// @Produce json
// @Param id path string true "Machine preset ID (UUID format)"
// @Success 200 {object} entities.MachinePresetEntity "Successfully retrieved machine preset"
// @Failure 400 {object} errors.HTTPError "Bad Request - Invalid ID format"
// @Failure 404 {object} errors.HTTPError "Not Found - Machine preset not found"
// @Security BearerAuth
// @Router /presets/machines/{id} [get]
func (h *Handler) GetMachinePresetByID(c *gin.Context) {
	organizationID, ok := requireOrganizationID(c)
	if !ok {
		return
	}

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		coreerrors.Respond(c, coreerrors.BadRequest("invalid_preset_id", "ID de preset inválido"))
		return
	}

	preset, err := h.findUC.FindMachinePresetByID(id, organizationID)
	if err != nil {
		respondPresetError(c, err)
		return
	}

	c.JSON(http.StatusOK, preset)
}

// UpdateMachinePreset updates a machine preset
// @Summary Update machine preset
// @Description Update an existing machine preset with new specifications
// @Tags Machine Presets
// @Accept json
// @Produce json
// @Param id path string true "Machine preset ID (UUID format)"
// @Param request body usecases.UpdateMachinePresetRequest true "Machine preset update data"
// @Success 200 {object} entities.PresetEntity "Machine preset updated successfully"
// @Failure 400 {object} errors.HTTPError "Bad Request - Invalid ID format or request data"
// @Failure 404 {object} errors.HTTPError "Not Found - Machine preset not found"
// @Failure 500 {object} errors.HTTPError "Internal Server Error"
// @Security BearerAuth
// @Router /presets/machines/{id} [put]
func (h *Handler) UpdateMachinePreset(c *gin.Context) {
	organizationID, ok := requireOrganizationID(c)
	if !ok {
		return
	}

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		coreerrors.Respond(c, coreerrors.BadRequest("invalid_preset_id", "ID de preset inválido"))
		return
	}

	var req usecases.UpdateMachinePresetRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		coreerrors.Respond(c, err)
		return
	}
	req.ID = id

	preset, err := h.updateUC.UpdateMachinePreset(&req, organizationID)
	if err != nil {
		respondPresetError(c, err)
		return
	}

	c.JSON(http.StatusOK, preset)

	// Record activity (fire-and-forget)
	h.activityService.Record(c.Request.Context(), activityEntities.ActivityEntity{
		OrganizationID: organizationID,
		UserID:         helpers.GetUserID(c),
		Action:         activityEntities.ActionUpdated,
		EntityType:     activityEntities.EntityPreset,
		EntityID:       preset.ID.String(),
		EntityName:     preset.Name,
		CreatedAt:      time.Now(),
	})
}

// GetMachinePresetsByBrand retrieves machine presets by brand, paginated with the standard envelope.
// @Summary Get machine presets by brand
// @Description Retrieve machine presets from a specific brand, paginated with the standard envelope.
// @Tags Machine Presets
// @Accept json
// @Produce json
// @Param brand path string true "Machine brand name"
// @Param page query int false "Page number (default 1)"
// @Param page_size query int false "Items per page (default 100, max 100)"
// @Param q query string false "Free-text search on name"
// @Param sort_by query string false "Sort field: name or created_at"
// @Param sort_dir query string false "Sort direction: asc or desc"
// @Success 200 {object} helpers.Page[usecases.MachinePresetResponse] "Paginated machine presets by brand"
// @Failure 500 {object} errors.HTTPError "Internal Server Error"
// @Security BearerAuth
// @Router /presets/machines/brand/{brand} [get]
func (h *Handler) GetMachinePresetsByBrand(c *gin.Context) {
	brand := c.Param("brand")

	organizationID, ok := requireOrganizationID(c)
	if !ok {
		return
	}

	q := helpers.ParseListQuery(c, presetListOptions())
	presets, total, err := h.findUC.FindMachinePresetsByBrand(brand, organizationID, q)
	if err != nil {
		coreerrors.Respond(c, err)
		return
	}

	c.JSON(http.StatusOK, helpers.NewPage(presets, total, q))
}

// CreateEnergyPreset creates a new energy preset
// @Summary Create energy preset
// @Description Create a new energy preset with cost per kWh, currency, and location information
// @Tags Energy Presets
// @Accept json
// @Produce json
// @Param request body usecases.CreateEnergyPresetRequest true "Energy preset creation data"
// @Success 201 {object} entities.PresetEntity "Energy preset created successfully"
// @Failure 400 {object} errors.HTTPError "Bad Request - Invalid request data"
// @Failure 500 {object} errors.HTTPError "Internal Server Error"
// @Security BearerAuth
// @Router /presets/energy [post]
func (h *Handler) CreateEnergyPreset(c *gin.Context) {
	var req usecases.CreateEnergyPresetRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		coreerrors.Respond(c, err)
		return
	}

	organizationID, ok := requireOrganizationID(c)
	if !ok {
		return
	}

	preset, err := h.createUC.CreateEnergyPreset(&req, organizationID, authenticatedUserID(c))
	if err != nil {
		respondPresetError(c, err)
		return
	}

	c.JSON(http.StatusCreated, preset)

	// Record activity (fire-and-forget)
	h.activityService.Record(c.Request.Context(), activityEntities.ActivityEntity{
		OrganizationID: organizationID,
		UserID:         helpers.GetUserID(c),
		Action:         activityEntities.ActionCreated,
		EntityType:     activityEntities.EntityPreset,
		EntityID:       preset.ID.String(),
		EntityName:     preset.Name,
		CreatedAt:      time.Now(),
	})
}

// GetEnergyPresets retrieves energy presets, paginated with the standard envelope.
// @Summary Get energy presets
// @Description Retrieve energy presets with pricing and location data, paginated with the standard envelope.
// @Tags Energy Presets
// @Accept json
// @Produce json
// @Param page query int false "Page number (default 1)"
// @Param page_size query int false "Items per page (default 100, max 100)"
// @Param q query string false "Free-text search on name"
// @Param sort_by query string false "Sort field: name or created_at"
// @Param sort_dir query string false "Sort direction: asc or desc"
// @Success 200 {object} helpers.Page[usecases.EnergyPresetResponse] "Paginated energy presets"
// @Failure 500 {object} errors.HTTPError "Internal Server Error"
// @Security BearerAuth
// @Router /presets/energy [get]
func (h *Handler) GetEnergyPresets(c *gin.Context) {
	organizationID, ok := requireOrganizationID(c)
	if !ok {
		return
	}

	q := helpers.ParseListQuery(c, presetListOptions())
	presets, total, err := h.findUC.FindAllEnergyPresets(organizationID, q)
	if err != nil {
		coreerrors.Respond(c, err)
		return
	}

	c.JSON(http.StatusOK, helpers.NewPage(presets, total, q))
}

// GetEnergyPresetByID retrieves an energy preset with full details
// @Summary Get energy preset by ID
// @Description Retrieve a specific energy preset with complete pricing and location details by its ID
// @Tags Energy Presets
// @Accept json
// @Produce json
// @Param id path string true "Energy preset ID (UUID format)"
// @Success 200 {object} entities.EnergyPresetEntity "Successfully retrieved energy preset"
// @Failure 400 {object} errors.HTTPError "Bad Request - Invalid ID format"
// @Failure 404 {object} errors.HTTPError "Not Found - Energy preset not found"
// @Security BearerAuth
// @Router /presets/energy/{id} [get]
func (h *Handler) GetEnergyPresetByID(c *gin.Context) {
	organizationID, ok := requireOrganizationID(c)
	if !ok {
		return
	}

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		coreerrors.Respond(c, coreerrors.BadRequest("invalid_preset_id", "ID de preset inválido"))
		return
	}

	preset, err := h.findUC.FindEnergyPresetByID(id, organizationID)
	if err != nil {
		respondPresetError(c, err)
		return
	}

	c.JSON(http.StatusOK, preset)
}

// UpdateEnergyPreset updates an energy preset
// @Summary Update energy preset
// @Description Update an existing energy preset with new pricing and location information
// @Tags Energy Presets
// @Accept json
// @Produce json
// @Param id path string true "Energy preset ID (UUID format)"
// @Param request body usecases.UpdateEnergyPresetRequest true "Energy preset update data"
// @Success 200 {object} entities.PresetEntity "Energy preset updated successfully"
// @Failure 400 {object} errors.HTTPError "Bad Request - Invalid ID format or request data"
// @Failure 404 {object} errors.HTTPError "Not Found - Energy preset not found"
// @Failure 500 {object} errors.HTTPError "Internal Server Error"
// @Security BearerAuth
// @Router /presets/energy/{id} [put]
func (h *Handler) UpdateEnergyPreset(c *gin.Context) {
	organizationID, ok := requireOrganizationID(c)
	if !ok {
		return
	}

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		coreerrors.Respond(c, coreerrors.BadRequest("invalid_preset_id", "ID de preset inválido"))
		return
	}

	var req usecases.UpdateEnergyPresetRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		coreerrors.Respond(c, err)
		return
	}
	req.ID = id

	preset, err := h.updateUC.UpdateEnergyPreset(&req, organizationID)
	if err != nil {
		respondPresetError(c, err)
		return
	}

	c.JSON(http.StatusOK, preset)

	// Record activity (fire-and-forget)
	h.activityService.Record(c.Request.Context(), activityEntities.ActivityEntity{
		OrganizationID: organizationID,
		UserID:         helpers.GetUserID(c),
		Action:         activityEntities.ActionUpdated,
		EntityType:     activityEntities.EntityPreset,
		EntityID:       preset.ID.String(),
		EntityName:     preset.Name,
		CreatedAt:      time.Now(),
	})
}

// GetEnergyPresetsByLocation retrieves energy presets by location, paginated with the standard envelope.
// @Summary Get energy presets by location
// @Description Retrieve energy presets filtered by country, state, and/or city, paginated with the standard envelope.
// @Tags Energy Presets
// @Accept json
// @Produce json
// @Param country query string false "Filter by country"
// @Param state query string false "Filter by state/province"
// @Param city query string false "Filter by city"
// @Param page query int false "Page number (default 1)"
// @Param page_size query int false "Items per page (default 100, max 100)"
// @Param q query string false "Free-text search on name"
// @Param sort_by query string false "Sort field: name or created_at"
// @Param sort_dir query string false "Sort direction: asc or desc"
// @Success 200 {object} helpers.Page[usecases.EnergyPresetResponse] "Paginated energy presets by location"
// @Failure 500 {object} errors.HTTPError "Internal Server Error"
// @Security BearerAuth
// @Router /presets/energy/location [get]
func (h *Handler) GetEnergyPresetsByLocation(c *gin.Context) {
	country := c.Query("country")
	state := c.Query("state")
	city := c.Query("city")

	organizationID, ok := requireOrganizationID(c)
	if !ok {
		return
	}

	q := helpers.ParseListQuery(c, presetListOptions())
	presets, total, err := h.findUC.FindEnergyPresetsByLocation(country, state, city, organizationID, q)
	if err != nil {
		coreerrors.Respond(c, err)
		return
	}

	c.JSON(http.StatusOK, helpers.NewPage(presets, total, q))
}

// GetEnergyPresetsByCurrency retrieves energy presets by currency, paginated with the standard envelope.
// @Summary Get energy presets by currency
// @Description Retrieve energy presets that use a specific currency (3-letter currency code), paginated with the standard envelope.
// @Tags Energy Presets
// @Accept json
// @Produce json
// @Param currency path string true "Currency code (3 letters, e.g., USD, EUR, BRL)"
// @Param page query int false "Page number (default 1)"
// @Param page_size query int false "Items per page (default 100, max 100)"
// @Param q query string false "Free-text search on name"
// @Param sort_by query string false "Sort field: name or created_at"
// @Param sort_dir query string false "Sort direction: asc or desc"
// @Success 200 {object} helpers.Page[usecases.EnergyPresetResponse] "Paginated energy presets by currency"
// @Failure 500 {object} errors.HTTPError "Internal Server Error"
// @Security BearerAuth
// @Router /presets/energy/currency/{currency} [get]
func (h *Handler) GetEnergyPresetsByCurrency(c *gin.Context) {
	currency := c.Param("currency")

	organizationID, ok := requireOrganizationID(c)
	if !ok {
		return
	}

	q := helpers.ParseListQuery(c, presetListOptions())
	presets, total, err := h.findUC.FindEnergyPresetsByCurrency(currency, organizationID, q)
	if err != nil {
		coreerrors.Respond(c, err)
		return
	}

	c.JSON(http.StatusOK, helpers.NewPage(presets, total, q))
}

// CreateCostPreset creates a new cost preset
// @Summary Create cost preset
// @Description Create a new cost preset with labor costs, packaging, shipping, overhead, and profit margins
// @Tags Cost Presets
// @Accept json
// @Produce json
// @Param request body usecases.CreateCostPresetRequest true "Cost preset creation data"
// @Success 201 {object} entities.PresetEntity "Cost preset created successfully"
// @Failure 400 {object} errors.HTTPError "Bad Request - Invalid request data"
// @Failure 500 {object} errors.HTTPError "Internal Server Error"
// @Security BearerAuth
// @Router /presets/costs [post]
func (h *Handler) CreateCostPreset(c *gin.Context) {
	var req usecases.CreateCostPresetRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		coreerrors.Respond(c, err)
		return
	}

	organizationID, ok := requireOrganizationID(c)
	if !ok {
		return
	}

	preset, err := h.createUC.CreateCostPreset(&req, organizationID, authenticatedUserID(c))
	if err != nil {
		respondPresetError(c, err)
		return
	}

	c.JSON(http.StatusCreated, preset)

	// Record activity (fire-and-forget)
	h.activityService.Record(c.Request.Context(), activityEntities.ActivityEntity{
		OrganizationID: organizationID,
		UserID:         helpers.GetUserID(c),
		Action:         activityEntities.ActionCreated,
		EntityType:     activityEntities.EntityPreset,
		EntityID:       preset.ID.String(),
		EntityName:     preset.Name,
		CreatedAt:      time.Now(),
	})
}

// GetCostPresets retrieves cost presets, paginated with the standard envelope.
// @Summary Get cost presets
// @Description Retrieve cost presets with pricing and margin configurations, paginated with the standard envelope.
// @Tags Cost Presets
// @Accept json
// @Produce json
// @Param page query int false "Page number (default 1)"
// @Param page_size query int false "Items per page (default 100, max 100)"
// @Param q query string false "Free-text search on name"
// @Param sort_by query string false "Sort field: name or created_at"
// @Param sort_dir query string false "Sort direction: asc or desc"
// @Success 200 {object} helpers.Page[usecases.CostPresetResponse] "Paginated cost presets"
// @Failure 500 {object} errors.HTTPError "Internal Server Error"
// @Security BearerAuth
// @Router /presets/costs [get]
func (h *Handler) GetCostPresets(c *gin.Context) {
	organizationID, ok := requireOrganizationID(c)
	if !ok {
		return
	}

	q := helpers.ParseListQuery(c, presetListOptions())
	presets, total, err := h.findUC.FindAllCostPresets(organizationID, q)
	if err != nil {
		coreerrors.Respond(c, err)
		return
	}

	c.JSON(http.StatusOK, helpers.NewPage(presets, total, q))
}

// GetCostPresetByID retrieves a cost preset with full details
// @Summary Get cost preset by ID
// @Description Retrieve a specific cost preset with complete pricing and margin details by its ID
// @Tags Cost Presets
// @Accept json
// @Produce json
// @Param id path string true "Cost preset ID (UUID format)"
// @Success 200 {object} entities.CostPresetEntity "Successfully retrieved cost preset"
// @Failure 400 {object} errors.HTTPError "Bad Request - Invalid ID format"
// @Failure 404 {object} errors.HTTPError "Not Found - Cost preset not found"
// @Security BearerAuth
// @Router /presets/costs/{id} [get]
func (h *Handler) GetCostPresetByID(c *gin.Context) {
	organizationID, ok := requireOrganizationID(c)
	if !ok {
		return
	}

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		coreerrors.Respond(c, coreerrors.BadRequest("invalid_preset_id", "ID de preset inválido"))
		return
	}

	preset, err := h.findUC.FindCostPresetByID(id, organizationID)
	if err != nil {
		respondPresetError(c, err)
		return
	}

	c.JSON(http.StatusOK, preset)
}

// UpdateCostPreset updates a cost preset
// @Summary Update cost preset
// @Description Update an existing cost preset with new pricing and margin configurations
// @Tags Cost Presets
// @Accept json
// @Produce json
// @Param id path string true "Cost preset ID (UUID format)"
// @Param request body usecases.UpdateCostPresetRequest true "Cost preset update data"
// @Success 200 {object} entities.PresetEntity "Cost preset updated successfully"
// @Failure 400 {object} errors.HTTPError "Bad Request - Invalid ID format or request data"
// @Failure 404 {object} errors.HTTPError "Not Found - Cost preset not found"
// @Failure 500 {object} errors.HTTPError "Internal Server Error"
// @Security BearerAuth
// @Router /presets/costs/{id} [put]
func (h *Handler) UpdateCostPreset(c *gin.Context) {
	organizationID, ok := requireOrganizationID(c)
	if !ok {
		return
	}

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		coreerrors.Respond(c, coreerrors.BadRequest("invalid_preset_id", "ID de preset inválido"))
		return
	}

	var req usecases.UpdateCostPresetRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		coreerrors.Respond(c, err)
		return
	}
	req.ID = id

	preset, err := h.updateUC.UpdateCostPreset(&req, organizationID)
	if err != nil {
		respondPresetError(c, err)
		return
	}

	c.JSON(http.StatusOK, preset)

	// Record activity (fire-and-forget)
	h.activityService.Record(c.Request.Context(), activityEntities.ActivityEntity{
		OrganizationID: organizationID,
		UserID:         helpers.GetUserID(c),
		Action:         activityEntities.ActionUpdated,
		EntityType:     activityEntities.EntityPreset,
		EntityID:       preset.ID.String(),
		EntityName:     preset.Name,
		CreatedAt:      time.Now(),
	})
}

// SuggestName returns an auto-generated preset name for the given type and fields.
// @Summary Suggest preset name
// @Description Generate a preset name from its type and fields so the web app can prefill the name input. Pure helper; does not persist anything.
// @Tags Presets
// @Accept json
// @Produce json
// @Param request body usecases.SuggestNameRequest true "Type and fields"
// @Success 200 {object} map[string]string "Suggested name"
// @Failure 400 {object} errors.HTTPError "Bad Request - Invalid request data"
// @Security BearerAuth
// @Router /presets/suggest-name [post]
func (h *Handler) SuggestName(c *gin.Context) {
	var req usecases.SuggestNameRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		coreerrors.Respond(c, err)
		return
	}

	name, err := h.manageUC.SuggestName(&req)
	if err != nil {
		coreerrors.Respond(c, coreerrors.BadRequest("invalid_preset_type", "Tipo de preset inválido"))
		return
	}

	c.JSON(http.StatusOK, gin.H{"name": name})
}

// GetTemplates returns the static preset templates catalog, paginated with the standard envelope.
// @Summary List preset templates
// @Description Retrieve the static catalog of preset templates (approximate starting points to adjust), optionally filtered by type, paginated with the standard envelope.
// @Tags Presets
// @Accept json
// @Produce json
// @Param type query string false "Template type filter (machine, energy, cost)"
// @Param page query int false "Page number (default 1)"
// @Param page_size query int false "Items per page (default 100, max 100)"
// @Param q query string false "Free-text search on name"
// @Success 200 {object} helpers.Page[entities.PresetTemplate] "Paginated templates"
// @Security BearerAuth
// @Router /presets/templates [get]
func (h *Handler) GetTemplates(c *gin.Context) {
	presetType := entities.PresetType(c.Query("type"))
	templates := entities.TemplatesByType(presetType)

	q := helpers.ParseListQuery(c, presetListOptions())
	if q.Search != "" {
		needle := strings.ToLower(q.Search)
		filtered := make([]entities.PresetTemplate, 0, len(templates))
		for _, t := range templates {
			if strings.Contains(strings.ToLower(t.Name), needle) {
				filtered = append(filtered, t)
			}
		}
		templates = filtered
	}

	total := int64(len(templates))
	off := q.Offset()
	if off > len(templates) {
		off = len(templates)
	}
	end := len(templates)
	if q.Limit() > 0 {
		end = off + q.Limit()
		if end > len(templates) {
			end = len(templates)
		}
	}

	c.JSON(http.StatusOK, helpers.NewPage(templates[off:end], total, q))
}

// CreateFromTemplate creates an organization preset from a template.
// @Summary Create preset from template
// @Description Instantiate a preset in the caller's organization from a template, with optional overrides (name, is_default).
// @Tags Presets
// @Accept json
// @Produce json
// @Param key path string true "Template key"
// @Param request body usecases.FromTemplateOverrides false "Optional overrides"
// @Success 201 {object} entities.PresetEntity "Preset created from template"
// @Failure 400 {object} errors.HTTPError "Bad Request - Invalid request data"
// @Failure 404 {object} errors.HTTPError "Not Found - Unknown template key"
// @Failure 500 {object} errors.HTTPError "Internal Server Error"
// @Security BearerAuth
// @Router /presets/from-template/{key} [post]
func (h *Handler) CreateFromTemplate(c *gin.Context) {
	organizationID, ok := requireOrganizationID(c)
	if !ok {
		return
	}

	key := c.Param("key")

	var overrides usecases.FromTemplateOverrides
	// Body is optional; ignore EOF/empty-body bind errors.
	_ = c.ShouldBindJSON(&overrides)

	preset, err := h.createUC.CreateFromTemplate(key, overrides, organizationID, authenticatedUserID(c))
	if err != nil {
		if stderrors.Is(err, entities.ErrTemplateNotFound) {
			coreerrors.Respond(c, coreerrors.NotFoundErr("preset_template_not_found", "Template de preset não encontrado"))
			return
		}
		respondPresetError(c, err)
		return
	}

	c.JSON(http.StatusCreated, preset)

	h.activityService.Record(c.Request.Context(), activityEntities.ActivityEntity{
		OrganizationID: organizationID,
		UserID:         helpers.GetUserID(c),
		Action:         activityEntities.ActionCreated,
		EntityType:     activityEntities.EntityPreset,
		EntityID:       preset.ID.String(),
		EntityName:     preset.Name,
		CreatedAt:      time.Now(),
	})
}

// SetDefault marks a preset as the default for its type.
// @Summary Set preset as default
// @Description Mark the preset as the single default for its (organization, type); clears any other default of the same type.
// @Tags Presets
// @Accept json
// @Produce json
// @Param id path string true "Preset ID (UUID format)"
// @Success 200 {object} entities.PresetEntity "Preset set as default"
// @Failure 400 {object} errors.HTTPError "Bad Request - Invalid ID format"
// @Failure 404 {object} errors.HTTPError "Not Found - Preset not found"
// @Failure 500 {object} errors.HTTPError "Internal Server Error"
// @Security BearerAuth
// @Router /presets/{id}/default [post]
func (h *Handler) SetDefault(c *gin.Context) {
	organizationID, ok := requireOrganizationID(c)
	if !ok {
		return
	}

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		coreerrors.Respond(c, coreerrors.BadRequest("invalid_preset_id", "ID de preset inválido"))
		return
	}

	preset, err := h.manageUC.SetDefault(id, organizationID)
	if err != nil {
		respondPresetError(c, err)
		return
	}

	c.JSON(http.StatusOK, preset)

	h.activityService.Record(c.Request.Context(), activityEntities.ActivityEntity{
		OrganizationID: organizationID,
		UserID:         helpers.GetUserID(c),
		Action:         activityEntities.ActionUpdated,
		EntityType:     activityEntities.EntityPreset,
		EntityID:       preset.ID.String(),
		EntityName:     preset.Name,
		CreatedAt:      time.Now(),
	})
}

// Duplicate copies a preset within the organization.
// @Summary Duplicate preset
// @Description Copy a preset (base + type-specific data) within the caller's organization. The copy is named "<name> (cópia)" and is not a default.
// @Tags Presets
// @Accept json
// @Produce json
// @Param id path string true "Preset ID (UUID format)"
// @Success 201 {object} entities.PresetEntity "Preset duplicated"
// @Failure 400 {object} errors.HTTPError "Bad Request - Invalid ID format"
// @Failure 404 {object} errors.HTTPError "Not Found - Preset not found"
// @Failure 500 {object} errors.HTTPError "Internal Server Error"
// @Security BearerAuth
// @Router /presets/{id}/duplicate [post]
func (h *Handler) Duplicate(c *gin.Context) {
	organizationID, ok := requireOrganizationID(c)
	if !ok {
		return
	}

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		coreerrors.Respond(c, coreerrors.BadRequest("invalid_preset_id", "ID de preset inválido"))
		return
	}

	preset, err := h.manageUC.Duplicate(id, organizationID)
	if err != nil {
		respondPresetError(c, err)
		return
	}

	c.JSON(http.StatusCreated, preset)

	h.activityService.Record(c.Request.Context(), activityEntities.ActivityEntity{
		OrganizationID: organizationID,
		UserID:         helpers.GetUserID(c),
		Action:         activityEntities.ActionCreated,
		EntityType:     activityEntities.EntityPreset,
		EntityID:       preset.ID.String(),
		EntityName:     preset.Name,
		CreatedAt:      time.Now(),
	})
}
