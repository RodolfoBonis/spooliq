package entities

// FindByIDBrandResponse represents the response for getting a brand by ID.
type FindByIDBrandResponse struct {
	Data BrandEntity `json:"data"`
}
