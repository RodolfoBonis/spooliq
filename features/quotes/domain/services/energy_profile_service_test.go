package services

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	presetsEntities "github.com/RodolfoBonis/spooliq/features/presets/domain/entities"
	"github.com/RodolfoBonis/spooliq/features/quotes/presentation/dto"
)

// MockPresetRepository implements PresetRepository for testing
type MockPresetRepository struct {
	presets map[string]*presetsEntities.Preset
}

func NewMockPresetRepository() *MockPresetRepository {
	// Create test energy preset data
	energyData := presetsEntities.EnergyPreset{
		Key:           "energy_sp_2024",
		BaseTariff:    0.65,
		FlagSurcharge: 0.10,
		Location:      "São Paulo",
		State:         "SP",
		City:          "São Paulo",
		Year:          2024,
		Description:   "Tarifa SP 2024",
	}
	
	// Marshal to JSON
	jsonData, _ := json.Marshal(energyData)
	
	return &MockPresetRepository{
		presets: map[string]*presetsEntities.Preset{
			"energy_sp_2024": {
				ID:   1,
				Key:  "energy_sp_2024",
				Data: string(jsonData),
			},
		},
	}
}

func (m *MockPresetRepository) GetPresetByKey(ctx context.Context, key string) (*presetsEntities.Preset, error) {
	preset, exists := m.presets[key]
	if !exists {
		return nil, fmt.Errorf("preset not found: %s", key)
	}
	return preset, nil
}

func TestEnergyProfileService_CreateFromPreset(t *testing.T) {
	// Arrange
	mockRepo := NewMockPresetRepository()
	service := NewEnergyProfileService(mockRepo)
	
	ctx := context.Background()
	userID := "test-user-123"
	
	req := &dto.CreateEnergyProfileRequest{
		PresetKey: "energy_sp_2024",
	}
	
	// Act
	result, err := service.CreateEnergyProfileFromRequest(ctx, req, userID)
	
	// Assert
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
	
	if result == nil {
		t.Fatal("Expected result, got nil")
	}
	
	// Verify data from preset
	if result.Name != "São Paulo 2024" {
		t.Errorf("Expected name 'São Paulo 2024', got: %s", result.Name)
	}
	
	if result.BaseTariff != 0.65 {
		t.Errorf("Expected base tariff 0.65, got: %f", result.BaseTariff)
	}
	
	if result.FlagSurcharge != 0.10 {
		t.Errorf("Expected flag surcharge 0.10, got: %f", result.FlagSurcharge)
	}
	
	if result.Location != "São Paulo" {
		t.Errorf("Expected location 'São Paulo', got: %s", result.Location)
	}
	
	if result.Year != 2024 {
		t.Errorf("Expected year 2024, got: %d", result.Year)
	}
}

func TestEnergyProfileService_CreateFromCustomData(t *testing.T) {
	// Arrange
	mockRepo := NewMockPresetRepository()
	service := NewEnergyProfileService(mockRepo)
	
	ctx := context.Background()
	userID := "test-user-123"
	
	req := &dto.CreateEnergyProfileRequest{
		Name:          "Custom Rio 2025",
		BaseTariff:    0.75,
		FlagSurcharge: 0.15,
		Location:      "Rio de Janeiro",
		Year:          2025,
		Description:   "Custom tariff",
	}
	
	// Act
	result, err := service.CreateEnergyProfileFromRequest(ctx, req, userID)
	
	// Assert
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
	
	if result == nil {
		t.Fatal("Expected result, got nil")
	}
	
	// Verify custom data
	if result.Name != "Custom Rio 2025" {
		t.Errorf("Expected name 'Custom Rio 2025', got: %s", result.Name)
	}
	
	if result.BaseTariff != 0.75 {
		t.Errorf("Expected base tariff 0.75, got: %f", result.BaseTariff)
	}
	
	if result.Location != "Rio de Janeiro" {
		t.Errorf("Expected location 'Rio de Janeiro', got: %s", result.Location)
	}
	
	if result.Year != 2025 {
		t.Errorf("Expected year 2025, got: %d", result.Year)
	}
}

func TestEnergyProfileService_AutoGenerateName(t *testing.T) {
	// Arrange
	mockRepo := NewMockPresetRepository()
	service := NewEnergyProfileService(mockRepo)
	
	ctx := context.Background()
	userID := "test-user-123"
	
	req := &dto.CreateEnergyProfileRequest{
		// No name provided - should auto-generate
		BaseTariff:    0.80,
		FlagSurcharge: 0.20,
		Location:      "Brasília",
		Year:          2024,
	}
	
	// Act
	result, err := service.CreateEnergyProfileFromRequest(ctx, req, userID)
	
	// Assert
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
	
	if result == nil {
		t.Fatal("Expected result, got nil")
	}
	
	// Verify auto-generated name
	if result.Name != "Brasília 2024" {
		t.Errorf("Expected auto-generated name 'Brasília 2024', got: %s", result.Name)
	}
}