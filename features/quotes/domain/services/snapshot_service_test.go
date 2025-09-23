package services

import (
	"context"
	"errors"
	"testing"

	metadataEntities "github.com/RodolfoBonis/spooliq/features/filament-metadata/domain/entities"
	filamentEntities "github.com/RodolfoBonis/spooliq/features/filaments/domain/entities"
	"github.com/RodolfoBonis/spooliq/features/quotes/presentation/dto"
)

// MockFilamentRepository implements FilamentRepository for testing
type MockFilamentRepository struct {
	filaments map[uint]*filamentEntities.Filament
}

func NewMockFilamentRepository() *MockFilamentRepository {
	// Create some test filament data
	pricePerMeter := 0.083
	return &MockFilamentRepository{
		filaments: map[uint]*filamentEntities.Filament{
			1: {
				ID:   1,
				Name: "PLA Premium Test",
				Brand: metadataEntities.FilamentBrand{
					ID:   1,
					Name: "TestBrand",
				},
				Material: metadataEntities.FilamentMaterial{
					ID:   1,
					Name: "PLA",
				},
				Color:         "Orange",
				ColorHex:      "#FF6600",
				PricePerKg:    24.99,
				PricePerMeter: &pricePerMeter,
				URL:           "https://test.com/filament",
				OwnerUserID:   nil, // Global filament
			},
		},
	}
}

func (m *MockFilamentRepository) GetByID(ctx context.Context, id uint, userID *string) (*filamentEntities.Filament, error) {
	filament, exists := m.filaments[id]
	if !exists {
		return nil, errors.New("filament not found")
	}
	return filament, nil
}

func TestSnapshotService_CreateFilamentSnapshot_AutomaticSnapshot(t *testing.T) {
	// Arrange
	mockRepo := NewMockFilamentRepository()
	service := NewSnapshotService(mockRepo)

	ctx := context.Background()
	userID := "test-user-123"
	filamentID := uint(1)

	req := &dto.CreateFilamentLineRequest{
		FilamentID:   &filamentID,
		WeightGrams:  125.5,
		LengthMeters: &[]float64{41.8}[0],
	}

	// Act
	result, err := service.CreateFilamentSnapshot(ctx, req, userID)

	// Assert
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if result == nil {
		t.Fatal("Expected result, got nil")
	}

	// Verify snapshot data was copied from filament
	if result.FilamentSnapshotName != "PLA Premium Test" {
		t.Errorf("Expected name 'PLA Premium Test', got: %s", result.FilamentSnapshotName)
	}

	if result.FilamentSnapshotBrand != "TestBrand" {
		t.Errorf("Expected brand 'TestBrand', got: %s", result.FilamentSnapshotBrand)
	}

	if result.FilamentSnapshotMaterial != "PLA" {
		t.Errorf("Expected material 'PLA', got: %s", result.FilamentSnapshotMaterial)
	}

	if result.FilamentSnapshotColor != "Orange" {
		t.Errorf("Expected color 'Orange', got: %s", result.FilamentSnapshotColor)
	}

	if result.FilamentSnapshotPricePerKg != 24.99 {
		t.Errorf("Expected price per kg 24.99, got: %f", result.FilamentSnapshotPricePerKg)
	}

	if result.WeightGrams != 125.5 {
		t.Errorf("Expected weight 125.5, got: %f", result.WeightGrams)
	}

	// CRITICAL: Verify FilamentID is set correctly
	if result.FilamentID != 1 {
		t.Errorf("Expected FilamentID 1, got: %d", result.FilamentID)
	}
}

func TestSnapshotService_CreateFilamentSnapshot_ManualSnapshot(t *testing.T) {
	// Arrange
	mockRepo := NewMockFilamentRepository()
	service := NewSnapshotService(mockRepo)

	ctx := context.Background()
	userID := "test-user-123"

	req := &dto.CreateFilamentLineRequest{
		FilamentID:                    nil, // No filament ID - manual snapshot
		FilamentSnapshotName:          "Manual PLA",
		FilamentSnapshotBrand:         "ManualBrand",
		FilamentSnapshotMaterial:      "PLA",
		FilamentSnapshotColor:         "Red",
		FilamentSnapshotColorHex:      "#FF0000",
		FilamentSnapshotPricePerKg:    29.99,
		FilamentSnapshotPricePerMeter: &[]float64{0.095}[0],
		FilamentSnapshotURL:           "https://manual.com/filament",
		WeightGrams:                   75.2,
		LengthMeters:                  &[]float64{25.1}[0],
	}

	// Act
	result, err := service.CreateFilamentSnapshot(ctx, req, userID)

	// Assert
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if result == nil {
		t.Fatal("Expected result, got nil")
	}

	// Verify manual snapshot data was used
	if result.FilamentSnapshotName != "Manual PLA" {
		t.Errorf("Expected name 'Manual PLA', got: %s", result.FilamentSnapshotName)
	}

	if result.FilamentSnapshotBrand != "ManualBrand" {
		t.Errorf("Expected brand 'ManualBrand', got: %s", result.FilamentSnapshotBrand)
	}

	if result.FilamentSnapshotPricePerKg != 29.99 {
		t.Errorf("Expected price per kg 29.99, got: %f", result.FilamentSnapshotPricePerKg)
	}

	if result.WeightGrams != 75.2 {
		t.Errorf("Expected weight 75.2, got: %f", result.WeightGrams)
	}

	// CRITICAL: Verify FilamentID is 0 for manual snapshot
	if result.FilamentID != 0 {
		t.Errorf("Expected FilamentID 0 for manual snapshot, got: %d", result.FilamentID)
	}
}
