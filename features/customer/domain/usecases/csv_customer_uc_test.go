package usecases

import (
	"context"
	"testing"

	otellogger "github.com/RodolfoBonis/go-otel-agent/logger"
	activityEntities "github.com/RodolfoBonis/spooliq/features/activity/domain/entities"
	activityUc "github.com/RodolfoBonis/spooliq/features/activity/domain/usecases"
	"github.com/RodolfoBonis/spooliq/features/customer/domain/entities"
	"github.com/RodolfoBonis/spooliq/features/customer/domain/repositories"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type csvNoopLogger struct{}

func (csvNoopLogger) Debug(context.Context, string, ...otellogger.Fields)   {}
func (csvNoopLogger) Info(context.Context, string, ...otellogger.Fields)    {}
func (csvNoopLogger) Warning(context.Context, string, ...otellogger.Fields) {}
func (csvNoopLogger) Error(context.Context, string, ...otellogger.Fields)   {}
func (csvNoopLogger) Fatal(context.Context, string, ...otellogger.Fields)   {}
func (csvNoopLogger) Panic(context.Context, string, ...otellogger.Fields)   {}
func (n csvNoopLogger) With(otellogger.Fields) otellogger.Logger            { return n }
func (csvNoopLogger) LogError(context.Context, string, error)               {}

type csvActivity struct{ activityUc.IActivityService }

func (csvActivity) Record(context.Context, activityEntities.ActivityEntity) {}

type csvRepo struct {
	repositories.CustomerRepository
	existing map[string]bool
	created  []*entities.CustomerEntity
}

func (r *csvRepo) ExistsByEmail(_ context.Context, email, _ string, _ *uuid.UUID) (bool, error) {
	return r.existing[email], nil
}

func (r *csvRepo) Create(_ context.Context, c *entities.CustomerEntity) error {
	r.created = append(r.created, c)
	return nil
}

func TestParseCustomerCSV_MapsPortugueseHeaders(t *testing.T) {
	header := []string{"Nome", "E-mail", "Telefone", "CPF", "Endereço", "Cidade", "UF", "CEP", "Observações", "Ignorada"}
	rows, errs, apiErr := ParseCustomerCSV(header, [][]string{
		{"Ana Souza", " Ana@X.com ", "82 99999-0000", "123", "Rua A", "Maceió", "al", "57000-000", "VIP", "x"},
		{"", "", "", "", "", "", "", "", "", ""}, // blank line is ignored
		{"", "sem-nome@x.com"},
	})
	require.Nil(t, apiErr)
	require.Len(t, rows, 1)
	req := rows[0].Request
	assert.Equal(t, 2, rows[0].Line)
	assert.Equal(t, "Ana Souza", req.Name)
	assert.Equal(t, "ana@x.com", *req.Email)
	assert.Equal(t, "AL", *req.State)
	assert.Equal(t, "Rua A", *req.Address)
	assert.Equal(t, "VIP", *req.Notes)

	require.Len(t, errs, 1)
	assert.Equal(t, 4, errs[0].Line)
	assert.Equal(t, "Nome obrigatório", errs[0].Message)
}

func TestParseCustomerCSV_RequiresNameColumn(t *testing.T) {
	_, _, apiErr := ParseCustomerCSV([]string{"email"}, [][]string{{"a@b.c"}})
	require.NotNil(t, apiErr)
	assert.Equal(t, "missing_name_column", apiErr.Code)
}

func TestImportCustomers_SkipsExistingAndDuplicatedEmails(t *testing.T) {
	repo := &csvRepo{existing: map[string]bool{"old@x.com": true}}
	uc := &CustomerUseCase{repository: repo, logger: csvNoopLogger{}, activityService: csvActivity{}}

	rows, errs, apiErr := ParseCustomerCSV([]string{"name", "email"}, [][]string{
		{"Novo", "new@x.com"},
		{"Antigo", "old@x.com"},
		{"Repetido", "new@x.com"},
		{"Sem e-mail", ""},
	})
	require.Nil(t, apiErr)

	result := uc.importCustomers(context.Background(), "org", "u1", rows, errs)

	assert.Equal(t, 2, result.Created)
	assert.Equal(t, 2, result.Skipped)
	assert.Equal(t, 0, result.Failed)
	require.Len(t, repo.created, 2)
	assert.Equal(t, "org", repo.created[0].OrganizationID)
	assert.Equal(t, "u1", repo.created[0].OwnerUserID)
}
