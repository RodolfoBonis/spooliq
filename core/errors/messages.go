package errors

// ErrorMessages contém todas as mensagens de erro padronizadas em português
var ErrorMessages = struct {
	// Validação
	InvalidRequestFormat string
	ValidationFailed     string
	InvalidID            string

	// Autenticação
	UserNotAuthenticated string
	AccessDenied         string

	// Filamentos
	FilamentNotFound           string
	FailedToCreateFilament     string
	FailedToUpdateFilament     string
	FailedToDeleteFilament     string
	FailedToGetFilaments       string
	FailedToGetUserFilaments   string
	FailedToGetGlobalFilaments string

	// Orçamentos
	QuoteNotFound          string
	FailedToCreateQuote    string
	FailedToUpdateQuote    string
	FailedToDeleteQuote    string
	FailedToGetQuotes      string
	FailedToDuplicateQuote string
	FailedToCalculateQuote string
}{
	// Validação
	InvalidRequestFormat: "Formato de requisição inválido",
	ValidationFailed:     "Falha na validação dos dados",
	InvalidID:            "ID inválido",

	// Autenticação
	UserNotAuthenticated: "Usuário não autenticado",
	AccessDenied:         "Acesso negado",

	// Filamentos
	FilamentNotFound:           "Filamento não encontrado",
	FailedToCreateFilament:     "Falha ao criar filamento",
	FailedToUpdateFilament:     "Falha ao atualizar filamento",
	FailedToDeleteFilament:     "Falha ao excluir filamento",
	FailedToGetFilaments:       "Falha ao buscar filamentos",
	FailedToGetUserFilaments:   "Falha ao buscar filamentos do usuário",
	FailedToGetGlobalFilaments: "Falha ao buscar filamentos globais",

	// Orçamentos
	QuoteNotFound:          "Orçamento não encontrado",
	FailedToCreateQuote:    "Falha ao criar orçamento",
	FailedToUpdateQuote:    "Falha ao atualizar orçamento",
	FailedToDeleteQuote:    "Falha ao excluir orçamento",
	FailedToGetQuotes:      "Falha ao buscar orçamentos",
	FailedToDuplicateQuote: "Falha ao duplicar orçamento",
	FailedToCalculateQuote: "Falha ao calcular orçamento",
}

// ErrorResponse creates a standardized error response map
func ErrorResponse(message string, details ...string) map[string]interface{} {
	response := map[string]interface{}{
		"error": message,
	}

	if len(details) > 0 && details[0] != "" {
		response["details"] = details[0]
	}

	return response
}

// ValidationErrorResponse creates a validation error response
func ValidationErrorResponse(details string) map[string]interface{} {
	return ErrorResponse(ErrorMessages.ValidationFailed, details)
}

// InvalidRequestResponse creates an invalid request error response
func InvalidRequestResponse(details string) map[string]interface{} {
	return ErrorResponse(ErrorMessages.InvalidRequestFormat, details)
}
