// Package entities holds the payments domain entities and the pure money math that
// decides what to charge next on a budget. Everything here is deterministic and
// side-effect free so it can be exhaustively unit-tested.
package entities

// Money constants. All amounts are in cents (BRL).
const (
	// MinChargeCents is the InfinitePay minimum charge: R$ 1,00.
	MinChargeCents int64 = 100
	// MaxInstallments is the hard cap on card installments.
	MaxInstallments = 12
)

// ChargeKind identifies what a given charge represents on a budget.
type ChargeKind string

// Charge kinds.
const (
	// KindDeposit is the up-front deposit (sinal) when a deposit percent is set.
	KindDeposit ChargeKind = "deposit"
	// KindBalance is the remaining balance after a partial payment.
	KindBalance ChargeKind = "balance"
	// KindFull is the entire total in a single charge.
	KindFull ChargeKind = "full"
)

// PaymentStatus is the aggregate payment state of a budget.
type PaymentStatus string

// Payment statuses.
const (
	// StatusUnpaid: nothing has been paid yet.
	StatusUnpaid PaymentStatus = "unpaid"
	// StatusPartial: some but not all of the total has been paid.
	StatusPartial PaymentStatus = "partial"
	// StatusPaid: the total has been fully covered.
	StatusPaid PaymentStatus = "paid"
)

// Reasons a budget is not currently chargeable.
const (
	// ReasonBelowMinimum: the charge (or remaining balance) is under R$ 1.
	ReasonBelowMinimum = "below_minimum"
	// ReasonAlreadyPaid: the total is already covered.
	ReasonAlreadyPaid = "already_paid"
	// ReasonInvalidTotal: the total is zero or negative.
	ReasonInvalidTotal = "invalid_total"
)

// NextCharge is the result of deciding what to charge next on a budget.
type NextCharge struct {
	Kind       ChargeKind
	Amount     int64 // cents
	Chargeable bool
	// Reason explains why Chargeable is false (empty when chargeable).
	Reason string
}

// DepositAmount returns the deposit for total at depositPercent, rounded HALF-UP in
// basis points so e.g. 50% of R$ 2,01 (201c) = 100.5c -> 101c. depositPercent is a
// percentage (0..100); negative or zero yields 0.
func DepositAmount(total int64, depositPercent float64) int64 {
	if total <= 0 || depositPercent <= 0 {
		return 0
	}
	// Basis points (percent * 100), rounded to the nearest integer bp.
	bps := int64(depositPercent*100 + 0.5)
	// Half-up division by 10_000 (100% in bps).
	return (total*bps + 5000) / 10000
}

// NextChargeFor decides the next charge for a budget given its total, the amount
// already paid (sum of paid charges' principal) and the optional deposit percent
// (nil => charge the full total). The rules:
//   - total <= 0                      -> not chargeable (invalid_total)
//   - paid >= total                   -> not chargeable (already_paid)
//   - paid > 0 (partial)              -> balance = total-paid; chargeable unless < R$1
//   - nothing paid, deposit set       -> deposit; but if the deposit OR the remainder
//     is under R$1 (or the deposit covers the whole total) -> charge the full total
//   - nothing paid, no deposit        -> full; not chargeable if total < R$1
func NextChargeFor(total, paid int64, depositPercent *float64) NextCharge {
	if total <= 0 {
		return NextCharge{Kind: KindFull, Amount: total, Chargeable: false, Reason: ReasonInvalidTotal}
	}
	if paid >= total {
		return NextCharge{Kind: KindFull, Amount: 0, Chargeable: false, Reason: ReasonAlreadyPaid}
	}
	if paid > 0 {
		balance := total - paid
		if balance < MinChargeCents {
			return NextCharge{Kind: KindBalance, Amount: balance, Chargeable: false, Reason: ReasonBelowMinimum}
		}
		return NextCharge{Kind: KindBalance, Amount: balance, Chargeable: true}
	}
	// Nothing paid yet.
	if depositPercent != nil {
		dep := DepositAmount(total, *depositPercent)
		remainder := total - dep
		if dep < MinChargeCents || remainder < MinChargeCents || dep >= total {
			// Splitting makes no sense: charge the whole total at once.
			return NextCharge{Kind: KindFull, Amount: total, Chargeable: true}
		}
		return NextCharge{Kind: KindDeposit, Amount: dep, Chargeable: true}
	}
	if total < MinChargeCents {
		return NextCharge{Kind: KindFull, Amount: total, Chargeable: false, Reason: ReasonBelowMinimum}
	}
	return NextCharge{Kind: KindFull, Amount: total, Chargeable: true}
}

// PixPrice returns the amount a Pix payer pays after the Pix discount, rounded
// HALF-UP. pixDiscountPercent is a percentage (0..100); values <= 0 return amount
// unchanged.
func PixPrice(amount int64, pixDiscountPercent float64) int64 {
	if amount <= 0 || pixDiscountPercent <= 0 {
		return amount
	}
	discount := DepositAmount(amount, pixDiscountPercent)
	return amount - discount
}

// InterestFreeInstallment returns the per-installment amount for n interest-free
// installments of amount, rounded HALF-UP. n <= 1 (or amount <= 0) returns amount.
func InterestFreeInstallment(amount int64, n int) int64 {
	if n <= 1 || amount <= 0 {
		return amount
	}
	return (amount + int64(n)/2) / int64(n)
}

// Status reports the aggregate payment status for total vs. amount paid.
func Status(total, paid int64) PaymentStatus {
	switch {
	case paid <= 0:
		return StatusUnpaid
	case paid >= total:
		return StatusPaid
	default:
		return StatusPartial
	}
}
