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

func TestPaymentReceivedNotification_WithNet(t *testing.T) {
	q := 7
	// R$ 217,12 paid, net R$ 199,97 after the fee.
	net := int64(19997)
	n := PaymentReceivedNotification(&q, "Vaso", "bid", 21712, &net, "dedupe")
	assert.Equal(t, notificationEntities.TypePaymentPaid, n.Type)
	assert.Equal(t, "Pagamento recebido — #7 Vaso", n.Title)
	assert.Equal(t, "O cliente pagou R$ 217,12 pelo link (você recebe ≈ R$ 199,97).", n.Body)
	assert.Equal(t, "dedupe", n.DedupeKey)
}

func TestPaymentReceivedNotification_WithoutNet(t *testing.T) {
	n := PaymentReceivedNotification(nil, "Vaso", "bid", 21712, nil, "dedupe")
	assert.Equal(t, "O cliente pagou R$ 217,12 pelo link.", n.Body)
}
