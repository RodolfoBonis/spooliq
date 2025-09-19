package services

import (
	"context"

	"github.com/RodolfoBonis/spooliq/features/export/domain/entities"
)

// ExportService define as operações de geração de exports
type ExportService interface {
	// ExportQuote gera um export da quote no formato especificado
	ExportQuote(ctx context.Context, request *entities.ExportRequest, userID string) (*entities.ExportResult, error)

	// GetSupportedFormats retorna os formatos suportados
	GetSupportedFormats() []entities.ExportFormat

	// ValidateRequest valida uma solicitação de export
	ValidateRequest(request *entities.ExportRequest) error
}