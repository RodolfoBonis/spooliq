package entities

// TopFilament represents a top filament by usage.
type TopFilament struct {
	ID           string  `json:"id"`
	Name         string  `json:"name"`
	BrandName    string  `json:"brand_name"`
	MaterialName string  `json:"material_name"`
	ColorHex     string  `json:"color_hex,omitempty"`
	TotalGrams   float64 `json:"total_grams"`
	UsageCount   int     `json:"usage_count"`
}

// TopFilamentsResponse contains top filaments data.
type TopFilamentsResponse struct {
	Filaments []TopFilament `json:"filaments"`
	Period    string        `json:"period"`
}
