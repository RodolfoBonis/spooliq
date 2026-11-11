package entities

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"
)

// capture_method values InfinitePay reports. FeeFor only recognises these two; any
// other value (including an empty one) is treated as UNKNOWN so the fee snapshot stays
// NULL rather than being guessed as a card charge.
const (
	CaptureMethodPix        = "pix"
	CaptureMethodCreditCard = "credit_card"
)

// FeeRates is an organization's InfinitePay fee table. Each value is a percentage of
// the charged amount, with up to two decimal places, between 0 and MaxFeeRatePercent.
//
//   - Pix is the Pix fee.
//   - Credit1x is the single-installment ("crédito à vista") card fee.
//   - CreditInstallments maps an installment count ("2".."12") to its card fee.
//
// A nil *FeeRates means "not configured": no fee can be estimated and the price-embed
// switch cannot be turned on. Within a configured table, Pix and Credit1x are always
// known (0 is a valid rate); an installment with no entry in CreditInstallments is
// UNKNOWN, so its estimate is nil.
type FeeRates struct {
	Pix                float64            `json:"pix"`
	Credit1x           float64            `json:"credit_1x"`
	CreditInstallments map[string]float64 `json:"credit_installments"`
}

// MaxFeeRatePercent bounds every configurable payment fee rate.
const MaxFeeRatePercent = 30.0

// Value implements driver.Valuer so a FeeRates serializes to jsonb. A nil pointer
// serializes to a SQL NULL (so an Updates map carrying a typed-nil *FeeRates never
// panics and correctly clears the column).
func (f *FeeRates) Value() (driver.Value, error) {
	if f == nil {
		return nil, nil
	}
	return json.Marshal(*f)
}

// Scan implements sql.Scanner so a jsonb column deserializes into FeeRates.
func (f *FeeRates) Scan(src interface{}) error {
	if src == nil {
		return nil
	}
	switch v := src.(type) {
	case []byte:
		return json.Unmarshal(v, f)
	case string:
		return json.Unmarshal([]byte(v), f)
	default:
		return fmt.Errorf("unsupported type for FeeRates: %T", src)
	}
}

// CreditRate returns the card fee for the given installment count and whether it is
// known. N<=1 uses the single-installment ("crédito à vista") rate, which is always
// known within a configured table; N>=2 is looked up in CreditInstallments and is
// unknown when absent. A nil table is always unknown.
func (f *FeeRates) CreditRate(installments int) (float64, bool) {
	if f == nil {
		return 0, false
	}
	if installments <= 1 {
		return f.Credit1x, true
	}
	v, ok := f.CreditInstallments[strconv.Itoa(installments)]
	return v, ok
}

// rateFor resolves the applicable rate for a (method, installments) pair. Only the two
// known methods are recognised (case-insensitively): "pix" uses the Pix rate and
// "credit_card" uses the card rate. Any other/empty method is UNKNOWN (ok=false) so the
// fee is not guessed.
func (f *FeeRates) rateFor(method string, installments int) (float64, bool) {
	if f == nil {
		return 0, false
	}
	switch {
	case strings.EqualFold(method, CaptureMethodPix):
		return f.Pix, true
	case strings.EqualFold(method, CaptureMethodCreditCard):
		return f.CreditRate(installments)
	default:
		return 0, false
	}
}

// feeCents rounds paid × rate/100 to the nearest cent (half away from zero, the same
// rounding the pricing/money helpers use).
func feeCents(paidCents int64, ratePercent float64) int64 {
	if paidCents <= 0 || ratePercent <= 0 {
		return 0
	}
	return int64(math.Round(float64(paidCents) * ratePercent / 100.0))
}

// FeeFor computes the InfinitePay fee for a captured payment from the org's fee table.
// It returns the applicable rate (percent), the fee in cents and whether the rate was
// known. When ok is false (table not configured, or the installment has no rate) the
// caller shows the fee/net as unknown (null).
//
//   - method "pix"                -> Pix rate
//   - method "credit_card" with N<=1 -> single-installment ("à vista") rate
//   - method "credit_card" with N>=2 -> CreditInstallments[N] (unknown when absent)
func FeeFor(rates *FeeRates, method string, installments int, paidCents int64) (float64, int64, bool) {
	rate, ok := rates.rateFor(method, installments)
	if !ok {
		return 0, 0, false
	}
	return rate, feeCents(paidCents, rate), true
}

// ErrFeeRateMissing is returned when embed_fee_in_price is requested but the reference
// card rate (credit at assured_installments) is not configured.
var ErrFeeRateMissing = errors.New("payment_fee_rate_missing")

// ValidateFeeRates checks every configured rate is within [0, MaxFeeRatePercent] and
// has at most two decimal places, and that installment keys are "2".."12". A nil table
// is valid (not configured).
func ValidateFeeRates(rates *FeeRates) error {
	if rates == nil {
		return nil
	}
	if err := validateRate(rates.Pix); err != nil {
		return err
	}
	if err := validateRate(rates.Credit1x); err != nil {
		return err
	}
	for k, v := range rates.CreditInstallments {
		n, err := strconv.Atoi(k)
		if err != nil || n < 2 || n > MaxInstallments {
			return fmt.Errorf("invalid installment key %q", k)
		}
		if err := validateRate(v); err != nil {
			return err
		}
	}
	return nil
}

// validateRate enforces the 0..30 range and the two-decimal-place precision.
func validateRate(rate float64) error {
	if rate < 0 || rate > MaxFeeRatePercent {
		return fmt.Errorf("fee rate %.2f out of range [0, %.0f]", rate, MaxFeeRatePercent)
	}
	// Reject more than two decimal places (e.g. 4.255) within a small epsilon.
	scaled := rate * 100
	if math.Abs(scaled-math.Round(scaled)) > 1e-6 {
		return fmt.Errorf("fee rate %v has more than two decimal places", rate)
	}
	return nil
}

// ReferenceRate returns the rate the price-embed switch uses: the card fee at
// assured_installments (à vista when N<=1). ok is false when it is not configured, in
// which case embed_fee_in_price must be refused (422 payment_fee_rate_missing).
func (f *FeeRates) ReferenceRate(assuredInstallments int) (float64, bool) {
	return f.CreditRate(assuredInstallments)
}
