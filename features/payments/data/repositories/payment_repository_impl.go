package repositories

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/RodolfoBonis/spooliq/features/payments/data/models"
	"github.com/RodolfoBonis/spooliq/features/payments/domain/entities"
	domainRepos "github.com/RodolfoBonis/spooliq/features/payments/domain/repositories"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// paymentRepository is the GORM implementation of the payments data port.
type paymentRepository struct {
	db *gorm.DB
}

// NewPaymentRepository builds the payments repository.
func NewPaymentRepository(db *gorm.DB) domainRepos.PaymentRepository {
	return &paymentRepository{db: db}
}

// WithTransaction runs fn against a transaction-bound repository.
func (r *paymentRepository) WithTransaction(ctx context.Context, fn func(repo domainRepos.PaymentRepository) error) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return fn(&paymentRepository{db: tx})
	})
}

// --- Integration ---

func (r *paymentRepository) GetIntegration(ctx context.Context, organizationID, provider string) (*entities.PaymentIntegration, error) {
	var m models.PaymentIntegrationModel
	err := r.db.WithContext(ctx).
		Where("organization_id = ? AND provider = ?", organizationID, provider).
		First(&m).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return m.ToEntity(), nil
}

func (r *paymentRepository) CreateIntegration(ctx context.Context, integration *entities.PaymentIntegration) error {
	m := models.PaymentIntegrationModel{
		ID:                     integration.ID,
		OrganizationID:         integration.OrganizationID,
		Provider:               integration.Provider,
		Handle:                 integration.Handle,
		APIKeyCiphertext:       integration.APIKeyCiphertext,
		APIKeyLast4:            integration.APIKeyLast4,
		WebhookTokenCiphertext: integration.WebhookTokenCiphertext,
		Enabled:                integration.Enabled,
		AssuredInstallments:    integration.AssuredInstallments,
		PixDiscountPercent:     integration.PixDiscountPercent,
		DefaultDepositPercent:  integration.DefaultDepositPercent,
		VerifiedAt:             integration.VerifiedAt,
		DisconnectedAt:         integration.DisconnectedAt,
	}
	if m.ID == uuid.Nil {
		m.ID = uuid.New()
	}
	return r.db.WithContext(ctx).Create(&m).Error
}

func (r *paymentRepository) UpdateIntegrationFields(ctx context.Context, id uuid.UUID, fields map[string]interface{}) error {
	return r.db.WithContext(ctx).Model(&models.PaymentIntegrationModel{}).
		Where("id = ?", id).
		Updates(fields).Error
}

func (r *paymentRepository) CountPendingLinks(ctx context.Context, organizationID string) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&models.BudgetPaymentModel{}).
		Where("organization_id = ? AND status = ?", organizationID, string(entities.PaymentPending)).
		Count(&count).Error
	return count, err
}

func (r *paymentRepository) ListDisconnectedBefore(ctx context.Context, cutoff time.Time) ([]*entities.PaymentIntegration, error) {
	var ms []models.PaymentIntegrationModel
	err := r.db.WithContext(ctx).
		Where("disconnected_at IS NOT NULL AND disconnected_at <= ?", cutoff).
		Find(&ms).Error
	if err != nil {
		return nil, err
	}
	out := make([]*entities.PaymentIntegration, 0, len(ms))
	for i := range ms {
		out = append(out, ms[i].ToEntity())
	}
	return out, nil
}

func (r *paymentRepository) DeleteIntegration(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Delete(&models.PaymentIntegrationModel{}, "id = ?", id).Error
}

// --- Budget payments ---

func (r *paymentRepository) GetPaymentsByBudget(ctx context.Context, budgetID uuid.UUID, organizationID string) ([]*entities.BudgetPayment, error) {
	var ms []models.BudgetPaymentModel
	err := r.db.WithContext(ctx).
		Where("budget_id = ? AND organization_id = ?", budgetID, organizationID).
		Order("created_at DESC").
		Find(&ms).Error
	if err != nil {
		return nil, err
	}
	out := make([]*entities.BudgetPayment, 0, len(ms))
	for i := range ms {
		out = append(out, ms[i].ToEntity())
	}
	return out, nil
}

func (r *paymentRepository) GetPendingByBudgetKind(ctx context.Context, budgetID uuid.UUID, kind entities.ChargeKind) (*entities.BudgetPayment, error) {
	var m models.BudgetPaymentModel
	err := r.db.WithContext(ctx).
		Where("budget_id = ? AND kind = ? AND status = ?", budgetID, string(kind), string(entities.PaymentPending)).
		First(&m).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return m.ToEntity(), nil
}

func (r *paymentRepository) GetPaymentByOrderNSU(ctx context.Context, organizationID, orderNSU string) (*entities.BudgetPayment, error) {
	var m models.BudgetPaymentModel
	err := r.db.WithContext(ctx).
		Where("organization_id = ? AND order_nsu = ?", organizationID, orderNSU).
		First(&m).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return m.ToEntity(), nil
}

func (r *paymentRepository) CreatePayment(ctx context.Context, payment *entities.BudgetPayment) error {
	m := models.BudgetPaymentModel{
		ID:              payment.ID,
		OrganizationID:  payment.OrganizationID,
		BudgetID:        payment.BudgetID,
		Kind:            string(payment.Kind),
		OrderNSU:        payment.OrderNSU,
		TransactionNSU:  payment.TransactionNSU,
		Slug:            payment.Slug,
		AmountCents:     payment.AmountCents,
		PaidAmountCents: payment.PaidAmountCents,
		Installments:    payment.Installments,
		CaptureMethod:   payment.CaptureMethod,
		CheckoutURL:     payment.CheckoutURL,
		ReceiptURL:      payment.ReceiptURL,
		Status:          string(payment.Status),
		StatusReason:    payment.StatusReason,
		LinkExpiresAt:   payment.LinkExpiresAt,
		PaidAt:          payment.PaidAt,
		CheckAttempts:   payment.CheckAttempts,
	}
	if m.ID == uuid.Nil {
		m.ID = uuid.New()
	}
	if m.Status == "" {
		m.Status = string(entities.PaymentPending)
	}
	return r.db.WithContext(ctx).Create(&m).Error
}

func (r *paymentRepository) UpdatePaymentFields(ctx context.Context, id uuid.UUID, fields map[string]interface{}) error {
	return r.db.WithContext(ctx).Model(&models.BudgetPaymentModel{}).
		Where("id = ?", id).
		Updates(fields).Error
}

func (r *paymentRepository) MarkPaidGuarded(ctx context.Context, id uuid.UUID, fields map[string]interface{}) (int64, error) {
	res := r.db.WithContext(ctx).Model(&models.BudgetPaymentModel{}).
		Where("id = ? AND status IN ?", id, []string{string(entities.PaymentPending), string(entities.PaymentCanceled)}).
		Updates(fields)
	return res.RowsAffected, res.Error
}

func (r *paymentRepository) CancelPendingForBudget(ctx context.Context, budgetID uuid.UUID, organizationID, reason string) error {
	return r.db.WithContext(ctx).Model(&models.BudgetPaymentModel{}).
		Where("budget_id = ? AND organization_id = ? AND status = ?", budgetID, organizationID, string(entities.PaymentPending)).
		Updates(map[string]interface{}{
			"status":        string(entities.PaymentCanceled),
			"status_reason": reason,
			"canceled_at":   time.Now(),
		}).Error
}

func (r *paymentRepository) CancelPendingForOrg(ctx context.Context, organizationID, reason string) error {
	return r.db.WithContext(ctx).Model(&models.BudgetPaymentModel{}).
		Where("organization_id = ? AND status = ?", organizationID, string(entities.PaymentPending)).
		Updates(map[string]interface{}{
			"status":        string(entities.PaymentCanceled),
			"status_reason": reason,
			"canceled_at":   time.Now(),
		}).Error
}

func (r *paymentRepository) ListExpiredPendingLinks(ctx context.Context, now time.Time, limit int) ([]*entities.BudgetPayment, error) {
	var ms []models.BudgetPaymentModel
	err := r.db.WithContext(ctx).
		Where("status = ? AND link_expires_at IS NOT NULL AND link_expires_at < ?", string(entities.PaymentPending), now).
		Limit(limit).
		Find(&ms).Error
	if err != nil {
		return nil, err
	}
	out := make([]*entities.BudgetPayment, 0, len(ms))
	for i := range ms {
		out = append(out, ms[i].ToEntity())
	}
	return out, nil
}

func (r *paymentRepository) ListPendingWithIdentifiers(ctx context.Context, since time.Time, limit int) ([]*entities.BudgetPayment, error) {
	var ms []models.BudgetPaymentModel
	err := r.db.WithContext(ctx).
		Where("status = ? AND created_at >= ? AND (slug IS NOT NULL OR transaction_nsu IS NOT NULL)", string(entities.PaymentPending), since).
		Limit(limit).
		Find(&ms).Error
	if err != nil {
		return nil, err
	}
	out := make([]*entities.BudgetPayment, 0, len(ms))
	for i := range ms {
		out = append(out, ms[i].ToEntity())
	}
	return out, nil
}

// --- Budget aggregate ---

// budgetChargeRow is the scan target for the budget slice + customer contact.
type budgetChargeRow struct {
	ID              uuid.UUID
	OrganizationID  string
	PublicToken     *string
	Status          string
	TotalCost       int64
	AmountPaidCents int64
	DepositPercent  *float64
	ValidUntil      *time.Time
	Name            string
	QuoteNumber     *int
	CustomerName    string
	CustomerEmail   *string
	CustomerPhone   *string
}

func (row *budgetChargeRow) toContext() *entities.BudgetChargeContext {
	ctxOut := &entities.BudgetChargeContext{
		ID:              row.ID,
		OrganizationID:  row.OrganizationID,
		PublicToken:     row.PublicToken,
		Status:          row.Status,
		TotalCost:       row.TotalCost,
		AmountPaidCents: row.AmountPaidCents,
		DepositPercent:  row.DepositPercent,
		ValidUntil:      row.ValidUntil,
		Name:            row.Name,
		QuoteNumber:     row.QuoteNumber,
		CustomerName:    row.CustomerName,
	}
	if row.CustomerEmail != nil {
		ctxOut.CustomerEmail = *row.CustomerEmail
	}
	if row.CustomerPhone != nil {
		ctxOut.CustomerPhone = *row.CustomerPhone
	}
	return ctxOut
}

const budgetChargeSelect = `
	b.id, b.organization_id, b.public_token, b.status, b.total_cost,
	b.amount_paid_cents, b.deposit_percent, b.valid_until, b.name, b.quote_number,
	c.name AS customer_name, c.email AS customer_email, c.phone AS customer_phone`

func (r *paymentRepository) GetBudgetChargeContext(ctx context.Context, budgetID uuid.UUID, organizationID string) (*entities.BudgetChargeContext, error) {
	var row budgetChargeRow
	err := r.db.WithContext(ctx).
		Table("budgets b").
		Select(budgetChargeSelect).
		Joins("LEFT JOIN customers c ON c.id = b.customer_id").
		Where("b.id = ? AND b.organization_id = ? AND b.deleted_at IS NULL", budgetID, organizationID).
		Scan(&row).Error
	if err != nil {
		return nil, err
	}
	if row.ID == uuid.Nil {
		return nil, nil
	}
	return row.toContext(), nil
}

func (r *paymentRepository) GetBudgetChargeContextByToken(ctx context.Context, token string) (*entities.BudgetChargeContext, error) {
	var row budgetChargeRow
	err := r.db.WithContext(ctx).
		Table("budgets b").
		Select(budgetChargeSelect).
		Joins("LEFT JOIN customers c ON c.id = b.customer_id").
		Where("b.public_token = ? AND b.deleted_at IS NULL", token).
		Scan(&row).Error
	if err != nil {
		return nil, err
	}
	if row.ID == uuid.Nil {
		return nil, nil
	}
	return row.toContext(), nil
}

func (r *paymentRepository) SumPaidPrincipal(ctx context.Context, budgetID uuid.UUID) (int64, error) {
	var sum *int64
	err := r.db.WithContext(ctx).Model(&models.BudgetPaymentModel{}).
		Where("budget_id = ? AND status = ?", budgetID, string(entities.PaymentPaid)).
		Select("COALESCE(SUM(amount_cents), 0)").
		Scan(&sum).Error
	if err != nil {
		return 0, err
	}
	if sum == nil {
		return 0, nil
	}
	return *sum, nil
}

func (r *paymentRepository) UpdateBudgetAggregate(ctx context.Context, budgetID uuid.UUID, amountPaid int64, status string, paidAt *time.Time) error {
	return r.db.WithContext(ctx).
		Table("budgets").
		Where("id = ?", budgetID).
		Updates(map[string]interface{}{
			"amount_paid_cents": amountPaid,
			"payment_status":    status,
			"paid_at":           paidAt,
			"updated_at":        time.Now(),
		}).Error
}

// --- Webhook events ---

func (r *paymentRepository) CreateWebhookEvent(ctx context.Context, organizationID string, payload []byte, receivedAt time.Time) (uuid.UUID, error) {
	m := models.PaymentWebhookEventModel{
		ID:             uuid.New(),
		OrganizationID: organizationID,
		Payload:        json.RawMessage(payload),
		ReceivedAt:     receivedAt,
	}
	if err := r.db.WithContext(ctx).Create(&m).Error; err != nil {
		return uuid.Nil, err
	}
	return m.ID, nil
}

func (r *paymentRepository) MarkWebhookProcessed(ctx context.Context, id uuid.UUID) error {
	now := time.Now()
	return r.db.WithContext(ctx).Model(&models.PaymentWebhookEventModel{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{"processed_at": now}).Error
}

func (r *paymentRepository) MarkWebhookAttempt(ctx context.Context, id uuid.UUID, lastError string) error {
	return r.db.WithContext(ctx).Model(&models.PaymentWebhookEventModel{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"attempts":   gorm.Expr("attempts + 1"),
			"last_error": lastError,
		}).Error
}

func (r *paymentRepository) ListUnprocessedWebhooks(ctx context.Context, since time.Time, maxAttempts, limit int) ([]*entities.WebhookEventRecord, error) {
	var ms []models.PaymentWebhookEventModel
	err := r.db.WithContext(ctx).
		Where("processed_at IS NULL AND received_at >= ? AND attempts < ?", since, maxAttempts).
		Order("received_at ASC").
		Limit(limit).
		Find(&ms).Error
	if err != nil {
		return nil, err
	}
	out := make([]*entities.WebhookEventRecord, 0, len(ms))
	for i := range ms {
		out = append(out, &entities.WebhookEventRecord{
			ID:             ms[i].ID,
			OrganizationID: ms[i].OrganizationID,
			Payload:        ms[i].Payload,
			Attempts:       ms[i].Attempts,
		})
	}
	return out, nil
}
