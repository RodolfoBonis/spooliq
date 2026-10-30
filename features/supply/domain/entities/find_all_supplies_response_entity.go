package entities

// FindAllSuppliesResponse is the paginated list envelope for supplies. It mirrors
// helpers.Page[SupplyEntity] (the value actually returned by the handler) and
// exists as a concrete, non-generic type so Swagger can document the response.
type FindAllSuppliesResponse struct {
	Data       []SupplyEntity `json:"data"`
	Total      int64          `json:"total"`
	Page       int            `json:"page"`
	PageSize   int            `json:"page_size"`
	TotalPages int            `json:"total_pages"`
}
