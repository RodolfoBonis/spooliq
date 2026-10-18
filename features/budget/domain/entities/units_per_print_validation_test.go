package entities_test

import (
	"testing"

	"github.com/RodolfoBonis/spooliq/core/validation"
	"github.com/RodolfoBonis/spooliq/features/budget/domain/entities"
	"github.com/google/uuid"
)

func TestBudgetItemRequest_UnitsPerPrintBounds(t *testing.T) {
	item := func(units *int) entities.BudgetItemRequest {
		return entities.BudgetItemRequest{
			ProductName:     "Potinho",
			ProductQuantity: 10,
			UnitsPerPrint:   units,
			Filaments: []entities.BudgetItemFilamentRequest{
				{FilamentID: uuid.New(), Quantity: 78.7, Order: 1},
			},
		}
	}
	ptr := func(v int) *int { return &v }

	for name, tc := range map[string]struct {
		units *int
		ok    bool
	}{
		"absent (total mode)": {nil, true},
		"ten per print":       {ptr(10), true},
		"zero":                {ptr(0), false},
		"negative":            {ptr(-3), false},
		"above the limit":     {ptr(1_000_001), false},
	} {
		t.Run(name, func(t *testing.T) {
			err := validation.Validate(item(tc.units))
			if tc.ok && err != nil {
				t.Errorf("expected valid, got %v", err)
			}
			if !tc.ok && err == nil {
				t.Error("expected a validation error")
			}
		})
	}
}
