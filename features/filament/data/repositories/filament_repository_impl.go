package repositories

import (
	"context"
	"fmt"

	"github.com/RodolfoBonis/spooliq/core/helpers"
	"github.com/RodolfoBonis/spooliq/features/filament/data/models"
	"github.com/RodolfoBonis/spooliq/features/filament/domain/entities"
	"github.com/RodolfoBonis/spooliq/features/filament/domain/repositories"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type filamentRepositoryImpl struct {
	db *gorm.DB
}

// NewFilamentRepository creates a new instance of FilamentRepository
func NewFilamentRepository(db *gorm.DB) repositories.FilamentRepository {
	return &filamentRepositoryImpl{db: db}
}

// Create creates a new filament in the database
func (r *filamentRepositoryImpl) Create(ctx context.Context, filament *entities.FilamentEntity) error {
	model := &models.FilamentModel{}
	model.FromEntity(filament)

	if err := r.db.WithContext(ctx).Create(model).Error; err != nil {
		return fmt.Errorf("failed to create filament: %w", err)
	}

	// Update entity with generated values
	*filament = *model.ToEntity()
	return nil
}

// FindByID finds a filament by ID with organization filtering
func (r *filamentRepositoryImpl) FindByID(ctx context.Context, id uuid.UUID, organizationID string) (*entities.FilamentEntity, error) {
	model := &models.FilamentModel{}

	if err := r.db.WithContext(ctx).
		Preload("Brand").
		Preload("Material").
		Preload("User").
		Where("id = ? AND organization_id = ?", id, organizationID).
		First(model).Error; err != nil {
		return nil, err
	}

	return model.ToEntity(), nil
}

// filamentUpdatableColumns are the columns a PUT is allowed to overwrite. They
// are passed to Select so GORM persists explicit zero values (e.g. is_active
// false) instead of skipping them, while ownership, tenant and creation columns
// stay immutable.
var filamentUpdatableColumns = []string{
	"name", "description", "brand_id", "material_id",
	"color", "color_hex", "color_type", "color_data", "color_preview",
	"diameter", "weight", "price_per_kg", "url",
	"print_temperature", "bed_temperature", "is_active",
	// Stock control: track_stock and the alert threshold are editable via PUT.
	// stock_grams is intentionally absent — it changes only through movements.
	"track_stock", "low_stock_threshold_grams", "updated_at",
}

// Update updates an existing filament. The update is organization-scoped and
// uses Select so a full PUT persists zero values.
func (r *filamentRepositoryImpl) Update(ctx context.Context, filament *entities.FilamentEntity) error {
	model := &models.FilamentModel{}
	model.FromEntity(filament)

	if err := r.db.WithContext(ctx).
		Model(&models.FilamentModel{}).
		Where("id = ? AND organization_id = ?", filament.ID, filament.OrganizationID).
		Select(filamentUpdatableColumns).
		Updates(model).Error; err != nil {
		return fmt.Errorf("failed to update filament: %w", err)
	}

	return nil
}

// Delete soft deletes a filament
func (r *filamentRepositoryImpl) Delete(ctx context.Context, id uuid.UUID) error {
	if err := r.db.WithContext(ctx).Delete(&models.FilamentModel{}, "id = ?", id).Error; err != nil {
		return fmt.Errorf("failed to delete filament: %w", err)
	}

	return nil
}

// FindAll retrieves a page of filaments for the organization.
func (r *filamentRepositoryImpl) FindAll(ctx context.Context, organizationID, search, order string, limit, offset int) ([]*entities.FilamentEntity, int64, error) {
	return r.list(ctx, organizationID, nil, search, order, limit, offset)
}

// SearchFilaments retrieves a page of filaments matching the structured filters
// and the free-text search.
func (r *filamentRepositoryImpl) SearchFilaments(ctx context.Context, organizationID string, filters map[string]interface{}, search, order string, limit, offset int) ([]*entities.FilamentEntity, int64, error) {
	return r.list(ctx, organizationID, filters, search, order, limit, offset)
}

// list is the shared query builder behind FindAll and SearchFilaments. All
// queries are organization-scoped. When search is non-empty it LEFT JOINs the
// brand and material tables (matched on the SAME organization_id, so a term can
// never match a foreign org's brand/material) and compares names with an
// escaped ILIKE. order is trusted (whitelisted by the caller) and defaults to
// newest-first.
func (r *filamentRepositoryImpl) list(ctx context.Context, organizationID string, filters map[string]interface{}, search, order string, limit, offset int) ([]*entities.FilamentEntity, int64, error) {
	query := r.db.WithContext(ctx).
		Model(&models.FilamentModel{}).
		Where("filaments.organization_id = ?", organizationID)

	if search != "" {
		like := "%" + helpers.EscapeLike(search) + "%"
		query = query.
			Joins("LEFT JOIN brands ON brands.id = filaments.brand_id AND brands.organization_id = filaments.organization_id").
			Joins("LEFT JOIN materials ON materials.id = filaments.material_id AND materials.organization_id = filaments.organization_id").
			Where(
				`filaments.name ILIKE ? ESCAPE '\' OR brands.name ILIKE ? ESCAPE '\' OR materials.name ILIKE ? ESCAPE '\'`,
				like, like, like,
			)
	}

	query = applyFilamentFilters(query, filters)

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to count filaments: %w", err)
	}

	if order == "" {
		order = "filaments.created_at desc"
	}

	var filaments []models.FilamentModel
	if err := query.
		Order(order).
		Limit(limit).
		Offset(offset).
		Find(&filaments).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to find filaments: %w", err)
	}

	result := make([]*entities.FilamentEntity, len(filaments))
	for i := range filaments {
		result[i] = filaments[i].ToEntity()
	}

	return result, total, nil
}

// applyFilamentFilters adds the structured WHERE clauses for the supported
// filters. Columns are qualified with the filaments table so they stay
// unambiguous when the search joins are present.
func applyFilamentFilters(query *gorm.DB, filters map[string]interface{}) *gorm.DB {
	if filters == nil {
		return query
	}
	if brandID, ok := filters["brand_id"]; ok {
		query = query.Where("filaments.brand_id = ?", brandID)
	}
	if materialID, ok := filters["material_id"]; ok {
		query = query.Where("filaments.material_id = ?", materialID)
	}
	if colorType, ok := filters["color_type"]; ok {
		query = query.Where("filaments.color_type = ?", colorType)
	}
	if diameter, ok := filters["diameter"]; ok {
		query = query.Where("filaments.diameter = ?", diameter)
	}
	if minPrice, ok := filters["min_price"]; ok {
		query = query.Where("filaments.price_per_kg >= ?", minPrice)
	}
	if maxPrice, ok := filters["max_price"]; ok {
		query = query.Where("filaments.price_per_kg <= ?", maxPrice)
	}
	// low_stock=true keeps only tracked filaments at or below their (non-null)
	// alert threshold, mirroring FilamentEntity.IsLowStock.
	if lowStock, ok := filters["low_stock"]; ok {
		if v, isBool := lowStock.(bool); isBool && v {
			query = query.Where("filaments.track_stock = ? AND filaments.low_stock_threshold_grams IS NOT NULL AND filaments.stock_grams <= filaments.low_stock_threshold_grams", true)
		}
	}
	return query
}

// ExistsByNameAndBrand checks if a filament with the given name and brand already
// exists within the organization.
func (r *filamentRepositoryImpl) ExistsByNameAndBrand(ctx context.Context, name string, brandID uuid.UUID, organizationID string, excludeID *uuid.UUID) (bool, error) {
	var count int64

	query := r.db.WithContext(ctx).Model(&models.FilamentModel{}).
		Where("name = ? AND brand_id = ? AND organization_id = ?", name, brandID, organizationID)

	if excludeID != nil {
		query = query.Where("id != ?", *excludeID)
	}

	if err := query.Count(&count).Error; err != nil {
		return false, fmt.Errorf("failed to check filament existence: %w", err)
	}

	return count > 0, nil
}

// GetBrandInfo fetches brand information by ID, scoped to the organization so a
// filament can never resolve a brand that belongs to another tenant.
func (r *filamentRepositoryImpl) GetBrandInfo(ctx context.Context, brandID uuid.UUID, organizationID string) (*entities.BrandInfo, error) {
	var brand struct {
		ID          uuid.UUID `gorm:"column:id"`
		Name        string    `gorm:"column:name"`
		Description string    `gorm:"column:description"`
	}

	if err := r.db.WithContext(ctx).
		Table("brands").
		Select("id, name, description").
		Where("id = ? AND organization_id = ?", brandID, organizationID).
		First(&brand).Error; err != nil {
		return nil, err
	}

	return &entities.BrandInfo{
		ID:          brand.ID.String(),
		Name:        brand.Name,
		Description: brand.Description,
	}, nil
}

// GetMaterialInfo fetches material information by ID, scoped to the organization.
func (r *filamentRepositoryImpl) GetMaterialInfo(ctx context.Context, materialID uuid.UUID, organizationID string) (*entities.MaterialInfo, error) {
	var material struct {
		ID           uuid.UUID `gorm:"column:id"`
		Name         string    `gorm:"column:name"`
		Description  string    `gorm:"column:description"`
		TempTable    float32   `gorm:"column:temp_table"`
		TempExtruder float32   `gorm:"column:temp_extruder"`
	}

	if err := r.db.WithContext(ctx).
		Table("materials").
		Select("id, name, description, temp_table, temp_extruder").
		Where("id = ? AND organization_id = ?", materialID, organizationID).
		First(&material).Error; err != nil {
		return nil, err
	}

	return &entities.MaterialInfo{
		ID:           material.ID.String(),
		Name:         material.Name,
		Description:  material.Description,
		TempTable:    material.TempTable,
		TempExtruder: material.TempExtruder,
	}, nil
}

// GetBrandsInfo fetches multiple brands in a single query, scoped to the
// organization, returning a map keyed by brand ID. This is the batch form used
// by list endpoints to avoid issuing one query per row (N+1).
func (r *filamentRepositoryImpl) GetBrandsInfo(ctx context.Context, brandIDs []uuid.UUID, organizationID string) (map[uuid.UUID]*entities.BrandInfo, error) {
	result := make(map[uuid.UUID]*entities.BrandInfo)
	ids := dedupeIDs(brandIDs)
	if len(ids) == 0 {
		return result, nil
	}

	var rows []struct {
		ID          uuid.UUID `gorm:"column:id"`
		Name        string    `gorm:"column:name"`
		Description string    `gorm:"column:description"`
	}

	if err := r.db.WithContext(ctx).
		Table("brands").
		Select("id, name, description").
		Where("id IN ? AND organization_id = ?", ids, organizationID).
		Find(&rows).Error; err != nil {
		return nil, fmt.Errorf("failed to fetch brands: %w", err)
	}

	for i := range rows {
		result[rows[i].ID] = &entities.BrandInfo{
			ID:          rows[i].ID.String(),
			Name:        rows[i].Name,
			Description: rows[i].Description,
		}
	}

	return result, nil
}

// GetMaterialsInfo fetches multiple materials in a single query, scoped to the
// organization, returning a map keyed by material ID. Batch form used by list
// endpoints to avoid N+1.
func (r *filamentRepositoryImpl) GetMaterialsInfo(ctx context.Context, materialIDs []uuid.UUID, organizationID string) (map[uuid.UUID]*entities.MaterialInfo, error) {
	result := make(map[uuid.UUID]*entities.MaterialInfo)
	ids := dedupeIDs(materialIDs)
	if len(ids) == 0 {
		return result, nil
	}

	var rows []struct {
		ID           uuid.UUID `gorm:"column:id"`
		Name         string    `gorm:"column:name"`
		Description  string    `gorm:"column:description"`
		TempTable    float32   `gorm:"column:temp_table"`
		TempExtruder float32   `gorm:"column:temp_extruder"`
	}

	if err := r.db.WithContext(ctx).
		Table("materials").
		Select("id, name, description, temp_table, temp_extruder").
		Where("id IN ? AND organization_id = ?", ids, organizationID).
		Find(&rows).Error; err != nil {
		return nil, fmt.Errorf("failed to fetch materials: %w", err)
	}

	for i := range rows {
		result[rows[i].ID] = &entities.MaterialInfo{
			ID:           rows[i].ID.String(),
			Name:         rows[i].Name,
			Description:  rows[i].Description,
			TempTable:    rows[i].TempTable,
			TempExtruder: rows[i].TempExtruder,
		}
	}

	return result, nil
}

// dedupeIDs removes zero and duplicate UUIDs so the IN clause stays minimal.
func dedupeIDs(ids []uuid.UUID) []uuid.UUID {
	seen := make(map[uuid.UUID]struct{}, len(ids))
	out := make([]uuid.UUID, 0, len(ids))
	for _, id := range ids {
		if id == uuid.Nil {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	return out
}
