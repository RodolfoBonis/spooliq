package mappers

import (
	"github.com/RodolfoBonis/spooliq/features/quotes/data/models"
	"github.com/RodolfoBonis/spooliq/features/quotes/domain/entities"
	"github.com/RodolfoBonis/spooliq/features/quotes/presentation/dto"
)

// ModelToEntity converte QuoteModel para Quote entity
func ModelToEntity(model *models.QuoteModel) *entities.Quote {
	if model == nil {
		return nil
	}

	entity := &entities.Quote{
		ID:          model.ID,
		Title:       model.Title,
		Notes:       model.Notes,
		OwnerUserID: model.OwnerUserID,
		CreatedAt:   model.CreatedAt,
		UpdatedAt:   model.UpdatedAt,
	}

	// Convert filament lines
	if len(model.FilamentLines) > 0 {
		entity.FilamentLines = make([]entities.QuoteFilamentLine, 0, len(model.FilamentLines))
		for _, lineModel := range model.FilamentLines {
			entity.FilamentLines = append(entity.FilamentLines, entities.QuoteFilamentLine{
				ID:                            lineModel.ID,
				QuoteID:                       lineModel.QuoteID,
				FilamentSnapshotName:          lineModel.FilamentSnapshotName,
				FilamentSnapshotBrand:         lineModel.FilamentSnapshotBrand,
				FilamentSnapshotMaterial:      lineModel.FilamentSnapshotMaterial,
				FilamentSnapshotColor:         lineModel.FilamentSnapshotColor,
				FilamentSnapshotColorHex:      lineModel.FilamentSnapshotColorHex,
				FilamentSnapshotPricePerKg:    lineModel.FilamentSnapshotPricePerKg,
				FilamentSnapshotPricePerMeter: lineModel.FilamentSnapshotPricePerMeter,
				FilamentSnapshotURL:           lineModel.FilamentSnapshotURL,
				WeightGrams:                   lineModel.WeightGrams,
				LengthMeters:                  lineModel.LengthMeters,
				CreatedAt:                     lineModel.CreatedAt,
				UpdatedAt:                     lineModel.UpdatedAt,
			})
		}
	}

	// Convert profiles
	if model.MachineProfile != nil {
		entity.MachineProfile = &entities.MachineProfile{
			ID:          model.MachineProfile.ID,
			QuoteID:     model.MachineProfile.QuoteID,
			Name:        model.MachineProfile.Name,
			Brand:       model.MachineProfile.Brand,
			Model:       model.MachineProfile.Model,
			Watt:        model.MachineProfile.Watt,
			IdleFactor:  model.MachineProfile.IdleFactor,
			Description: model.MachineProfile.Description,
			URL:         model.MachineProfile.URL,
			CreatedAt:   model.MachineProfile.CreatedAt,
			UpdatedAt:   model.MachineProfile.UpdatedAt,
		}
	}

	if model.EnergyProfile != nil {
		entity.EnergyProfile = &entities.EnergyProfile{
			ID:            model.EnergyProfile.ID,
			QuoteID:       model.EnergyProfile.QuoteID,
			BaseTariff:    model.EnergyProfile.BaseTariff,
			FlagSurcharge: model.EnergyProfile.FlagSurcharge,
			Location:      model.EnergyProfile.Location,
			Year:          model.EnergyProfile.Year,
			Description:   model.EnergyProfile.Description,
			CreatedAt:     model.EnergyProfile.CreatedAt,
			UpdatedAt:     model.EnergyProfile.UpdatedAt,
		}
	}

	if model.CostProfile != nil {
		entity.CostProfile = &entities.CostProfile{
			ID:             model.CostProfile.ID,
			QuoteID:        model.CostProfile.QuoteID,
			WearPercentage: model.CostProfile.WearPercentage,
			OverheadAmount: model.CostProfile.OverheadAmount,
			Description:    model.CostProfile.Description,
			CreatedAt:      model.CostProfile.CreatedAt,
			UpdatedAt:      model.CostProfile.UpdatedAt,
		}
	}

	if model.MarginProfile != nil {
		entity.MarginProfile = &entities.MarginProfile{
			ID:                    model.MarginProfile.ID,
			QuoteID:               model.MarginProfile.QuoteID,
			PrintingOnlyMargin:    model.MarginProfile.PrintingOnlyMargin,
			PrintingPlusMargin:    model.MarginProfile.PrintingPlusMargin,
			FullServiceMargin:     model.MarginProfile.FullServiceMargin,
			OperatorRatePerHour:   model.MarginProfile.OperatorRatePerHour,
			ModelerRatePerHour:    model.MarginProfile.ModelerRatePerHour,
			Description:           model.MarginProfile.Description,
			CreatedAt:             model.MarginProfile.CreatedAt,
			UpdatedAt:             model.MarginProfile.UpdatedAt,
		}
	}

	return entity
}

// EntityToModel converte Quote entity para QuoteModel
func EntityToModel(entity *entities.Quote) *models.QuoteModel {
	if entity == nil {
		return nil
	}

	model := &models.QuoteModel{
		ID:          entity.ID,
		Title:       entity.Title,
		Notes:       entity.Notes,
		OwnerUserID: entity.OwnerUserID,
		CreatedAt:   entity.CreatedAt,
		UpdatedAt:   entity.UpdatedAt,
	}

	// Convert filament lines
	if len(entity.FilamentLines) > 0 {
		model.FilamentLines = make([]models.QuoteFilamentLineModel, 0, len(entity.FilamentLines))
		for _, lineEntity := range entity.FilamentLines {
			model.FilamentLines = append(model.FilamentLines, models.QuoteFilamentLineModel{
				ID:                            lineEntity.ID,
				QuoteID:                       lineEntity.QuoteID,
				FilamentSnapshotName:          lineEntity.FilamentSnapshotName,
				FilamentSnapshotBrand:         lineEntity.FilamentSnapshotBrand,
				FilamentSnapshotMaterial:      lineEntity.FilamentSnapshotMaterial,
				FilamentSnapshotColor:         lineEntity.FilamentSnapshotColor,
				FilamentSnapshotColorHex:      lineEntity.FilamentSnapshotColorHex,
				FilamentSnapshotPricePerKg:    lineEntity.FilamentSnapshotPricePerKg,
				FilamentSnapshotPricePerMeter: lineEntity.FilamentSnapshotPricePerMeter,
				FilamentSnapshotURL:           lineEntity.FilamentSnapshotURL,
				WeightGrams:                   lineEntity.WeightGrams,
				LengthMeters:                  lineEntity.LengthMeters,
				CreatedAt:                     lineEntity.CreatedAt,
				UpdatedAt:                     lineEntity.UpdatedAt,
			})
		}
	}

	// Convert profiles
	if entity.MachineProfile != nil {
		model.MachineProfile = &models.MachineProfileModel{
			ID:          entity.MachineProfile.ID,
			QuoteID:     entity.MachineProfile.QuoteID,
			Name:        entity.MachineProfile.Name,
			Brand:       entity.MachineProfile.Brand,
			Model:       entity.MachineProfile.Model,
			Watt:        entity.MachineProfile.Watt,
			IdleFactor:  entity.MachineProfile.IdleFactor,
			Description: entity.MachineProfile.Description,
			URL:         entity.MachineProfile.URL,
			CreatedAt:   entity.MachineProfile.CreatedAt,
			UpdatedAt:   entity.MachineProfile.UpdatedAt,
		}
	}

	if entity.EnergyProfile != nil {
		model.EnergyProfile = &models.EnergyProfileModel{
			ID:            entity.EnergyProfile.ID,
			QuoteID:       entity.EnergyProfile.QuoteID,
			BaseTariff:    entity.EnergyProfile.BaseTariff,
			FlagSurcharge: entity.EnergyProfile.FlagSurcharge,
			Location:      entity.EnergyProfile.Location,
			Year:          entity.EnergyProfile.Year,
			Description:   entity.EnergyProfile.Description,
			CreatedAt:     entity.EnergyProfile.CreatedAt,
			UpdatedAt:     entity.EnergyProfile.UpdatedAt,
		}
	}

	if entity.CostProfile != nil {
		model.CostProfile = &models.CostProfileModel{
			ID:             entity.CostProfile.ID,
			QuoteID:        entity.CostProfile.QuoteID,
			WearPercentage: entity.CostProfile.WearPercentage,
			OverheadAmount: entity.CostProfile.OverheadAmount,
			Description:    entity.CostProfile.Description,
			CreatedAt:      entity.CostProfile.CreatedAt,
			UpdatedAt:      entity.CostProfile.UpdatedAt,
		}
	}

	if entity.MarginProfile != nil {
		model.MarginProfile = &models.MarginProfileModel{
			ID:                    entity.MarginProfile.ID,
			QuoteID:               entity.MarginProfile.QuoteID,
			PrintingOnlyMargin:    entity.MarginProfile.PrintingOnlyMargin,
			PrintingPlusMargin:    entity.MarginProfile.PrintingPlusMargin,
			FullServiceMargin:     entity.MarginProfile.FullServiceMargin,
			OperatorRatePerHour:   entity.MarginProfile.OperatorRatePerHour,
			ModelerRatePerHour:    entity.MarginProfile.ModelerRatePerHour,
			Description:           entity.MarginProfile.Description,
			CreatedAt:             entity.MarginProfile.CreatedAt,
			UpdatedAt:             entity.MarginProfile.UpdatedAt,
		}
	}

	return model
}

// CreateRequestToEntity converte CreateQuoteRequest para Quote entity
// Note: This function now only handles the basic quote data.
// Filament line processing with snapshots should be handled in the service layer.
func CreateRequestToEntity(req *dto.CreateQuoteRequest, ownerUserID string) *entities.Quote {
	if req == nil {
		return nil
	}

	entity := &entities.Quote{
		Title:       req.Title,
		Notes:       req.Notes,
		OwnerUserID: ownerUserID,
	}

	// Note: FilamentLines are now processed separately in the service layer
	// to support both automatic snapshots and manual snapshot data

	// Convert profiles
	if req.MachineProfile != nil {
		entity.MachineProfile = &entities.MachineProfile{
			Name:        req.MachineProfile.Name,
			Brand:       req.MachineProfile.Brand,
			Model:       req.MachineProfile.Model,
			Watt:        req.MachineProfile.Watt,
			IdleFactor:  req.MachineProfile.IdleFactor,
			Description: req.MachineProfile.Description,
			URL:         req.MachineProfile.URL,
		}
	}

	if req.EnergyProfile != nil {
		entity.EnergyProfile = &entities.EnergyProfile{
			BaseTariff:    req.EnergyProfile.BaseTariff,
			FlagSurcharge: req.EnergyProfile.FlagSurcharge,
			Location:      req.EnergyProfile.Location,
			Year:          req.EnergyProfile.Year,
			Description:   req.EnergyProfile.Description,
		}
	}

	if req.CostProfile != nil {
		entity.CostProfile = &entities.CostProfile{
			WearPercentage: req.CostProfile.WearPercentage,
			OverheadAmount: req.CostProfile.OverheadAmount,
			Description:    req.CostProfile.Description,
		}
	}

	if req.MarginProfile != nil {
		entity.MarginProfile = &entities.MarginProfile{
			PrintingOnlyMargin:  req.MarginProfile.PrintingOnlyMargin,
			PrintingPlusMargin:  req.MarginProfile.PrintingPlusMargin,
			FullServiceMargin:   req.MarginProfile.FullServiceMargin,
			OperatorRatePerHour: req.MarginProfile.OperatorRatePerHour,
			ModelerRatePerHour:  req.MarginProfile.ModelerRatePerHour,
			Description:         req.MarginProfile.Description,
		}
	}

	return entity
}

// ModelsToEntities converte slice de QuoteModel para slice de Quote entities
func ModelsToEntities(models []*models.QuoteModel) []*entities.Quote {
	if models == nil {
		return nil
	}

	entities := make([]*entities.Quote, 0, len(models))
	for _, model := range models {
		entities = append(entities, ModelToEntity(model))
	}
	return entities
}