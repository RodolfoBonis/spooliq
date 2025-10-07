package entities

// FindAllMaterialsResponse represents the response for getting all materials.
type FindAllMaterialsResponse struct {
	Data []MaterialEntity `json:"data"`
}
