package usecases

import (
	"fmt"
	"time"

	activityEntities "github.com/RodolfoBonis/spooliq/features/activity/domain/entities"
	"github.com/RodolfoBonis/spooliq/features/budget/domain/entities"
	notificationEntities "github.com/RodolfoBonis/spooliq/features/notification/domain/entities"
)

// budgetLabel is "#12 Vaso" (or just the name before a quote number exists).
func budgetLabel(quoteNumber *int, name string) string {
	if quoteNumber == nil || *quoteNumber == 0 {
		return name
	}
	return fmt.Sprintf("#%d %s", *quoteNumber, name)
}

func budgetLink(id string) string { return "/budgets/" + id }

// formatBRL renders a cents amount as "R$ 1234,50" (no thousands separator; the
// clients localize for display — this is a best-effort label).
func formatBRL(cents int64) string {
	if cents < 0 {
		cents = 0
	}
	return fmt.Sprintf("R$ %d,%02d", cents/100, cents%100)
}

// CustomerResponseNotification tells the team a customer answered a shared budget.
func CustomerResponseNotification(budget *entities.BudgetEntity, status entities.BudgetStatus, customerName string, reason *string) notificationEntities.NewNotification {
	label := budgetLabel(budget.QuoteNumber, budget.Name)
	if status == entities.StatusRejected {
		body := customerName + " recusou o orçamento."
		if reason != nil && *reason != "" {
			body = fmt.Sprintf("%s recusou: %q", customerName, *reason)
		}
		return notificationEntities.NewNotification{
			Type:  notificationEntities.TypeBudgetRejected,
			Title: "Orçamento " + label + " recusado",
			Body:  body,
			Link:  budgetLink(budget.ID.String()),
		}
	}
	return notificationEntities.NewNotification{
		Type:  notificationEntities.TypeBudgetApproved,
		Title: "Orçamento " + label + " aprovado",
		Body:  customerName + " aprovou o orçamento pelo link.",
		Link:  budgetLink(budget.ID.String()),
	}
}

// ExpiredNotification tells the team a sent budget passed its validity.
func ExpiredNotification(b entities.ExpiredBudget) notificationEntities.NewNotification {
	return notificationEntities.NewNotification{
		Type:  notificationEntities.TypeBudgetExpired,
		Title: "Orçamento " + budgetLabel(b.QuoteNumber, b.Name) + " expirou",
		Body:  "A validade passou sem resposta do cliente.",
		Link:  budgetLink(b.ID.String()),
	}
}

// PaymentReceivedNotification tells the team a customer paid (deposit, balance or
// full) on a budget. dedupeKey (per payment) prevents a duplicate notification when
// the webhook and the redirect confirm the same payment. When netCents is non-nil the
// body also shows the net amount the shop receives after the InfinitePay fee.
func PaymentReceivedNotification(quoteNumber *int, name, budgetID string, amountCents int64, netCents *int64, dedupeKey string) notificationEntities.NewNotification {
	body := "O cliente pagou " + formatBRL(amountCents) + " pelo link."
	if netCents != nil {
		body = "O cliente pagou " + formatBRL(amountCents) + " pelo link (você recebe ≈ " + formatBRL(*netCents) + ")."
	}
	return notificationEntities.NewNotification{
		Type:      notificationEntities.TypePaymentPaid,
		Title:     "Pagamento recebido — " + budgetLabel(quoteNumber, name),
		Body:      body,
		Link:      budgetLink(budgetID),
		DedupeKey: dedupeKey,
	}
}

// PaidAfterCancelNotification warns the team that a cancelled link was paid anyway,
// so the payment must be refunded through the InfinitePay app.
func PaidAfterCancelNotification(quoteNumber *int, name, budgetID string) notificationEntities.NewNotification {
	return notificationEntities.NewNotification{
		Type:  notificationEntities.TypePaymentPaid,
		Title: "Pagamento em link cancelado — " + budgetLabel(quoteNumber, name),
		Body:  "Um link já cancelado foi pago. Reembolse pela InfinitePay.",
		Link:  budgetLink(budgetID),
	}
}

// PossibleRefundNotification warns that a previously-paid payment no longer reports
// paid at the gateway (a likely refund). dedupeKey suppresses repeats from webhook/job
// retries. It never auto-flips the record — the team confirms on the InfinitePay app.
func PossibleRefundNotification(quoteNumber *int, name, budgetID, dedupeKey string) notificationEntities.NewNotification {
	return notificationEntities.NewNotification{
		Type:      notificationEntities.TypePaymentPaid,
		Title:     "Possível estorno — " + budgetLabel(quoteNumber, name),
		Body:      "Um pagamento não consta mais como pago. Confira na InfinitePay.",
		Link:      budgetLink(budgetID),
		DedupeKey: dedupeKey,
	}
}

// PaymentReceivedActivity builds the activity log entry for a received payment.
func PaymentReceivedActivity(organizationID, budgetID, name string, quoteNumber *int, amountCents int64, now time.Time) activityEntities.ActivityEntity {
	qn := 0
	if quoteNumber != nil {
		qn = *quoteNumber
	}
	return activityEntities.ActivityEntity{
		OrganizationID: organizationID,
		UserID:         "system",
		Action:         activityEntities.ActionPaymentReceived,
		EntityType:     activityEntities.EntityBudget,
		EntityID:       budgetID,
		EntityName:     name,
		Metadata: map[string]any{
			"quote_number": qn,
			"amount_cents": amountCents,
		},
		CreatedAt: now,
	}
}
