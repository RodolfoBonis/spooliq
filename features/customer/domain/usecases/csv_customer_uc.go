package usecases

import (
	"context"
	"net/http"
	"strconv"
	"strings"
	"time"

	"unicode"

	coreErrors "github.com/RodolfoBonis/spooliq/core/errors"
	"github.com/RodolfoBonis/spooliq/core/helpers"
	"github.com/RodolfoBonis/spooliq/core/validation"
	activityEntities "github.com/RodolfoBonis/spooliq/features/activity/domain/entities"
	"github.com/RodolfoBonis/spooliq/features/customer/domain/entities"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"golang.org/x/text/runes"
	"golang.org/x/text/transform"
	"golang.org/x/text/unicode/norm"
)

const (
	maxCustomerExportRows = 5000
	maxCustomerImportRows = 2000
	maxCustomerImportSize = 2 << 20 // 2 MB
)

var customerCSVHeader = []string{
	"Nome", "E-mail", "Telefone", "Documento", "Endereço", "Cidade", "UF", "CEP",
	"Observações", "Orçamentos", "Total (R$)", "Ativo", "Cliente desde",
}

// ExportCSV exports the organization's customers as CSV.
// @Summary Export customers (CSV)
// @Description Up to 5000 customers (optionally filtered by q), with budget count and total. UTF-8 with BOM, ';' separator. The file can be re-imported.
// @Tags customers
// @Produce text/csv
// @Security BearerAuth
// @Param q query string false "Search"
// @Success 200 {file} file "CSV"
// @Router /customers/export.csv [get]
func (uc *CustomerUseCase) ExportCSV(c *gin.Context) {
	ctx := c.Request.Context()
	organizationID := helpers.GetOrganizationID(c)
	if organizationID == "" {
		coreErrors.Respond(c, coreErrors.BadRequest("organization_required", "Organização não encontrada no contexto"))
		return
	}
	q := helpers.ParseListQuery(c, helpers.ListQueryOptions{
		DefaultPageSize: 20,
		SortWhitelist:   customerSortWhitelist,
		DefaultSort:     "name",
		TieBreaker:      "id",
	})
	customers, _, err := uc.repository.FindAll(ctx, organizationID, q.Search, q.OrderClause(), maxCustomerExportRows, 0)
	if err != nil {
		uc.logger.Error(ctx, "Failed to export customers", map[string]interface{}{"error": err.Error()})
		coreErrors.Respond(c, err)
		return
	}
	rows := make([][]string, 0, len(customers))
	for _, r := range uc.buildCustomerResponses(ctx, customers) {
		rows = append(rows, customerCSVRow(r))
	}
	helpers.WriteCSV(c, "clientes-"+time.Now().Format("2006-01-02")+".csv", customerCSVHeader, rows)
}

func customerCSVRow(r entities.CustomerResponse) []string {
	c := r.Customer
	total := int64(0)
	if r.TotalBudgets != nil {
		total = *r.TotalBudgets
	}
	active := "Sim"
	if !c.IsActive {
		active = "Não"
	}
	created := c.CreatedAt
	return []string{
		c.Name, str(c.Email), str(c.Phone), str(c.Document), str(c.Address),
		str(c.City), str(c.State), str(c.ZipCode), str(c.Notes),
		strconv.Itoa(r.BudgetCount), helpers.CSVCents(total), active, helpers.CSVDate(&created),
	}
}

func str(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}

// ImportRowResult reports what happened to one CSV line.
type ImportRowResult struct {
	Line    int    `json:"line"`
	Name    string `json:"name,omitempty"`
	Status  string `json:"status"` // created | skipped | error
	Message string `json:"message,omitempty"`
}

// ImportResult summarizes a customer import.
type ImportResult struct {
	Created int               `json:"created"`
	Skipped int               `json:"skipped"`
	Failed  int               `json:"failed"`
	Rows    []ImportRowResult `json:"rows"`
}

// ImportCSV creates customers from a CSV file (multipart field "file").
// @Summary Import customers (CSV)
// @Description Columns are matched by header name (pt-BR or English, case/accent-insensitive): nome/name (required), email, telefone/phone, documento/cpf/cnpj, endereço/address, cidade/city, uf/estado/state, cep/zip, observações/notas/notes. Rows whose e-mail already exists are skipped. Max 2000 rows / 2 MB. Accepts ';' or ',' (the export format re-imports as is).
// @Tags customers
// @Accept multipart/form-data
// @Produce json
// @Security BearerAuth
// @Param file formData file true "CSV file"
// @Success 200 {object} ImportResult
// @Failure 400 {object} errors.HTTPError
// @Router /customers/import [post]
func (uc *CustomerUseCase) ImportCSV(c *gin.Context) {
	ctx := c.Request.Context()
	organizationID := helpers.GetOrganizationID(c)
	userID := helpers.GetUserID(c)
	if organizationID == "" || userID == "" {
		coreErrors.Respond(c, coreErrors.BadRequest("organization_required", "Organização não encontrada no contexto"))
		return
	}

	fileHeader, err := c.FormFile("file")
	if err != nil {
		coreErrors.Respond(c, coreErrors.BadRequest("file_required", "Envie o arquivo CSV no campo \"file\""))
		return
	}
	if fileHeader.Size > maxCustomerImportSize {
		coreErrors.Respond(c, coreErrors.BadRequest("file_too_large", "O arquivo deve ter no máximo 2 MB"))
		return
	}
	file, err := fileHeader.Open()
	if err != nil {
		coreErrors.Respond(c, coreErrors.BadRequest("invalid_file", "Não foi possível ler o arquivo"))
		return
	}
	defer file.Close()

	header, records, err := helpers.ReadCSV(file)
	if err != nil {
		coreErrors.Respond(c, coreErrors.BadRequest("invalid_csv", "CSV inválido: "+err.Error()))
		return
	}
	requests, rowErrors, apiErr := ParseCustomerCSV(header, records)
	if apiErr != nil {
		coreErrors.Respond(c, apiErr)
		return
	}

	result := uc.importCustomers(ctx, organizationID, userID, requests, rowErrors)
	uc.logger.Info(ctx, "Customer import finished", map[string]interface{}{
		"organization_id": organizationID,
		"created":         result.Created,
		"skipped":         result.Skipped,
		"failed":          result.Failed,
	})
	c.JSON(http.StatusOK, result)
}

// CustomerCSVRow is a parsed CSV line ready to be created.
type CustomerCSVRow struct {
	Line    int
	Request entities.CreateCustomerRequest
}

// ParseCustomerCSV maps the header to customer fields and validates each row.
// Lines are 1-based and count the header (line 2 is the first data row).
func ParseCustomerCSV(header []string, records [][]string) ([]CustomerCSVRow, []ImportRowResult, *coreErrors.APIError) {
	columns := map[string]int{}
	for i, h := range header {
		if field, ok := customerColumnAliases[normalizeHeader(h)]; ok {
			if _, dup := columns[field]; !dup {
				columns[field] = i
			}
		}
	}
	if _, ok := columns["name"]; !ok {
		return nil, nil, coreErrors.BadRequest("missing_name_column", "O CSV precisa de uma coluna \"Nome\"")
	}
	if len(records) > maxCustomerImportRows {
		return nil, nil, coreErrors.BadRequest("too_many_rows", "Importe no máximo 2000 clientes por vez")
	}

	var rows []CustomerCSVRow
	var errs []ImportRowResult
	for i, rec := range records {
		line := i + 2
		get := func(field string) *string {
			idx, ok := columns[field]
			if !ok || idx >= len(rec) {
				return nil
			}
			v := strings.TrimSpace(rec[idx])
			if v == "" {
				return nil
			}
			return &v
		}
		if isBlank(rec) {
			continue
		}
		req := entities.CreateCustomerRequest{
			Name:     str(get("name")),
			Email:    lower(get("email")),
			Phone:    get("phone"),
			Document: get("document"),
			Address:  get("address"),
			City:     get("city"),
			State:    upper(get("state")),
			ZipCode:  get("zip_code"),
			Notes:    get("notes"),
		}
		if err := validation.Validate(req); err != nil {
			errs = append(errs, ImportRowResult{Line: line, Name: req.Name, Status: "error", Message: validationMessage(req)})
			continue
		}
		rows = append(rows, CustomerCSVRow{Line: line, Request: req})
	}
	return rows, errs, nil
}

func (uc *CustomerUseCase) importCustomers(ctx context.Context, organizationID, userID string, rows []CustomerCSVRow, rowErrors []ImportRowResult) ImportResult {
	result := ImportResult{Rows: append([]ImportRowResult{}, rowErrors...)}
	result.Failed = len(rowErrors)
	seenEmails := map[string]bool{}

	for _, row := range rows {
		req := row.Request
		res := ImportRowResult{Line: row.Line, Name: req.Name}
		if req.Email != nil {
			email := *req.Email
			exists := seenEmails[email]
			if !exists {
				var err error
				exists, err = uc.repository.ExistsByEmail(ctx, email, organizationID, nil)
				if err != nil {
					res.Status, res.Message = "error", "Falha ao verificar o e-mail"
					result.Failed++
					result.Rows = append(result.Rows, res)
					continue
				}
			}
			if exists {
				res.Status, res.Message = "skipped", "Já existe um cliente com este e-mail"
				result.Skipped++
				result.Rows = append(result.Rows, res)
				continue
			}
			seenEmails[email] = true
		}

		now := time.Now()
		customer := &entities.CustomerEntity{
			ID: uuid.New(), OrganizationID: organizationID, Name: req.Name,
			Email: req.Email, Phone: req.Phone, Document: req.Document, Address: req.Address,
			City: req.City, State: req.State, ZipCode: req.ZipCode, Notes: req.Notes,
			OwnerUserID: userID, IsActive: true, CreatedAt: now, UpdatedAt: now,
		}
		if err := uc.repository.Create(ctx, customer); err != nil {
			uc.logger.Error(ctx, "Failed to import customer", map[string]interface{}{"error": err.Error(), "line": row.Line})
			res.Status, res.Message = "error", "Falha ao salvar o cliente"
			result.Failed++
			result.Rows = append(result.Rows, res)
			continue
		}
		res.Status = "created"
		result.Created++
		result.Rows = append(result.Rows, res)
		uc.activityService.Record(ctx, activityEntities.ActivityEntity{
			OrganizationID: organizationID,
			UserID:         userID,
			Action:         activityEntities.ActionCreated,
			EntityType:     activityEntities.EntityCustomer,
			EntityID:       customer.ID.String(),
			EntityName:     customer.Name,
			Description:    "Customer imported from CSV: " + customer.Name,
		})
	}
	return result
}

// customerColumnAliases maps normalized header names to customer fields.
var customerColumnAliases = map[string]string{
	"nome": "name", "name": "name", "cliente": "name",
	"email": "email", "e-mail": "email",
	"telefone": "phone", "phone": "phone", "celular": "phone", "whatsapp": "phone",
	"documento": "document", "document": "document", "cpf": "document", "cnpj": "document", "cpf/cnpj": "document",
	"endereco": "address", "address": "address",
	"cidade": "city", "city": "city",
	"uf": "state", "estado": "state", "state": "state",
	"cep": "zip_code", "zip": "zip_code", "zip_code": "zip_code", "zipcode": "zip_code",
	"observacoes": "notes", "observacao": "notes", "notas": "notes", "notes": "notes",
}

// normalizeHeader lowercases and strips accents ("Endereço" -> "endereco").
func normalizeHeader(h string) string {
	t := transform.Chain(norm.NFD, runes.Remove(runes.In(unicode.Mn)), norm.NFC)
	out, _, err := transform.String(t, strings.ToLower(strings.TrimSpace(h)))
	if err != nil {
		return strings.ToLower(strings.TrimSpace(h))
	}
	return out
}

func validationMessage(req entities.CreateCustomerRequest) string {
	switch {
	case strings.TrimSpace(req.Name) == "":
		return "Nome obrigatório"
	case req.Email != nil && !strings.Contains(*req.Email, "@"):
		return "E-mail inválido"
	default:
		return "Dados inválidos (verifique os tamanhos dos campos)"
	}
}

func isBlank(rec []string) bool {
	for _, v := range rec {
		if strings.TrimSpace(v) != "" {
			return false
		}
	}
	return true
}

func lower(p *string) *string {
	if p == nil {
		return nil
	}
	v := strings.ToLower(*p)
	return &v
}

func upper(p *string) *string {
	if p == nil {
		return nil
	}
	v := strings.ToUpper(*p)
	return &v
}
