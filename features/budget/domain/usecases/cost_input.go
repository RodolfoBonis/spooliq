package usecases

import (
	coreErrors "github.com/RodolfoBonis/spooliq/core/errors"
	"github.com/RodolfoBonis/spooliq/features/budget/domain/entities"
)

// resolveIncludeMachineCost applies the "default TRUE for new budgets" rule: a nil
// pointer (field omitted from the request) means machine cost is charged.
func resolveIncludeMachineCost(v *bool) bool {
	if v == nil {
		return true
	}
	return *v
}

// derefString returns the pointed-to string, or "" when nil.
func derefString(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

// derefFloat returns the pointed-to float64, or 0 when nil.
func derefFloat(f *float64) float64 {
	if f == nil {
		return 0
	}
	return *f
}

// validateDiscountInput enforces the discount rules the declarative `validate` tags
// cannot express on their own: discount_type and discount_value must be provided
// together, a "percent" value must be within 0-100, and a "fixed" value must be
// non-negative. It returns a pt-BR BadRequest (via core/errors) on failure, or nil
// when the discount is absent or valid. The field range for "fixed" (>= 0) and the
// allowed type set are already enforced by the request tags; this adds the
// cross-field and percent-cap checks.
func validateDiscountInput(discountType *string, discountValue *float64) error {
	if discountType == nil && discountValue == nil {
		return nil
	}
	if discountType == nil || discountValue == nil {
		return coreErrors.BadRequest(coreErrors.CodeValidationError, "discount_type e discount_value devem ser informados juntos")
	}
	switch *discountType {
	case entities.DiscountTypePercent:
		if *discountValue < 0 || *discountValue > 100 {
			return coreErrors.BadRequest(coreErrors.CodeValidationError, "O desconto percentual deve estar entre 0 e 100")
		}
	case entities.DiscountTypeFixed:
		if *discountValue < 0 {
			return coreErrors.BadRequest(coreErrors.CodeValidationError, "O desconto fixo não pode ser negativo")
		}
	default:
		return coreErrors.BadRequest(coreErrors.CodeValidationError, "Tipo de desconto inválido")
	}
	return nil
}
