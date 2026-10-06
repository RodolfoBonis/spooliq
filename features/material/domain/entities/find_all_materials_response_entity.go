package entities

// FindAllMaterialsResponse is the paginated list envelope for materials. It
// mirrors helpers.Page[MaterialEntity] (the value actually returned by the
// handler) and exists as a concrete, non-generic type so Swagger can document
// the response.
type FindAllMaterialsResponse struct {
	Data       []MaterialEntity `json:"data"`
	Total      int64            `json:"total"`
	Page       int              `json:"page"`
	PageSize   int              `json:"page_size"`
	TotalPages int              `json:"total_pages"`
}
