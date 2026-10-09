package usecases

import (
	"fmt"

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
