package usecases

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"mime"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/RodolfoBonis/go-otel-agent/logger"
	coreEntities "github.com/RodolfoBonis/spooliq/core/entities"
	"github.com/RodolfoBonis/spooliq/core/services"
	activityEntities "github.com/RodolfoBonis/spooliq/features/activity/domain/entities"
	"github.com/RodolfoBonis/spooliq/features/model3d/domain/entities"
	"github.com/RodolfoBonis/spooliq/features/model3d/domain/repositories"
	slicerservice "github.com/RodolfoBonis/spooliq/features/slicer/domain/service"
	slicersuggest "github.com/RodolfoBonis/spooliq/features/slicer/domain/suggest"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func init() { gin.SetMode(gin.TestMode) }

const (
	testOrg      = "org-1"
	testOtherOrg = "org-2"
)

// fakeModel3DRepo is a controllable in-memory repository for handler tests.
type fakeModel3DRepo struct {
	byID        map[string]*entities.Model3DEntity // key "org|id"
	byHash      *entities.Model3DEntity
	createErr   error
	customers   map[string]bool // key "org|customerID"
	updated     *entities.Model3DEntity
	deletedID   *uuid.UUID
	deletedOrg  string
	listResult  []*entities.Model3DEntity
	listTotal   int64
	lastFilters repositories.Model3DFilters
	lastSearch  string
}

func newFakeRepo() *fakeModel3DRepo {
	return &fakeModel3DRepo{byID: map[string]*entities.Model3DEntity{}, customers: map[string]bool{}}
}

func key(org string, id uuid.UUID) string { return org + "|" + id.String() }

func (f *fakeModel3DRepo) Create(_ context.Context, model *entities.Model3DEntity) error {
	if f.createErr != nil {
		return f.createErr
	}
	if model.ID == uuid.Nil {
		model.ID = uuid.New()
	}
	f.byID[key(model.OrganizationID, model.ID)] = model
	return nil
}

func (f *fakeModel3DRepo) Update(_ context.Context, model *entities.Model3DEntity) error {
	f.updated = model
	f.byID[key(model.OrganizationID, model.ID)] = model
	return nil
}

func (f *fakeModel3DRepo) Delete(_ context.Context, id uuid.UUID, organizationID string) error {
	f.deletedID = &id
	f.deletedOrg = organizationID
	delete(f.byID, key(organizationID, id))
	return nil
}

func (f *fakeModel3DRepo) FindByID(_ context.Context, id uuid.UUID, organizationID string) (*entities.Model3DEntity, error) {
	if m, ok := f.byID[key(organizationID, id)]; ok {
		return m, nil
	}
	return nil, gorm.ErrRecordNotFound
}

func (f *fakeModel3DRepo) FindAll(_ context.Context, _ string, filters repositories.Model3DFilters, search, _ string, _, _ int) ([]*entities.Model3DEntity, int64, error) {
	f.lastFilters = filters
	f.lastSearch = search
	return f.listResult, f.listTotal, nil
}

func (f *fakeModel3DRepo) FindByCustomerID(_ context.Context, _ uuid.UUID, _ string) ([]*entities.Model3DEntity, error) {
	return f.listResult, nil
}

func (f *fakeModel3DRepo) FindByHash(_ context.Context, _ string, _ string) (*entities.Model3DEntity, error) {
	return f.byHash, nil
}

func (f *fakeModel3DRepo) CustomerExists(_ context.Context, customerID uuid.UUID, organizationID string) (bool, error) {
	return f.customers[key(organizationID, customerID)], nil
}

type fakeActivity struct{}

func (fakeActivity) Record(_ context.Context, _ activityEntities.ActivityEntity) {}
func (fakeActivity) ListActivities(_ *gin.Context)                               {}
func (fakeActivity) FindRecentByOrganization(_ string, _ int) ([]activityEntities.ActivityEntity, error) {
	return nil, nil
}

func newUC(repo repositories.Model3DRepository) *Model3DUseCase {
	cdn := services.NewCDNService("", coreEntities.CdnKeysEntity{}, &logger.NoopLogger{})
	thumb := services.NewThumbnailService(&logger.NoopLogger{})
	slicerSvc := slicerservice.NewService(noopCatalog{}, &logger.NoopLogger{})
	return &Model3DUseCase{
		repository:       repo,
		cdnService:       cdn,
		thumbnailService: thumb,
		slicerService:    slicerSvc,
		logger:           &logger.NoopLogger{},
		activityService:  fakeActivity{},
	}
}

// noopCatalog is an empty filament catalog so suggestions resolve to nil in tests.
type noopCatalog struct{}

func (noopCatalog) LoadCandidates(_ context.Context, _ string) ([]slicersuggest.Candidate, error) {
	return nil, nil
}

func ctxWithOrg(org string) (*gin.Context, *httptest.ResponseRecorder) {
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	if org != "" {
		c.Set("organization_id", org)
	}
	c.Set("user_id", "user-1")
	return c, rec
}

func decodeEnvelope(t *testing.T, rec *httptest.ResponseRecorder) map[string]any {
	t.Helper()
	var body map[string]any
	assert.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	return body
}

// --- Upload ---

func multipartUpload(t *testing.T, filename, content string, fields map[string]string) (*http.Request, string) {
	t.Helper()
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	if filename != "" {
		fw, err := w.CreateFormFile("file", filename)
		assert.NoError(t, err)
		_, _ = fw.Write([]byte(content))
	}
	for k, v := range fields {
		_ = w.WriteField(k, v)
	}
	_ = w.Close()
	req := httptest.NewRequest(http.MethodPost, "/models3d", &buf)
	req.Header.Set("Content-Type", w.FormDataContentType())
	return req, w.FormDataContentType()
}

func TestUpload_OrganizationRequired(t *testing.T) {
	uc := newUC(newFakeRepo())
	c, rec := ctxWithOrg("")
	req, _ := multipartUpload(t, "model.stl", "solid x", map[string]string{"name": "A"})
	c.Request = req
	uc.Upload(c)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Equal(t, "organization_required", decodeEnvelope(t, rec)["code"])
}

func TestUpload_FileRequired(t *testing.T) {
	uc := newUC(newFakeRepo())
	c, rec := ctxWithOrg(testOrg)
	req, _ := multipartUpload(t, "", "", map[string]string{"name": "A"})
	c.Request = req
	uc.Upload(c)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Equal(t, "file_required", decodeEnvelope(t, rec)["code"])
}

func TestUpload_UnsupportedFormat(t *testing.T) {
	uc := newUC(newFakeRepo())
	c, rec := ctxWithOrg(testOrg)
	req, _ := multipartUpload(t, "model.obj", "data", map[string]string{"name": "A"})
	c.Request = req
	uc.Upload(c)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Equal(t, "unsupported_file_format", decodeEnvelope(t, rec)["code"])
}

func TestUpload_InvalidCustomerID(t *testing.T) {
	uc := newUC(newFakeRepo())
	c, rec := ctxWithOrg(testOrg)
	req, _ := multipartUpload(t, "model.stl", "data", map[string]string{"name": "A", "customer_id": "not-a-uuid"})
	c.Request = req
	uc.Upload(c)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Equal(t, "invalid_customer_id", decodeEnvelope(t, rec)["code"])
}

func TestUpload_CustomerNotFound(t *testing.T) {
	uc := newUC(newFakeRepo())
	c, rec := ctxWithOrg(testOrg)
	cid := uuid.New()
	req, _ := multipartUpload(t, "model.stl", "data", map[string]string{"name": "A", "customer_id": cid.String()})
	c.Request = req
	uc.Upload(c)
	assert.Equal(t, http.StatusNotFound, rec.Code)
	assert.Equal(t, "customer_not_found", decodeEnvelope(t, rec)["code"])
}

func TestUpload_Duplicate409IncludesExisting(t *testing.T) {
	repo := newFakeRepo()
	existing := &entities.Model3DEntity{ID: uuid.New(), OrganizationID: testOrg, Name: "Existing", FileName: "e.stl"}
	repo.byHash = existing
	uc := newUC(repo)
	c, rec := ctxWithOrg(testOrg)
	req, _ := multipartUpload(t, "model.stl", "same-content", map[string]string{"name": "New"})
	c.Request = req
	uc.Upload(c)

	assert.Equal(t, http.StatusConflict, rec.Code)
	var body entities.DuplicateModel3DResponse
	assert.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	assert.Equal(t, "model3d_duplicate", body.Code)
	assert.NotEmpty(t, body.Message)
	assert.Equal(t, body.Message, body.Error)
	assert.NotNil(t, body.Existing)
	assert.Equal(t, existing.ID, body.Existing.ID)
}

// --- FindByID ---

func TestFindByID_InvalidID(t *testing.T) {
	uc := newUC(newFakeRepo())
	c, rec := ctxWithOrg(testOrg)
	c.Request = httptest.NewRequest(http.MethodGet, "/models3d/bad", nil)
	c.Params = gin.Params{{Key: "id", Value: "bad"}}
	uc.FindByID(c)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Equal(t, "invalid_model3d_id", decodeEnvelope(t, rec)["code"])
}

func TestFindByID_OrgIsolation404(t *testing.T) {
	repo := newFakeRepo()
	m := &entities.Model3DEntity{ID: uuid.New(), OrganizationID: testOtherOrg, Name: "Other"}
	repo.byID[key(testOtherOrg, m.ID)] = m
	uc := newUC(repo)
	c, rec := ctxWithOrg(testOrg) // requesting as org-1
	c.Request = httptest.NewRequest(http.MethodGet, "/models3d/"+m.ID.String(), nil)
	c.Params = gin.Params{{Key: "id", Value: m.ID.String()}}
	uc.FindByID(c)
	assert.Equal(t, http.StatusNotFound, rec.Code)
	assert.Equal(t, "model3d_not_found", decodeEnvelope(t, rec)["code"])
}

// --- FindAll ---

func TestFindAll_InvalidFormat(t *testing.T) {
	uc := newUC(newFakeRepo())
	c, rec := ctxWithOrg(testOrg)
	c.Request = httptest.NewRequest(http.MethodGet, "/models3d?format=.obj", nil)
	uc.FindAll(c)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Equal(t, "invalid_format", decodeEnvelope(t, rec)["code"])
}

func TestFindAll_InvalidCustomerID(t *testing.T) {
	uc := newUC(newFakeRepo())
	c, rec := ctxWithOrg(testOrg)
	c.Request = httptest.NewRequest(http.MethodGet, "/models3d?customer_id=nope", nil)
	uc.FindAll(c)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Equal(t, "invalid_customer_id", decodeEnvelope(t, rec)["code"])
}

func TestFindAll_PageEnvelope(t *testing.T) {
	repo := newFakeRepo()
	repo.listResult = []*entities.Model3DEntity{{ID: uuid.New(), OrganizationID: testOrg, Name: "A"}}
	repo.listTotal = 1
	uc := newUC(repo)
	c, rec := ctxWithOrg(testOrg)
	c.Request = httptest.NewRequest(http.MethodGet, "/models3d?format=.stl&q=abc", nil)
	uc.FindAll(c)
	assert.Equal(t, http.StatusOK, rec.Code)
	body := decodeEnvelope(t, rec)
	assert.Contains(t, body, "data")
	assert.EqualValues(t, 1, body["total"])
	assert.Equal(t, ".stl", repo.lastFilters.Format)
	assert.Equal(t, "abc", repo.lastSearch)
}

// --- Update ---

func TestUpdate_ClearsCustomerID(t *testing.T) {
	repo := newFakeRepo()
	cid := uuid.New()
	m := &entities.Model3DEntity{ID: uuid.New(), OrganizationID: testOrg, Name: "A", CustomerID: &cid}
	repo.byID[key(testOrg, m.ID)] = m
	uc := newUC(repo)
	c, rec := ctxWithOrg(testOrg)
	c.Request = httptest.NewRequest(http.MethodPut, "/models3d/"+m.ID.String(), strings.NewReader(`{"customer_id": null}`))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Params = gin.Params{{Key: "id", Value: m.ID.String()}}
	uc.Update(c)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.NotNil(t, repo.updated)
	assert.Nil(t, repo.updated.CustomerID, "explicit null should detach the customer")
}

func TestUpdate_KeepsCustomerIDWhenAbsent(t *testing.T) {
	repo := newFakeRepo()
	cid := uuid.New()
	m := &entities.Model3DEntity{ID: uuid.New(), OrganizationID: testOrg, Name: "A", CustomerID: &cid}
	repo.byID[key(testOrg, m.ID)] = m
	uc := newUC(repo)
	c, rec := ctxWithOrg(testOrg)
	c.Request = httptest.NewRequest(http.MethodPut, "/models3d/"+m.ID.String(), strings.NewReader(`{"name":"B"}`))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Params = gin.Params{{Key: "id", Value: m.ID.String()}}
	uc.Update(c)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.NotNil(t, repo.updated)
	assert.NotNil(t, repo.updated.CustomerID)
	assert.Equal(t, cid, *repo.updated.CustomerID)
	assert.Equal(t, "B", repo.updated.Name)
}

func TestUpdate_OrgIsolation404(t *testing.T) {
	repo := newFakeRepo()
	m := &entities.Model3DEntity{ID: uuid.New(), OrganizationID: testOtherOrg, Name: "Other"}
	repo.byID[key(testOtherOrg, m.ID)] = m
	uc := newUC(repo)
	c, rec := ctxWithOrg(testOrg)
	c.Request = httptest.NewRequest(http.MethodPut, "/models3d/"+m.ID.String(), strings.NewReader(`{"name":"B"}`))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Params = gin.Params{{Key: "id", Value: m.ID.String()}}
	uc.Update(c)
	assert.Equal(t, http.StatusNotFound, rec.Code)
	assert.Equal(t, "model3d_not_found", decodeEnvelope(t, rec)["code"])
}

// --- Delete ---

func TestDelete_OrgIsolation404(t *testing.T) {
	repo := newFakeRepo()
	m := &entities.Model3DEntity{ID: uuid.New(), OrganizationID: testOtherOrg, Name: "Other"}
	repo.byID[key(testOtherOrg, m.ID)] = m
	uc := newUC(repo)
	c, rec := ctxWithOrg(testOrg)
	c.Request = httptest.NewRequest(http.MethodDelete, "/models3d/"+m.ID.String(), nil)
	c.Params = gin.Params{{Key: "id", Value: m.ID.String()}}
	uc.Delete(c)
	assert.Equal(t, http.StatusNotFound, rec.Code)
	assert.Nil(t, repo.deletedID, "must not delete another org's model")
}

func TestDelete_SoftDeletesScoped(t *testing.T) {
	repo := newFakeRepo()
	m := &entities.Model3DEntity{ID: uuid.New(), OrganizationID: testOrg, Name: "A", FileURL: "https://cdn/x.stl"}
	repo.byID[key(testOrg, m.ID)] = m
	uc := newUC(repo)
	c, _ := ctxWithOrg(testOrg)
	c.Request = httptest.NewRequest(http.MethodDelete, "/models3d/"+m.ID.String(), nil)
	c.Params = gin.Params{{Key: "id", Value: m.ID.String()}}
	uc.Delete(c)
	// 204 has no body, so gin may not flush the status to the recorder in a
	// direct handler call; assert on the gin writer status instead.
	assert.Equal(t, http.StatusNoContent, c.Writer.Status())
	assert.NotNil(t, repo.deletedID)
	assert.Equal(t, m.ID, *repo.deletedID)
	assert.Equal(t, testOrg, repo.deletedOrg)
}

// --- StreamFile ---

func TestStreamFile_InvalidID(t *testing.T) {
	uc := newUC(newFakeRepo())
	c, rec := ctxWithOrg(testOrg)
	c.Request = httptest.NewRequest(http.MethodGet, "/models3d/bad/file", nil)
	c.Params = gin.Params{{Key: "id", Value: "bad"}}
	uc.StreamFile(c)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Equal(t, "invalid_model3d_id", decodeEnvelope(t, rec)["code"])
}

func TestStreamFile_OrgIsolation404(t *testing.T) {
	repo := newFakeRepo()
	m := &entities.Model3DEntity{ID: uuid.New(), OrganizationID: testOtherOrg, Name: "Other", FileURL: "https://cdn/x.stl"}
	repo.byID[key(testOtherOrg, m.ID)] = m
	uc := newUC(repo)
	c, rec := ctxWithOrg(testOrg)
	c.Request = httptest.NewRequest(http.MethodGet, "/models3d/"+m.ID.String()+"/file", nil)
	c.Params = gin.Params{{Key: "id", Value: m.ID.String()}}
	uc.StreamFile(c)
	assert.Equal(t, http.StatusNotFound, rec.Code)
	assert.Equal(t, "model3d_not_found", decodeEnvelope(t, rec)["code"])
}

// --- Oversized upload (413) ---

func TestUpload_FileTooLarge(t *testing.T) {
	orig := uploadBodyLimit
	uploadBodyLimit = 1024 // shrink so a small fixture exceeds it
	defer func() { uploadBodyLimit = orig }()

	uc := newUC(newFakeRepo())
	c, rec := ctxWithOrg(testOrg)
	// A file comfortably larger than the shrunk body limit.
	big := strings.Repeat("A", 8*1024)
	req, _ := multipartUpload(t, "model.stl", big, map[string]string{"name": "Big"})
	c.Request = req
	uc.Upload(c)

	assert.Equal(t, http.StatusRequestEntityTooLarge, rec.Code)
	assert.Equal(t, "file_too_large", decodeEnvelope(t, rec)["code"])
}

// --- Filename sanitization & Content-Disposition ---

func TestSanitizeFileName(t *testing.T) {
	assert.Equal(t, "model.stl", sanitizeFileName(`mo"del.stl`))
	assert.Equal(t, "evil.stl", sanitizeFileName(`../../evil.stl`))
	assert.Equal(t, "ab.stl", sanitizeFileName("a\\b.stl"))
	assert.Equal(t, "linebreak.stl", sanitizeFileName("line\r\nbreak.stl"))
	assert.Equal(t, "model", sanitizeFileName(""))
	assert.Equal(t, "model", sanitizeFileName(".."))
	assert.Equal(t, "cão.3mf", sanitizeFileName("cão.3mf")) // non-ASCII kept
}

func TestContentDisposition_QuotesAndUnicode(t *testing.T) {
	// A name with a double quote must be safely quoted/escaped, never breaking the header.
	d := contentDisposition(`a"b.stl`)
	assert.Contains(t, d, "inline")
	assert.NotContains(t, d, "filename=\"a\"b.stl\"", "raw unescaped quote must not appear")

	// Parsing it back yields the original filename (proves it is well-formed).
	_, params, err := mime.ParseMediaType(d)
	require.NoError(t, err)
	assert.Equal(t, `a"b.stl`, params["filename"])
}

func TestIsRequestTooLarge(t *testing.T) {
	assert.True(t, isRequestTooLarge(&http.MaxBytesError{Limit: 10}))
	assert.True(t, isRequestTooLarge(errors.New("http: request body too large")))
	assert.False(t, isRequestTooLarge(errors.New("some other error")))
	assert.False(t, isRequestTooLarge(nil))
}
