package filaments

import (
	"github.com/RodolfoBonis/spooliq/core/roles"
	"github.com/RodolfoBonis/spooliq/features/filaments/domain/usecases"
	"github.com/gin-gonic/gin"
)

// CreateFilamentHandler handles creating a new filament.
// @Summary Create Filament
// @Schemes
// @Description Create a new filament record
// @Tags Filaments
// @Accept json
// @Produce json
// @Param request body usecases.CreateFilamentRequest true "Filament data"
// @Success 201 {object} usecases.FilamentResponse "Successfully created filament"
// @Failure 400 {object} errors.HTTPError
// @Failure 401 {object} errors.HTTPError
// @Failure 500 {object} errors.HTTPError
// @Router /filaments [post]
// @Security Bearer
func CreateFilamentHandler(filamentsUc usecases.FilamentUseCase) gin.HandlerFunc {
	return func(c *gin.Context) {
		filamentsUc.CreateFilament(c)
	}
}

// GetFilamentHandler handles getting a filament by ID.
// @Summary Get Filament
// @Schemes
// @Description Get a filament by its ID
// @Tags Filaments
// @Accept json
// @Produce json
// @Param id path int true "Filament ID"
// @Success 200 {object} usecases.FilamentResponse "Successfully retrieved filament"
// @Failure 400 {object} errors.HTTPError
// @Failure 404 {object} errors.HTTPError
// @Failure 500 {object} errors.HTTPError
// @Router /filaments/{id} [get]
func GetFilamentHandler(filamentsUc usecases.FilamentUseCase) gin.HandlerFunc {
	return func(c *gin.Context) {
		filamentsUc.GetFilament(c)
	}
}

// GetAllFilamentsHandler handles getting all accessible filaments.
// @Summary Get All Filaments
// @Schemes
// @Description Get all filaments accessible to the user (global + user's own)
// @Tags Filaments
// @Accept json
// @Produce json
// @Success 200 {object} usecases.ListResponse "Successfully retrieved filaments"
// @Failure 500 {object} errors.HTTPError
// @Router /filaments [get]
func GetAllFilamentsHandler(filamentsUc usecases.FilamentUseCase) gin.HandlerFunc {
	return func(c *gin.Context) {
		filamentsUc.GetAllFilaments(c)
	}
}

// UpdateFilamentHandler handles updating a filament.
// @Summary Update Filament
// @Schemes
// @Description Update an existing filament (only user's own filaments)
// @Tags Filaments
// @Accept json
// @Produce json
// @Param id path int true "Filament ID"
// @Param request body usecases.UpdateFilamentRequest true "Updated filament data"
// @Success 200 {object} usecases.FilamentResponse "Successfully updated filament"
// @Failure 400 {object} errors.HTTPError
// @Failure 401 {object} errors.HTTPError
// @Failure 404 {object} errors.HTTPError
// @Failure 500 {object} errors.HTTPError
// @Router /filaments/{id} [put]
// @Security Bearer
func UpdateFilamentHandler(filamentsUc usecases.FilamentUseCase) gin.HandlerFunc {
	return func(c *gin.Context) {
		filamentsUc.UpdateFilament(c)
	}
}

// DeleteFilamentHandler handles deleting a filament.
// @Summary Delete Filament
// @Schemes
// @Description Delete a filament (only user's own filaments)
// @Tags Filaments
// @Accept json
// @Produce json
// @Param id path int true "Filament ID"
// @Success 204 "Successfully deleted filament"
// @Failure 400 {object} errors.HTTPError
// @Failure 401 {object} errors.HTTPError
// @Failure 404 {object} errors.HTTPError
// @Failure 500 {object} errors.HTTPError
// @Router /filaments/{id} [delete]
// @Security Bearer
func DeleteFilamentHandler(filamentsUc usecases.FilamentUseCase) gin.HandlerFunc {
	return func(c *gin.Context) {
		filamentsUc.DeleteFilament(c)
	}
}

// GetUserFilamentsHandler handles getting user's own filaments.
// @Summary Get User Filaments
// @Schemes
// @Description Get all filaments owned by the authenticated user
// @Tags Filaments
// @Accept json
// @Produce json
// @Success 200 {object} ListResponse "Successfully retrieved user filaments"
// @Failure 401 {object} errors.HTTPError
// @Failure 500 {object} errors.HTTPError
// @Router /filaments/my [get]
// @Security Bearer
func GetUserFilamentsHandler(filamentsUc usecases.FilamentUseCase) gin.HandlerFunc {
	return func(c *gin.Context) {
		filamentsUc.GetUserFilaments(c)
	}
}

// GetGlobalFilamentsHandler handles getting global filaments.
// @Summary Get Global Filaments
// @Schemes
// @Description Get all global filaments (not owned by any user)
// @Tags Filaments
// @Accept json
// @Produce json
// @Success 200 {object} ListResponse "Successfully retrieved global filaments"
// @Failure 500 {object} errors.HTTPError
// @Router /filaments/global [get]
func GetGlobalFilamentsHandler(filamentsUc usecases.FilamentUseCase) gin.HandlerFunc {
	return func(c *gin.Context) {
		filamentsUc.GetGlobalFilaments(c)
	}
}

// Routes registers filament routes for the application.
func Routes(route *gin.RouterGroup, filamentsUC usecases.FilamentUseCase, protectFactory func(handler gin.HandlerFunc, role string) gin.HandlerFunc) {
	filaments := route.Group("/filaments")

	// Public routes (can be accessed without authentication, but will show different data based on auth status)
	filaments.GET("", GetAllFilamentsHandler(filamentsUC))
	filaments.GET("/global", GetGlobalFilamentsHandler(filamentsUC))
	filaments.GET("/:id", GetFilamentHandler(filamentsUC))

	// Protected routes (require authentication)
	filaments.POST("", protectFactory(CreateFilamentHandler(filamentsUC), roles.UserRole))
	filaments.GET("/my", protectFactory(GetUserFilamentsHandler(filamentsUC), roles.UserRole))
	filaments.PUT("/:id", protectFactory(UpdateFilamentHandler(filamentsUC), roles.UserRole))
	filaments.DELETE("/:id", protectFactory(DeleteFilamentHandler(filamentsUC), roles.UserRole))
}
