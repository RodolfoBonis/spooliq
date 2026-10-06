package subscriptions

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	otellogger "github.com/RodolfoBonis/go-otel-agent/logger"
	adminEntities "github.com/RodolfoBonis/spooliq/features/admin/domain/entities"
	"github.com/RodolfoBonis/spooliq/features/subscriptions/domain/entities"
	"github.com/RodolfoBonis/spooliq/features/subscriptions/domain/usecases"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type noopLogger struct{}

func (noopLogger) Debug(context.Context, string, ...otellogger.Fields)   {}
func (noopLogger) Info(context.Context, string, ...otellogger.Fields)    {}
func (noopLogger) Warning(context.Context, string, ...otellogger.Fields) {}
func (noopLogger) Error(context.Context, string, ...otellogger.Fields)   {}
func (noopLogger) Fatal(context.Context, string, ...otellogger.Fields)   {}
func (noopLogger) Panic(context.Context, string, ...otellogger.Fields)   {}
func (n noopLogger) With(otellogger.Fields) otellogger.Logger            { return n }
func (noopLogger) LogError(context.Context, string, error)               {}

// stubPlanRepo implements repositories.SubscriptionPlanRepository with only
// FindAllActive meaningfully populated; the rest return zero values.
type stubPlanRepo struct {
	active []*entities.SubscriptionPlanEntity
}

func (r *stubPlanRepo) Create(context.Context, *entities.SubscriptionPlanEntity) error { return nil }
func (r *stubPlanRepo) FindByID(context.Context, uuid.UUID) (*entities.SubscriptionPlanEntity, error) {
	return nil, nil
}
func (r *stubPlanRepo) FindByName(context.Context, string) (*entities.SubscriptionPlanEntity, error) {
	return nil, nil
}
func (r *stubPlanRepo) FindAll(context.Context) ([]*entities.SubscriptionPlanEntity, error) {
	return r.active, nil
}
func (r *stubPlanRepo) FindAllActive(context.Context) ([]*entities.SubscriptionPlanEntity, error) {
	return r.active, nil
}
func (r *stubPlanRepo) Update(context.Context, *entities.SubscriptionPlanEntity) error { return nil }
func (r *stubPlanRepo) Delete(context.Context, uuid.UUID) error                        { return nil }
func (r *stubPlanRepo) GetPlanStats(context.Context, uuid.UUID) (*adminEntities.PlanStats, error) {
	return nil, nil
}
func (r *stubPlanRepo) GetPlanCompanies(context.Context, uuid.UUID, int, int, string) (*adminEntities.ListPlanCompaniesResponse, error) {
	return nil, nil
}
func (r *stubPlanRepo) GetPlanFinancialReport(context.Context, uuid.UUID, string) (*adminEntities.PlanFinancialReport, error) {
	return nil, nil
}
func (r *stubPlanRepo) CanDeletePlan(context.Context, uuid.UUID) (*adminEntities.PlanDeletionCheck, error) {
	return nil, nil
}
func (r *stubPlanRepo) BulkUpdate(context.Context, []uuid.UUID, map[string]interface{}, string, string, string) (*adminEntities.BulkOperationResult, error) {
	return nil, nil
}
func (r *stubPlanRepo) BulkActivate(context.Context, []uuid.UUID, string, string, string) (*adminEntities.BulkOperationResult, error) {
	return nil, nil
}
func (r *stubPlanRepo) BulkDeactivate(context.Context, []uuid.UUID, string, string, string) (*adminEntities.BulkOperationResult, error) {
	return nil, nil
}
func (r *stubPlanRepo) CreateAuditEntry(context.Context, *adminEntities.PlanAuditEntry) error {
	return nil
}
func (r *stubPlanRepo) GetPlanHistory(context.Context, uuid.UUID, int, int) (*adminEntities.PlanAuditResponse, error) {
	return nil, nil
}
func (r *stubPlanRepo) CreateTemplate(context.Context, *adminEntities.PlanTemplate) error { return nil }
func (r *stubPlanRepo) GetTemplates(context.Context, string) ([]*adminEntities.PlanTemplate, error) {
	return nil, nil
}
func (r *stubPlanRepo) GetTemplateByID(context.Context, uuid.UUID) (*adminEntities.PlanTemplate, error) {
	return nil, nil
}
func (r *stubPlanRepo) CreatePlanFromTemplate(context.Context, uuid.UUID, map[string]interface{}, string, string, string) (*entities.SubscriptionPlanEntity, error) {
	return nil, nil
}
func (r *stubPlanRepo) IncrementTemplateUsage(context.Context, uuid.UUID) error { return nil }
func (r *stubPlanRepo) GetAvailableFeatures(context.Context) ([]*adminEntities.AvailableFeature, error) {
	return nil, nil
}
func (r *stubPlanRepo) ValidateFeatures(context.Context, []adminEntities.PlanTemplateFeature) (*adminEntities.FeatureValidationResult, error) {
	return nil, nil
}
func (r *stubPlanRepo) CreateMigration(context.Context, *adminEntities.PlanMigrationRequest, string, string) (*adminEntities.PlanMigrationResult, error) {
	return nil, nil
}
func (r *stubPlanRepo) ExecutePlanMigration(context.Context, uuid.UUID) (*adminEntities.PlanMigrationResult, error) {
	return nil, nil
}
func (r *stubPlanRepo) GetMigrationStatus(context.Context, uuid.UUID) (*adminEntities.PlanMigrationResult, error) {
	return nil, nil
}

type planEnvelope struct {
	Data       []json.RawMessage `json:"data"`
	Total      int64             `json:"total"`
	Page       int               `json:"page"`
	PageSize   int               `json:"page_size"`
	TotalPages int               `json:"total_pages"`
}

// GET /plans returns the standard envelope (not a bare array).
func TestListActivePlans_Envelope(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repo := &stubPlanRepo{active: []*entities.SubscriptionPlanEntity{
		{ID: uuid.New(), Name: "Basic"},
		{ID: uuid.New(), Name: "Pro"},
		{ID: uuid.New(), Name: "Enterprise"},
	}}
	uc := usecases.NewSubscriptionPlanUseCase(repo, noopLogger{})

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/plans?page_size=2", nil)

	uc.ListActivePlans(c)

	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	var env planEnvelope
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &env))
	assert.Equal(t, int64(3), env.Total)
	assert.Len(t, env.Data, 2)
	assert.Equal(t, 2, env.PageSize)
	assert.Equal(t, 2, env.TotalPages)
}
