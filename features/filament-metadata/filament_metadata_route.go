package filament_metadata

import (
	"github.com/RodolfoBonis/spooliq/core/roles"
	"github.com/RodolfoBonis/spooliq/features/filament-metadata/domain/usecases"
	"github.com/gin-gonic/gin"
)

// CreateBrandHandler handles creating a new brand.
// @Summary Create Brand
// @Schemes
// @Description Create a new filament brand
// @Tags Filament Metadata
// @Accept json
// @Produce json
// @Param request body usecases.CreateBrandRequest true "Brand data"
// @Success 201 {object} usecases.BrandResponse "Successfully created brand"
// @Failure 400 {object} errors.HTTPError
// @Failure 401 {object} errors.HTTPError
// @Failure 409 {object} errors.HTTPError
// @Failure 500 {object} errors.HTTPError
// @Router /filament-brands [post]
// @Security Bearer
func CreateBrandHandler(brandUC usecases.BrandUseCase) gin.HandlerFunc {
	return func(c *gin.Context) {
		brandUC.CreateBrand(c)
	}
}

// GetBrandHandler handles getting a brand by ID.
// @Summary Get Brand
// @Schemes
// @Description Get a brand by its ID
// @Tags Filament Metadata
// @Accept json
// @Produce json
// @Param id path int true "Brand ID"
// @Success 200 {object} usecases.BrandResponse "Successfully retrieved brand"
// @Failure 400 {object} errors.HTTPError
// @Failure 404 {object} errors.HTTPError
// @Failure 500 {object} errors.HTTPError
// @Router /filament-brands/{id} [get]
func GetBrandHandler(brandUC usecases.BrandUseCase) gin.HandlerFunc {
	return func(c *gin.Context) {
		brandUC.GetBrand(c)
	}
}

// GetAllBrandsHandler handles getting all brands.
// @Summary Get All Brands
// @Schemes
// @Description Get all filament brands
// @Tags Filament Metadata
// @Accept json
// @Produce json
// @Param active_only query bool false "Filter only active brands"
// @Success 200 {object} usecases.BrandListResponse "Successfully retrieved brands"
// @Failure 500 {object} errors.HTTPError
// @Router /filament-brands [get]
func GetAllBrandsHandler(brandUC usecases.BrandUseCase) gin.HandlerFunc {
	return func(c *gin.Context) {
		brandUC.GetAllBrands(c)
	}
}

// UpdateBrandHandler handles updating a brand.
// @Summary Update Brand
// @Schemes
// @Description Update an existing brand (admin only)
// @Tags Filament Metadata
// @Accept json
// @Produce json
// @Param id path int true "Brand ID"
// @Param request body usecases.UpdateBrandRequest true "Updated brand data"
// @Success 200 {object} usecases.BrandResponse "Successfully updated brand"
// @Failure 400 {object} errors.HTTPError
// @Failure 401 {object} errors.HTTPError
// @Failure 404 {object} errors.HTTPError
// @Failure 409 {object} errors.HTTPError
// @Failure 500 {object} errors.HTTPError
// @Router /filament-brands/{id} [put]
// @Security Bearer
func UpdateBrandHandler(brandUC usecases.BrandUseCase) gin.HandlerFunc {
	return func(c *gin.Context) {
		brandUC.UpdateBrand(c)
	}
}

// DeleteBrandHandler handles deleting a brand.
// @Summary Delete Brand
// @Schemes
// @Description Delete/deactivate a brand (admin only)
// @Tags Filament Metadata
// @Accept json
// @Produce json
// @Param id path int true "Brand ID"
// @Success 204 "Successfully deleted brand"
// @Failure 400 {object} errors.HTTPError
// @Failure 401 {object} errors.HTTPError
// @Failure 404 {object} errors.HTTPError
// @Failure 500 {object} errors.HTTPError
// @Router /filament-brands/{id} [delete]
// @Security Bearer
func DeleteBrandHandler(brandUC usecases.BrandUseCase) gin.HandlerFunc {
	return func(c *gin.Context) {
		brandUC.DeleteBrand(c)
	}
}

// CreateMaterialHandler handles creating a new material.
// @Summary Create Material
// @Schemes
// @Description Create a new filament material
// @Tags Filament Metadata
// @Accept json
// @Produce json
// @Param request body usecases.CreateMaterialRequest true "Material data"
// @Success 201 {object} usecases.MaterialResponse "Successfully created material"
// @Failure 400 {object} errors.HTTPError
// @Failure 401 {object} errors.HTTPError
// @Failure 409 {object} errors.HTTPError
// @Failure 500 {object} errors.HTTPError
// @Router /filament-materials [post]
// @Security Bearer
func CreateMaterialHandler(materialUC usecases.MaterialUseCase) gin.HandlerFunc {
	return func(c *gin.Context) {
		materialUC.CreateMaterial(c)
	}
}

// GetMaterialHandler handles getting a material by ID.
// @Summary Get Material
// @Schemes
// @Description Get a material by its ID
// @Tags Filament Metadata
// @Accept json
// @Produce json
// @Param id path int true "Material ID"
// @Success 200 {object} usecases.MaterialResponse "Successfully retrieved material"
// @Failure 400 {object} errors.HTTPError
// @Failure 404 {object} errors.HTTPError
// @Failure 500 {object} errors.HTTPError
// @Router /filament-materials/{id} [get]
func GetMaterialHandler(materialUC usecases.MaterialUseCase) gin.HandlerFunc {
	return func(c *gin.Context) {
		materialUC.GetMaterial(c)
	}
}

// GetAllMaterialsHandler handles getting all materials.
// @Summary Get All Materials
// @Schemes
// @Description Get all filament materials
// @Tags Filament Metadata
// @Accept json
// @Produce json
// @Param active_only query bool false "Filter only active materials"
// @Success 200 {object} usecases.MaterialListResponse "Successfully retrieved materials"
// @Failure 500 {object} errors.HTTPError
// @Router /filament-materials [get]
func GetAllMaterialsHandler(materialUC usecases.MaterialUseCase) gin.HandlerFunc {
	return func(c *gin.Context) {
		materialUC.GetAllMaterials(c)
	}
}

// UpdateMaterialHandler handles updating a material.
// @Summary Update Material
// @Schemes
// @Description Update an existing material (admin only)
// @Tags Filament Metadata
// @Accept json
// @Produce json
// @Param id path int true "Material ID"
// @Param request body usecases.UpdateMaterialRequest true "Updated material data"
// @Success 200 {object} usecases.MaterialResponse "Successfully updated material"
// @Failure 400 {object} errors.HTTPError
// @Failure 401 {object} errors.HTTPError
// @Failure 404 {object} errors.HTTPError
// @Failure 409 {object} errors.HTTPError
// @Failure 500 {object} errors.HTTPError
// @Router /filament-materials/{id} [put]
// @Security Bearer
func UpdateMaterialHandler(materialUC usecases.MaterialUseCase) gin.HandlerFunc {
	return func(c *gin.Context) {
		materialUC.UpdateMaterial(c)
	}
}

// DeleteMaterialHandler handles deleting a material.
// @Summary Delete Material
// @Schemes
// @Description Delete/deactivate a material (admin only)
// @Tags Filament Metadata
// @Accept json
// @Produce json
// @Param id path int true "Material ID"
// @Success 204 "Successfully deleted material"
// @Failure 400 {object} errors.HTTPError
// @Failure 401 {object} errors.HTTPError
// @Failure 404 {object} errors.HTTPError
// @Failure 500 {object} errors.HTTPError
// @Router /filament-materials/{id} [delete]
// @Security Bearer
func DeleteMaterialHandler(materialUC usecases.MaterialUseCase) gin.HandlerFunc {
	return func(c *gin.Context) {
		materialUC.DeleteMaterial(c)
	}
}

// Routes registers filament metadata routes for the application.
func Routes(route *gin.RouterGroup, brandUC usecases.BrandUseCase, materialUC usecases.MaterialUseCase, protectFactory func(handler gin.HandlerFunc, role string) gin.HandlerFunc) {
	// Brand routes
	brands := route.Group("/filament-brands")
	{
		// Public routes
		brands.GET("", GetAllBrandsHandler(brandUC))
		brands.GET("/:id", GetBrandHandler(brandUC))

		// Protected routes (admin only for create/update/delete)
		brands.POST("", protectFactory(CreateBrandHandler(brandUC), roles.AdminRole))
		brands.PUT("/:id", protectFactory(UpdateBrandHandler(brandUC), roles.AdminRole))
		brands.DELETE("/:id", protectFactory(DeleteBrandHandler(brandUC), roles.AdminRole))
	}

	// Material routes
	materials := route.Group("/filament-materials")
	{
		// Public routes
		materials.GET("", GetAllMaterialsHandler(materialUC))
		materials.GET("/:id", GetMaterialHandler(materialUC))

		// Protected routes (admin only for create/update/delete)
		materials.POST("", protectFactory(CreateMaterialHandler(materialUC), roles.AdminRole))
		materials.PUT("/:id", protectFactory(UpdateMaterialHandler(materialUC), roles.AdminRole))
		materials.DELETE("/:id", protectFactory(DeleteMaterialHandler(materialUC), roles.AdminRole))
	}
}
