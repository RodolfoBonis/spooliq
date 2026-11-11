package usecases

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/RodolfoBonis/spooliq/features/budget/domain/entities"
	budgetRepo "github.com/RodolfoBonis/spooliq/features/budget/domain/repositories"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// shareFake is a stateful BudgetRepository fake for the share tests. It embeds the
// canned fakeBudgetRepo and overrides the few methods the share flow exercises,
// mutating a single stored budget so re-reads reflect prior writes.
type shareFake struct {
	*fakeBudgetRepo
	stored       *entities.BudgetEntity
	validityDays int
	setTokenN    int
	setValidN    int
	historyN     int
}

func (s *shareFake) WithTransaction(_ context.Context, fn func(repo budgetRepo.BudgetRepository) error) error {
	return fn(s)
}

func (s *shareFake) FindByID(_ context.Context, _ uuid.UUID, _ string) (*entities.BudgetEntity, error) {
	cp := *s.stored
	return &cp, nil
}

func (s *shareFake) UpdateStatus(_ context.Context, _ uuid.UUID, _ string, _, newStatus entities.BudgetStatus) error {
	s.stored.Status = newStatus
	return nil
}

func (s *shareFake) SetValidUntil(_ context.Context, _ uuid.UUID, _ string, vu *time.Time) error {
	s.setValidN++
	s.stored.ValidUntil = vu
	return nil
}

func (s *shareFake) SetShareToken(_ context.Context, _ uuid.UUID, _ string, token string, ts time.Time) error {
	s.setTokenN++
	s.stored.PublicToken = &token
	t := ts
	s.stored.PublicTokenCreatedAt = &t
	return nil
}

func (s *shareFake) RevokeShareToken(_ context.Context, _ uuid.UUID, _ string) error {
	s.stored.PublicToken = nil
	s.stored.PublicTokenCreatedAt = nil
	return nil
}

func (s *shareFake) AddStatusHistory(_ context.Context, _ *entities.BudgetStatusHistoryEntity) error {
	s.historyN++
	return nil
}

func (s *shareFake) GetCompanyQuoteDefaults(_ context.Context, _ string) (int, *string, error) {
	days := s.validityDays
	if days == 0 {
		days = 15
	}
	return days, nil, nil
}

func newShareFake(b *entities.BudgetEntity) *shareFake {
	return &shareFake{fakeBudgetRepo: &fakeBudgetRepo{}, stored: b, validityDays: 15}
}

func callShare(t *testing.T, repo *shareFake, id uuid.UUID) shareResponse {
	t.Helper()
	uc := newUseCaseWith(repo)
	c, w := newJSONContext(http.MethodPost, "/v1/budgets/"+id.String()+"/share", "", gin.Params{{Key: "id", Value: id.String()}})
	uc.Share(c)
	if w.Code != http.StatusOK {
		t.Fatalf("share: got status %d, body %s", w.Code, w.Body.String())
	}
	var resp shareResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode share response: %v", err)
	}
	return resp
}

func TestShare_DraftTransitionsToSentWithValidUntil(t *testing.T) {
	id := uuid.New()
	repo := newShareFake(&entities.BudgetEntity{ID: id, OrganizationID: "org-a", Status: entities.StatusDraft})

	resp := callShare(t, repo, id)

	if resp.Status != string(entities.StatusSent) {
		t.Errorf("status: got %q want sent", resp.Status)
	}
	if resp.PublicToken == "" {
		t.Error("expected a public token to be generated")
	}
	if resp.ValidUntil == nil {
		t.Error("expected valid_until to be set on draft->sent")
	}
	if repo.historyN != 1 {
		t.Errorf("expected 1 status-history row, got %d", repo.historyN)
	}
}

func TestShare_IdempotentToken(t *testing.T) {
	id := uuid.New()
	repo := newShareFake(&entities.BudgetEntity{ID: id, OrganizationID: "org-a", Status: entities.StatusDraft})

	first := callShare(t, repo, id)
	second := callShare(t, repo, id)

	if first.PublicToken != second.PublicToken {
		t.Errorf("token not idempotent: %q != %q", first.PublicToken, second.PublicToken)
	}
	if repo.setTokenN != 1 {
		t.Errorf("token should be written exactly once, got %d writes", repo.setTokenN)
	}
}

func TestShare_CancelledReturns409(t *testing.T) {
	id := uuid.New()
	repo := newShareFake(&entities.BudgetEntity{ID: id, OrganizationID: "org-a", Status: entities.StatusCancelled})
	uc := newUseCaseWith(repo)

	c, w := newJSONContext(http.MethodPost, "/v1/budgets/"+id.String()+"/share", "", gin.Params{{Key: "id", Value: id.String()}})
	uc.Share(c)

	if w.Code != http.StatusConflict {
		t.Fatalf("got status %d want 409, body %s", w.Code, w.Body.String())
	}
	var body map[string]any
	_ = json.Unmarshal(w.Body.Bytes(), &body)
	if body["code"] != CodeBudgetNotShareable {
		t.Errorf("got code %v want %s", body["code"], CodeBudgetNotShareable)
	}
}

func TestDuplicate_ResetsPublicFields(t *testing.T) {
	id := uuid.New()
	token := "SOME_TOKEN"
	past := time.Now().Add(-time.Hour)
	name := "Alice"
	original := &entities.BudgetEntity{
		ID:                   id,
		OrganizationID:       "org-a",
		Name:                 "Orig",
		Status:               entities.StatusApproved,
		QuoteNumber:          intPtr(7),
		PublicToken:          &token,
		ValidUntil:           &past,
		CustomerResponseAt:   &past,
		CustomerResponseName: &name,
		RejectionReason:      &name,
	}
	repo := &fakeBudgetRepo{budget: original}
	uc := newUseCaseWith(repo)

	c, w := newJSONContext(http.MethodPost, "/v1/budgets/"+id.String()+"/duplicate", "", gin.Params{{Key: "id", Value: id.String()}})
	uc.Duplicate(c)

	if w.Code != http.StatusCreated {
		t.Fatalf("duplicate: got status %d, body %s", w.Code, w.Body.String())
	}
	created := repo.lastCreated
	if created == nil {
		t.Fatal("expected a new budget to be created")
	}
	if created.Status != entities.StatusDraft {
		t.Errorf("status: got %q want draft", created.Status)
	}
	if created.PublicToken != nil {
		t.Error("public_token should be reset to nil")
	}
	if created.ValidUntil != nil {
		t.Error("valid_until should be reset to nil")
	}
	if created.CustomerResponseAt != nil || created.CustomerResponseName != nil || created.RejectionReason != nil {
		t.Error("customer response fields should be reset to nil")
	}
	if created.QuoteNumber == nil || *created.QuoteNumber != 1 {
		t.Errorf("expected a freshly allocated quote number (1 from fake), got %v", created.QuoteNumber)
	}
}

func intPtr(n int) *int { return &n }
