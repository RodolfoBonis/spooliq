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
// the webhook and the redirect confirm the same payment.
func PaymentReceivedNotification(quoteNumber *int, name, budgetID string, amountCents int64, dedupeKey string) notificationEntities.NewNotification {
	return notificationEntities.NewNotification{
		Type:      notificationEntities.TypePaymentPaid,
		Title:     "Pagamento recebido — " + budgetLabel(quoteNumber, name),
		Body:      "O cliente pagou " + formatBRL(amountCents) + " pelo link.",
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
