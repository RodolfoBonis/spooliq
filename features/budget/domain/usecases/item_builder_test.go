package usecases

import (
	"testing"

	"github.com/RodolfoBonis/spooliq/features/budget/domain/entities"
	"github.com/google/uuid"
)

func strPtr(s string) *string { return &s }

func TestBuildBudgetItems(t *testing.T) {
	budgetID := uuid.New()
	const orgID = "org-123"
	costPresetID := uuid.New()
	fil1 := uuid.New()
	fil2 := uuid.New()

	reqs := []entities.BudgetItemRequest{
		{
			ProductName:             "Keychain batch",
			ProductDescription:      strPtr("100 pink keychains"),
			ProductQuantity:         100,
			ProductDimensions:       strPtr("40x20mm"),
			PrintTimeHours:          5,
			PrintTimeMinutes:        30,
			SetupTimeMinutes:        15,
			ManualLaborMinutesTotal: 60,
			CostPresetID:            &costPresetID,
			AdditionalNotes:         strPtr("fragile"),
			Order:                   1,
			Filaments: []entities.BudgetItemFilamentRequest{
				{FilamentID: fil1, Quantity: 2800, Order: 1},
				{FilamentID: fil2, Quantity: 200, Order: 2},
			},
		},
	}

	built := buildBudgetItems(budgetID, orgID, reqs)

	if len(built) != 1 {
		t.Fatalf("expected 1 built item, got %d", len(built))
	}

	b := built[0]
	item := b.Item

	// Organization + budget wiring.
	if item.OrganizationID != orgID {
		t.Errorf("item.OrganizationID = %q, want %q", item.OrganizationID, orgID)
	}
	if item.BudgetID != budgetID {
		t.Errorf("item.BudgetID = %v, want %v", item.BudgetID, budgetID)
	}
	if item.ID == uuid.Nil {
		t.Error("item.ID should be generated, got uuid.Nil")
	}

	// Legacy columns must be populated for backward compatibility.
	if item.FilamentID != fil1 {
		t.Errorf("item.FilamentID (legacy primary) = %v, want first filament %v", item.FilamentID, fil1)
	}
	if item.Quantity != 3000 {
		t.Errorf("item.Quantity (legacy total grams) = %v, want 3000", item.Quantity)
	}

	// All product fields copied.
	if item.ProductName != "Keychain batch" {
		t.Errorf("ProductName = %q", item.ProductName)
	}
	if item.ProductDescription == nil || *item.ProductDescription != "100 pink keychains" {
		t.Errorf("ProductDescription not copied: %v", item.ProductDescription)
	}
	if item.ProductQuantity != 100 {
		t.Errorf("ProductQuantity = %d, want 100", item.ProductQuantity)
	}
	if item.ProductDimensions == nil || *item.ProductDimensions != "40x20mm" {
		t.Errorf("ProductDimensions not copied: %v", item.ProductDimensions)
	}
	if item.PrintTimeHours != 5 || item.PrintTimeMinutes != 30 {
		t.Errorf("print time = %dh%dm, want 5h30m", item.PrintTimeHours, item.PrintTimeMinutes)
	}
	if item.SetupTimeMinutes != 15 {
		t.Errorf("SetupTimeMinutes = %d, want 15", item.SetupTimeMinutes)
	}
	if item.ManualLaborMinutesTotal != 60 {
		t.Errorf("ManualLaborMinutesTotal = %d, want 60", item.ManualLaborMinutesTotal)
	}
	if item.CostPresetID == nil || *item.CostPresetID != costPresetID {
		t.Errorf("CostPresetID not copied: %v", item.CostPresetID)
	}
	if item.AdditionalNotes == nil || *item.AdditionalNotes != "fragile" {
		t.Errorf("AdditionalNotes not copied: %v", item.AdditionalNotes)
	}
	if item.Order != 1 {
		t.Errorf("Order = %d, want 1", item.Order)
	}

	// Filaments: org set, linked to the item, fields copied.
	if len(b.Filaments) != 2 {
		t.Fatalf("expected 2 filaments, got %d", len(b.Filaments))
	}
	for _, f := range b.Filaments {
		if f.OrganizationID != orgID {
			t.Errorf("filament.OrganizationID = %q, want %q", f.OrganizationID, orgID)
		}
		if f.BudgetItemID != item.ID {
			t.Errorf("filament.BudgetItemID = %v, want %v", f.BudgetItemID, item.ID)
		}
		if f.ID == uuid.Nil {
			t.Error("filament.ID should be generated, got uuid.Nil")
		}
	}
	if b.Filaments[0].FilamentID != fil1 || b.Filaments[0].Quantity != 2800 || b.Filaments[0].Order != 1 {
		t.Errorf("first filament not copied correctly: %+v", b.Filaments[0])
	}
	if b.Filaments[1].FilamentID != fil2 || b.Filaments[1].Quantity != 200 || b.Filaments[1].Order != 2 {
		t.Errorf("second filament not copied correctly: %+v", b.Filaments[1])
	}
}

func TestBuildBudgetItems_MultipleItemsHaveUniqueIDs(t *testing.T) {
	budgetID := uuid.New()
	const orgID = "org-xyz"
	fil := uuid.New()

	reqs := []entities.BudgetItemRequest{
		{ProductName: "A", ProductQuantity: 1, Filaments: []entities.BudgetItemFilamentRequest{{FilamentID: fil, Quantity: 10, Order: 1}}},
		{ProductName: "B", ProductQuantity: 1, Filaments: []entities.BudgetItemFilamentRequest{{FilamentID: fil, Quantity: 20, Order: 1}}},
	}

	built := buildBudgetItems(budgetID, orgID, reqs)
	if len(built) != 2 {
		t.Fatalf("expected 2 items, got %d", len(built))
	}
	if built[0].Item.ID == built[1].Item.ID {
		t.Error("expected distinct item IDs")
	}
	// Each filament must point to its own item.
	if built[0].Filaments[0].BudgetItemID != built[0].Item.ID {
		t.Error("item 0 filament not linked to item 0")
	}
	if built[1].Filaments[0].BudgetItemID != built[1].Item.ID {
		t.Error("item 1 filament not linked to item 1")
	}
}

func TestCollectFilamentIDs(t *testing.T) {
	fil1 := uuid.New()
	fil2 := uuid.New()
	fil3 := uuid.New()

	reqs := []entities.BudgetItemRequest{
		{Filaments: []entities.BudgetItemFilamentRequest{{FilamentID: fil1}, {FilamentID: fil2}}},
		{Filaments: []entities.BudgetItemFilamentRequest{{FilamentID: fil3}}},
	}

	got := collectFilamentIDs(reqs)
	if len(got) != 3 {
		t.Fatalf("expected 3 filament IDs, got %d", len(got))
	}

	want := map[uuid.UUID]bool{fil1: true, fil2: true, fil3: true}
	for _, id := range got {
		if !want[id] {
			t.Errorf("unexpected filament ID %v", id)
		}
	}
}
