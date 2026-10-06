package entities

// FindAllBrandsResponse is the paginated list envelope for brands. It mirrors
// helpers.Page[BrandEntity] (the value actually returned by the handler) and
// exists as a concrete, non-generic type so Swagger can document the response.
type FindAllBrandsResponse struct {
	Data       []BrandEntity `json:"data"`
	Total      int64         `json:"total"`
	Page       int           `json:"page"`
	PageSize   int           `json:"page_size"`
	TotalPages int           `json:"total_pages"`
}
