package entities

// FindAllBrandsResponse represents the response for getting all brands.
type FindAllBrandsResponse struct {
	Data []BrandEntity `json:"data"`
}
