package usecases

import (
	"testing"

	"github.com/RodolfoBonis/spooliq/features/budget/domain/entities"
)

func TestPerPrintForRequiresUnitsPerPrint(t *testing.T) {
	per := &entities.PerPrintValues{PrintMinutes: 205}
	if got := perPrintFor(entities.BudgetItemRequest{PerPrint: per}); got != nil {
		t.Errorf("per_print without units_per_print must be dropped, got %+v", got)
	}
	units := 10
	if got := perPrintFor(entities.BudgetItemRequest{UnitsPerPrint: &units, PerPrint: per}); got != per {
		t.Errorf("per_print must be kept with units_per_print")
	}
}
