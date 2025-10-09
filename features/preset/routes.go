package preset

import (
	"net/http"

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

// SetupRoutes configures the preset routes
func SetupRoutes(router *gin.RouterGroup, handler *Handler) {
	presets := router.Group("/presets")
	{
		// Base preset routes
		presets.GET("", handler.GetPresets)
		presets.GET("/:id", handler.GetPresetByID)
		presets.DELETE("/:id", handler.DeletePreset)

		// Machine preset routes
		machines := presets.Group("/machines")
		{
			machines.POST("", handler.CreateMachinePreset)
			machines.GET("", handler.GetMachinePresets)
			machines.GET("/:id", handler.GetMachinePresetByID)
			machines.PUT("/:id", handler.UpdateMachinePreset)
			machines.GET("/brand/:brand", handler.GetMachinePresetsByBrand)
		}

		// Energy preset routes
		energy := presets.Group("/energy")
		{
			energy.POST("", handler.CreateEnergyPreset)
			energy.GET("", handler.GetEnergyPresets)
			energy.GET("/:id", handler.GetEnergyPresetByID)
			energy.PUT("/:id", handler.UpdateEnergyPreset)
			energy.GET("/location", handler.GetEnergyPresetsByLocation)
			energy.GET("/currency/:currency", handler.GetEnergyPresetsByCurrency)
		}

		// Cost preset routes
		costs := presets.Group("/costs")
		{
			costs.POST("", handler.CreateCostPreset)
			costs.GET("", handler.GetCostPresets)
			costs.GET("/:id", handler.GetCostPresetByID)
			costs.PUT("/:id", handler.UpdateCostPreset)
		}
	}
}

// GetPresets retrieves presets with optional filters
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
func (h *Handler) GetMachinePresets(c *gin.Context) {
	presets, err := h.findUC.FindByType(entities.PresetTypeMachine)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, presets)
}

// GetMachinePresetByID retrieves a machine preset with full details
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
func (h *Handler) GetEnergyPresets(c *gin.Context) {
	presets, err := h.findUC.FindByType(entities.PresetTypeEnergy)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, presets)
}

// GetEnergyPresetByID retrieves an energy preset with full details
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
func (h *Handler) GetCostPresets(c *gin.Context) {
	presets, err := h.findUC.FindByType(entities.PresetTypeCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, presets)
}

// GetCostPresetByID retrieves a cost preset with full details
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
