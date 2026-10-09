package usecases

import (
	"testing"

	"github.com/RodolfoBonis/spooliq/features/budget/domain/entities"
	notificationEntities "github.com/RodolfoBonis/spooliq/features/notification/domain/entities"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestCustomerResponseNotification(t *testing.T) {
	q := 12
	budget := &entities.BudgetEntity{ID: uuid.New(), Name: "Vaso", QuoteNumber: &q}
	reason := "muito caro"

	approved := CustomerResponseNotification(budget, entities.StatusApproved, "Ana", nil)
	assert.Equal(t, notificationEntities.TypeBudgetApproved, approved.Type)
	assert.Equal(t, "Orçamento #12 Vaso aprovado", approved.Title)
	assert.Equal(t, "/budgets/"+budget.ID.String(), approved.Link)

	rejected := CustomerResponseNotification(budget, entities.StatusRejected, "Ana", &reason)
	assert.Equal(t, notificationEntities.TypeBudgetRejected, rejected.Type)
	assert.Contains(t, rejected.Body, "muito caro")
}

func TestExpiredNotificationWithoutQuoteNumber(t *testing.T) {
	n := ExpiredNotification(entities.ExpiredBudget{ID: uuid.New(), Name: "Vaso"})
	assert.Equal(t, "Orçamento Vaso expirou", n.Title)
}
