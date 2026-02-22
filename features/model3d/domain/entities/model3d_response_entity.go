package entities

// FindAllModel3DResponse represents a paginated list of 3D models.
type FindAllModel3DResponse struct {
	Data       []*Model3DEntity `json:"data"`
	Total      int64            `json:"total"`
	Page       int              `json:"page"`
	PageSize   int              `json:"page_size"`
	TotalPages int              `json:"total_pages"`
}
