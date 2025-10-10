package preset

import (
	"net/http"

	"github.com/RodolfoBonis/spooliq/core/roles"
	"github.com/RodolfoBonis/spooliq/features/preset/domain/entities"
	"github.com/RodolfoBonis/spooliq/features/preset/domain/usecases"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// Handler handles HTTP requests for preset operations
type Handler struct {
	createUC *usecases.CreatePresetUseCase
	findUC   *usecases.FindPresetUseCase
	updateUC *usecases.UpdatePresetUseCase
	deleteUC *usecases.DeletePresetUseCase
}

// NewPresetHandler creates a new preset handler
func NewPresetHandler(
	createUC *usecases.CreatePresetUseCase,
	findUC *usecases.FindPresetUseCase,
	updateUC *usecases.UpdatePresetUseCase,
	deleteUC *usecases.DeletePresetUseCase,
) *Handler {
	return &Handler{
		createUC: createUC,
		findUC:   findUC,
		updateUC: updateUC,
		deleteUC: deleteUC,
	}
}

// SetupRoutes configures the preset routes with authentication middleware
func SetupRoutes(router *gin.RouterGroup, handler *Handler, protectFactory func(handler gin.HandlerFunc, role string) gin.HandlerFunc) {
	presets := router.Group("/presets")
	{
		// Base preset routes - GET endpoints accessible to all authenticated users
		presets.GET("", protectFactory(handler.GetPresets, roles.UserRole))
		presets.GET("/:id", protectFactory(handler.GetPresetByID, roles.UserRole))
		presets.DELETE("/:id", protectFactory(handler.DeletePreset, roles.UserRole))

		// Machine preset routes
		machines := presets.Group("/machines")
		{
			machines.POST("", protectFactory(handler.CreateMachinePreset, roles.UserRole))
			machines.GET("", protectFactory(handler.GetMachinePresets, roles.UserRole))
			machines.GET("/:id", protectFactory(handler.GetMachinePresetByID, roles.UserRole))
			machines.PUT("/:id", protectFactory(handler.UpdateMachinePreset, roles.UserRole))
			machines.GET("/brand/:brand", protectFactory(handler.GetMachinePresetsByBrand, roles.UserRole))
		}

		// Energy preset routes
		energy := presets.Group("/energy")
		{
			energy.POST("", protectFactory(handler.CreateEnergyPreset, roles.UserRole))
			energy.GET("", protectFactory(handler.GetEnergyPresets, roles.UserRole))
			energy.GET("/:id", protectFactory(handler.GetEnergyPresetByID, roles.UserRole))
			energy.PUT("/:id", protectFactory(handler.UpdateEnergyPreset, roles.UserRole))
			energy.GET("/location", protectFactory(handler.GetEnergyPresetsByLocation, roles.UserRole))
			energy.GET("/currency/:currency", protectFactory(handler.GetEnergyPresetsByCurrency, roles.UserRole))
		}

		// Cost preset routes
		costs := presets.Group("/costs")
		{
			costs.POST("", protectFactory(handler.CreateCostPreset, roles.UserRole))
			costs.GET("", protectFactory(handler.GetCostPresets, roles.UserRole))
			costs.GET("/:id", protectFactory(handler.GetCostPresetByID, roles.UserRole))
			costs.PUT("/:id", protectFactory(handler.UpdateCostPreset, roles.UserRole))
		}
	}
}

// GetPresets retrieves presets with optional filters
// @Summary Get presets with filters
// @Description Retrieve presets with optional filters including type, active status, default status, global status, and user ID
// @Tags Presets
// @Accept json
// @Produce json
// @Param type query string false "Preset type filter (machine, energy, cost)"
// @Param active query boolean false "Filter only active presets"
// @Param default query boolean false "Filter only default presets"
// @Param global query boolean false "Filter only global presets"
// @Param user_id query string false "Filter presets by user ID (UUID format)"
// @Success 200 {object} interface{} "Successfully retrieved presets"
// @Failure 400 {object} errors.HTTPError "Bad Request - Invalid user ID format"
// @Failure 500 {object} errors.HTTPError "Internal Server Error"
// @Security BearerAuth
// @Router /presets [get]
func (h *Handler) GetPresets(c *gin.Context) {
	presetType := c.Query("type")
	activeOnly := c.Query("active") == "true"
	defaultOnly := c.Query("default") == "true"
	globalOnly := c.Query("global") == "true"
	userIDStr := c.Query("user_id")

	var presets interface{}
	var err error

	switch {
	case userIDStr != "":
		userID, parseErr := uuid.Parse(userIDStr)
		if parseErr != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID format"})
			return
		}
		presets, err = h.findUC.FindByUserID(userID)
	case globalOnly:
		presets, err = h.findUC.FindGlobalPresets()
	case activeOnly:
		presets, err = h.findUC.FindActivePresets()
	case defaultOnly:
		presets, err = h.findUC.FindDefaultPresets()
	case presetType != "":
		presets, err = h.findUC.FindByType(entities.PresetType(presetType))
	default:
		presets, err = h.findUC.FindActivePresets() // Default to active presets
	}

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, presets)
}

// GetPresetByID retrieves a preset by ID
// @Summary Get preset by ID
// @Description Retrieve a specific preset by its unique identifier
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
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID format"})
		return
	}

	preset, err := h.findUC.FindByID(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Preset not found"})
		return
	}

	c.JSON(http.StatusOK, preset)
}

// DeletePreset deletes a preset by ID
// @Summary Delete preset
// @Description Delete a preset by its unique identifier
// @Tags Presets
// @Accept json
// @Produce json
// @Param id path string true "Preset ID (UUID format)"
// @Success 204 "Preset deleted successfully"
// @Failure 400 {object} errors.HTTPError "Bad Request - Invalid ID format"
// @Failure 500 {object} errors.HTTPError "Internal Server Error"
// @Security BearerAuth
// @Router /presets/{id} [delete]
func (h *Handler) DeletePreset(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID format"})
		return
	}

	if err := h.deleteUC.Execute(id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusNoContent, nil)
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
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	preset, err := h.createUC.CreateMachinePreset(&req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, preset)
}

// GetMachinePresets retrieves all machine presets
// @Summary Get all machine presets
// @Description Retrieve all available machine presets with their specifications
// @Tags Machine Presets
// @Accept json
// @Produce json
// @Success 200 {array} entities.PresetEntity "Successfully retrieved machine presets"
// @Failure 500 {object} errors.HTTPError "Internal Server Error"
// @Security BearerAuth
// @Router /presets/machines [get]
func (h *Handler) GetMachinePresets(c *gin.Context) {
	presets, err := h.findUC.FindByType(entities.PresetTypeMachine)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, presets)
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
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID format"})
		return
	}

	preset, err := h.findUC.FindMachinePresetByID(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Machine preset not found"})
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
// @Failure 500 {object} errors.HTTPError "Internal Server Error"
// @Security BearerAuth
// @Router /presets/machines/{id} [put]
func (h *Handler) UpdateMachinePreset(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID format"})
		return
	}

	var req usecases.UpdateMachinePresetRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	req.ID = id

	preset, err := h.updateUC.UpdateMachinePreset(&req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, preset)
}

// GetMachinePresetsByBrand retrieves machine presets by brand
// @Summary Get machine presets by brand
// @Description Retrieve all machine presets from a specific brand
// @Tags Machine Presets
// @Accept json
// @Produce json
// @Param brand path string true "Machine brand name"
// @Success 200 {array} entities.MachinePresetEntity "Successfully retrieved machine presets by brand"
// @Failure 500 {object} errors.HTTPError "Internal Server Error"
// @Security BearerAuth
// @Router /presets/machines/brand/{brand} [get]
func (h *Handler) GetMachinePresetsByBrand(c *gin.Context) {
	brand := c.Param("brand")

	presets, err := h.findUC.FindMachinePresetsByBrand(brand)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, presets)
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
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	preset, err := h.createUC.CreateEnergyPreset(&req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, preset)
}

// GetEnergyPresets retrieves all energy presets
// @Summary Get all energy presets
// @Description Retrieve all available energy presets with pricing and location data
// @Tags Energy Presets
// @Accept json
// @Produce json
// @Success 200 {array} entities.PresetEntity "Successfully retrieved energy presets"
// @Failure 500 {object} errors.HTTPError "Internal Server Error"
// @Security BearerAuth
// @Router /presets/energy [get]
func (h *Handler) GetEnergyPresets(c *gin.Context) {
	presets, err := h.findUC.FindByType(entities.PresetTypeEnergy)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, presets)
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
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID format"})
		return
	}

	preset, err := h.findUC.FindEnergyPresetByID(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Energy preset not found"})
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
// @Failure 500 {object} errors.HTTPError "Internal Server Error"
// @Security BearerAuth
// @Router /presets/energy/{id} [put]
func (h *Handler) UpdateEnergyPreset(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID format"})
		return
	}

	var req usecases.UpdateEnergyPresetRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	req.ID = id

	preset, err := h.updateUC.UpdateEnergyPreset(&req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, preset)
}

// GetEnergyPresetsByLocation retrieves energy presets by location
// @Summary Get energy presets by location
// @Description Retrieve energy presets filtered by country, state, and/or city
// @Tags Energy Presets
// @Accept json
// @Produce json
// @Param country query string false "Filter by country"
// @Param state query string false "Filter by state/province"
// @Param city query string false "Filter by city"
// @Success 200 {array} entities.EnergyPresetEntity "Successfully retrieved energy presets by location"
// @Failure 500 {object} errors.HTTPError "Internal Server Error"
// @Security BearerAuth
// @Router /presets/energy/location [get]
func (h *Handler) GetEnergyPresetsByLocation(c *gin.Context) {
	country := c.Query("country")
	state := c.Query("state")
	city := c.Query("city")

	presets, err := h.findUC.FindEnergyPresetsByLocation(country, state, city)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, presets)
}

// GetEnergyPresetsByCurrency retrieves energy presets by currency
// @Summary Get energy presets by currency
// @Description Retrieve energy presets that use a specific currency (3-letter currency code)
// @Tags Energy Presets
// @Accept json
// @Produce json
// @Param currency path string true "Currency code (3 letters, e.g., USD, EUR, BRL)"
// @Success 200 {array} entities.EnergyPresetEntity "Successfully retrieved energy presets by currency"
// @Failure 500 {object} errors.HTTPError "Internal Server Error"
// @Security BearerAuth
// @Router /presets/energy/currency/{currency} [get]
func (h *Handler) GetEnergyPresetsByCurrency(c *gin.Context) {
	currency := c.Param("currency")

	presets, err := h.findUC.FindEnergyPresetsByCurrency(currency)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, presets)
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
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	preset, err := h.createUC.CreateCostPreset(&req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, preset)
}

// GetCostPresets retrieves all cost presets
// @Summary Get all cost presets
// @Description Retrieve all available cost presets with pricing and margin configurations
// @Tags Cost Presets
// @Accept json
// @Produce json
// @Success 200 {array} entities.PresetEntity "Successfully retrieved cost presets"
// @Failure 500 {object} errors.HTTPError "Internal Server Error"
// @Security BearerAuth
// @Router /presets/costs [get]
func (h *Handler) GetCostPresets(c *gin.Context) {
	presets, err := h.findUC.FindByType(entities.PresetTypeCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, presets)
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
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID format"})
		return
	}

	preset, err := h.findUC.FindCostPresetByID(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Cost preset not found"})
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
// @Failure 500 {object} errors.HTTPError "Internal Server Error"
// @Security BearerAuth
// @Router /presets/costs/{id} [put]
func (h *Handler) UpdateCostPreset(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID format"})
		return
	}

	var req usecases.UpdateCostPresetRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	req.ID = id

	preset, err := h.updateUC.UpdateCostPreset(&req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, preset)
}
