package entities

// TopMaterial represents a top material by usage.
type TopMaterial struct {
	ID         string  `json:"id"`
	Name       string  `json:"name"`
	TotalGrams float64 `json:"total_grams"`
	UsageCount int     `json:"usage_count"`
}

// TopMaterialsResponse contains top materials data.
type TopMaterialsResponse struct {
	Materials []TopMaterial `json:"materials"`
	Period    string        `json:"period"`
}
