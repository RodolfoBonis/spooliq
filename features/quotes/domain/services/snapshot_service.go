package services

import (
	"context"
	"fmt"

	filamentEntities "github.com/RodolfoBonis/spooliq/features/filaments/domain/entities"
	quotesEntities "github.com/RodolfoBonis/spooliq/features/quotes/domain/entities"
	"github.com/RodolfoBonis/spooliq/features/quotes/presentation/dto"
)

// SnapshotService handles creation of price snapshots from current filament data
type SnapshotService interface {
	// CreateFilamentSnapshot creates a QuoteFilamentLine from a filament ID and request data
	CreateFilamentSnapshot(ctx context.Context, req *dto.CreateFilamentLineRequest, userID string) (*quotesEntities.QuoteFilamentLine, error)

	// ValidateFilamentAccess validates if a user can access a filament for snapshotting
	ValidateFilamentAccess(ctx context.Context, filamentID uint, userID string, isAdmin bool) error
}

type snapshotServiceImpl struct {
	filamentRepo FilamentRepository
}

// FilamentRepository defines the interface for accessing filament data
type FilamentRepository interface {
	GetByID(ctx context.Context, id uint, userID *string) (*filamentEntities.Filament, error)
}

// NewSnapshotService creates a new snapshot service
func NewSnapshotService(filamentRepo FilamentRepository) SnapshotService {
	return &snapshotServiceImpl{
		filamentRepo: filamentRepo,
	}
}

// CreateFilamentSnapshot creates a QuoteFilamentLine from either filament ID or manual data
func (s *snapshotServiceImpl) CreateFilamentSnapshot(ctx context.Context, req *dto.CreateFilamentLineRequest, userID string) (*quotesEntities.QuoteFilamentLine, error) {
	// Validate the request first
	if err := req.Validate(); err != nil {
		return nil, fmt.Errorf("invalid request: %w", err)
	}

	var line quotesEntities.QuoteFilamentLine

	// Set the always-required fields
	line.WeightGrams = req.WeightGrams
	line.LengthMeters = req.LengthMeters

	// Handle automatic snapshot from filament ID
	if req.FilamentID != nil {
		// Pass userID to repository for access control
		filament, err := s.filamentRepo.GetByID(ctx, *req.FilamentID, &userID)
		if err != nil {
			return nil, fmt.Errorf("failed to get filament %d: %w", *req.FilamentID, err)
		}

		// Validate user access to the filament
		// Note: We'll assume isAdmin=false for now, this should be passed from the calling service
		if !filament.CanUserAccess(userID, false) {
			return nil, fmt.Errorf("user %s does not have access to filament %d", userID, *req.FilamentID)
		}

		// Create snapshot from current filament data
		line.FilamentSnapshotName = filament.Name
		line.FilamentSnapshotBrand = filament.Brand.Name
		line.FilamentSnapshotMaterial = filament.Material.Name
		line.FilamentSnapshotColor = filament.Color
		line.FilamentSnapshotColorHex = filament.ColorHex
		line.FilamentSnapshotPricePerKg = filament.PricePerKg
		line.FilamentSnapshotPricePerMeter = filament.PricePerMeter
		line.FilamentSnapshotURL = filament.URL
	} else {
		// Use manual snapshot data
		line.FilamentSnapshotName = req.FilamentSnapshotName
		line.FilamentSnapshotBrand = req.FilamentSnapshotBrand
		line.FilamentSnapshotMaterial = req.FilamentSnapshotMaterial
		line.FilamentSnapshotColor = req.FilamentSnapshotColor
		line.FilamentSnapshotColorHex = req.FilamentSnapshotColorHex
		line.FilamentSnapshotPricePerKg = req.FilamentSnapshotPricePerKg
		line.FilamentSnapshotPricePerMeter = req.FilamentSnapshotPricePerMeter
		line.FilamentSnapshotURL = req.FilamentSnapshotURL
	}

	return &line, nil
}

// ValidateFilamentAccess validates if a user can access a filament for snapshotting
func (s *snapshotServiceImpl) ValidateFilamentAccess(ctx context.Context, filamentID uint, userID string, isAdmin bool) error {
	filament, err := s.filamentRepo.GetByID(ctx, filamentID, &userID)
	if err != nil {
		return fmt.Errorf("filament not found: %w", err)
	}

	if !filament.CanUserAccess(userID, isAdmin) {
		return fmt.Errorf("access denied to filament %d", filamentID)
	}

	return nil
}
