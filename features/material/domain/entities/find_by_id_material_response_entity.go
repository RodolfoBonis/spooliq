package entities

// FindByIDMaterialResponse represents the response for getting a material by ID.
type FindByIDMaterialResponse struct {
	Data MaterialEntity `json:"data"`
}
