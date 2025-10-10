package entities

// FilamentResponse represents a filament response with related data
type FilamentResponse struct {
	*FilamentEntity
	Brand    *BrandInfo    `json:"brand,omitempty"`
	Material *MaterialInfo `json:"material,omitempty"`
}

// BrandInfo represents basic brand information for responses
type BrandInfo struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
}

// MaterialInfo represents basic material information for responses
type MaterialInfo struct {
	ID           string  `json:"id"`
	Name         string  `json:"name"`
	Description  string  `json:"description,omitempty"`
	TempTable    float32 `json:"temp_table,omitempty"`
	TempExtruder float32 `json:"temp_extruder,omitempty"`
}

// FindAllFilamentsResponse represents the response for listing filaments with relations
type FindAllFilamentsResponse struct {
	Data       []FilamentResponse `json:"data"`
	Total      int                `json:"total"`
	Page       int                `json:"page"`
	Limit      int                `json:"limit"`
	TotalPages int                `json:"total_pages"`
}

// FindByIDFilamentResponse represents the response for getting a single filament
type FindByIDFilamentResponse struct {
	Data *FilamentResponse `json:"data"`
}

// ColorTypesResponse represents the response for available color types
type ColorTypesResponse struct {
	Data []ColorTypeInfo `json:"data"`
}

// ColorTypeInfo represents information about a color type
type ColorTypeInfo struct {
	Type        ColorType `json:"type"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Schema      string    `json:"schema"` // JSON schema for the color data
}

// FilamentStatsResponse represents statistics about filaments
type FilamentStatsResponse struct {
	Total       int64            `json:"total"`
	ByBrand     map[string]int64 `json:"by_brand"`
	ByMaterial  map[string]int64 `json:"by_material"`
	ByColorType map[string]int64 `json:"by_color_type"`
	PriceRanges PriceRangeStats  `json:"price_ranges"`
}

// PriceRangeStats represents price statistics
type PriceRangeStats struct {
	Min     float64 `json:"min"`
	Max     float64 `json:"max"`
	Average float64 `json:"average"`
	Median  float64 `json:"median"`
}

// FilamentCompatibilityResponse represents compatibility information
type FilamentCompatibilityResponse struct {
	FilamentID          string   `json:"filament_id"`
	CompatibleMachines  []string `json:"compatible_machines"`
	RecommendedSettings struct {
		PrintTemperature int `json:"print_temperature"`
		BedTemperature   int `json:"bed_temperature"`
		PrintSpeed       int `json:"print_speed"`
	} `json:"recommended_settings"`
}

// BulkOperationResponse represents the response for bulk operations
type BulkOperationResponse struct {
	Success      int      `json:"success"`
	Failed       int      `json:"failed"`
	Errors       []string `json:"errors,omitempty"`
	ProcessedIDs []string `json:"processed_ids,omitempty"`
}
