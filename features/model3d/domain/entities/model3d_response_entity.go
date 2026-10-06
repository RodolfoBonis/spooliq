package entities

// ListModel3DResponse is the paginated list envelope for 3D models. It mirrors
// helpers.Page[*Model3DEntity] and exists so Swagger can document the shape; the
// handler builds the real response with helpers.NewPage.
type ListModel3DResponse struct {
	Data       []*Model3DEntity `json:"data"`
	Total      int64            `json:"total"`
	Page       int              `json:"page"`
	PageSize   int              `json:"page_size"`
	TotalPages int              `json:"total_pages"`
}

// DuplicateModel3DResponse is the 409 body returned when an upload matches an
// existing model's file hash. It keeps the standard error envelope
// ({error, message, code}) and adds the existing model so the web app can link to
// it (it reads data.existing).
type DuplicateModel3DResponse struct {
	Error    string         `json:"error"`
	Message  string         `json:"message"`
	Code     string         `json:"code"`
	Existing *Model3DEntity `json:"existing"`
}
