package middlewares

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestRecoveryReturnsEnvelopeAndDoesNotLeak(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(Recovery(nil))
	router.GET("/boom", func(_ *gin.Context) {
		panic("secret internal detail")
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/boom", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", w.Code)
	}
	body := w.Body.String()
	if strings.Contains(body, "secret internal detail") {
		t.Fatalf("panic detail leaked: %s", body)
	}

	var envelope map[string]any
	if err := json.Unmarshal([]byte(body), &envelope); err != nil {
		t.Fatalf("body is not JSON: %v", err)
	}
	if envelope["code"] != "internal_error" {
		t.Errorf("code = %v, want internal_error", envelope["code"])
	}
	if envelope["error"] == nil || envelope["message"] == nil {
		t.Errorf("envelope must contain error and message: %v", envelope)
	}
}
