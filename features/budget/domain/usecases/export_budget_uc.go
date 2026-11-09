package usecases

import (
	"strconv"
	"time"

	coreErrors "github.com/RodolfoBonis/spooliq/core/errors"
	"github.com/RodolfoBonis/spooliq/core/helpers"
	"github.com/RodolfoBonis/spooliq/features/budget/domain/entities"
	"github.com/gin-gonic/gin"
)

// maxExportRows caps a CSV export so one request can't scan everything.
const maxExportRows = 5000

// budgetStatusLabels are the pt-BR labels used in exports.
var budgetStatusLabels = map[entities.BudgetStatus]string{
	entities.StatusDraft:     "Rascunho",
	entities.StatusSent:      "Enviado",
	entities.StatusApproved:  "Aprovado",
	entities.StatusRejected:  "Rejeitado",
	entities.StatusPrinting:  "Imprimindo",
	entities.StatusCompleted: "Concluído",
	entities.StatusExpired:   "Expirado",
	entities.StatusCancelled: "Cancelado",
}

// ExportCSV exports the budgets matching the list filters as CSV.
// @Summary Export budgets (CSV)
// @Description Same filters and sorting as GET /budgets (q, status, customer_id, from, to), up to 5000 rows. UTF-8 with BOM, ';' separator.
// @Tags budgets
// @Produce text/csv
// @Security BearerAuth
// @Param q query string false "Search (name or quote number)"
// @Param status query string false "Status filter (comma-separated)"
// @Param customer_id query string false "Customer ID"
// @Param from query string false "Created from (YYYY-MM-DD)"
// @Param to query string false "Created until (YYYY-MM-DD)"
// @Success 200 {file} file "CSV"
// @Failure 400 {object} errors.HTTPError
// @Router /budgets/export.csv [get]
func (uc *BudgetUseCase) ExportCSV(c *gin.Context) {
	ctx := c.Request.Context()
	organizationID := helpers.GetOrganizationID(c)
	if organizationID == "" {
		coreErrors.Respond(c, coreErrors.BadRequest(CodeOrganizationRequired, "Organização não identificada"))
		return
	}

	listQuery := budgetListQuery(c)
	filters, apiErr := parseBudgetFilters(c, listQuery.Search)
	if apiErr != nil {
		coreErrors.Respond(c, apiErr)
		return
	}

	budgets, _, err := uc.budgetRepository.SearchBudgets(ctx, organizationID, filters, listQuery.OrderClause(), maxExportRows, 0)
	if err != nil {
		uc.logger.Error(ctx, "Failed to export budgets", map[string]interface{}{"error": err.Error()})
		respondBudgetError(c, err)
		return
	}
	responses, err := uc.buildBudgetListResponses(ctx, budgets, organizationID)
	if err != nil {
		respondBudgetError(c, err)
		return
	}

	rows := make([][]string, 0, len(responses))
	for _, b := range responses {
		rows = append(rows, budgetCSVRow(b))
	}
	helpers.WriteCSV(c, "orcamentos-"+time.Now().Format("2006-01-02")+".csv", budgetCSVHeader, rows)
}

var budgetCSVHeader = []string{
	"Número", "Nome", "Cliente", "Status", "Total (R$)", "Desconto (R$)", "Frete (R$)",
	"Impostos (R$)", "Tempo de impressão", "Criado em", "Válido até",
}

func budgetCSVRow(b entities.BudgetResponse) []string {
	quote := ""
	if b.QuoteNumber != nil {
		quote = strconv.Itoa(*b.QuoteNumber)
	}
	customer := ""
	if b.Customer != nil {
		customer = b.Customer.Name
	}
	status, ok := budgetStatusLabels[b.Status]
	if !ok {
		status = string(b.Status)
	}
	created := b.CreatedAt
	return []string{
		quote,
		b.Name,
		customer,
		status,
		helpers.CSVCents(b.TotalCost),
		helpers.CSVCents(b.DiscountAmount),
		helpers.CSVCents(b.ShippingCost),
		helpers.CSVCents(b.TaxAmount),
		b.TotalPrintTimeDisplay,
		helpers.CSVDate(&created),
		helpers.CSVDate(b.ValidUntil),
	}
}
