package usecases

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func newCtxWithHeaders(remoteAddr string, headers map[string]string) *gin.Context {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	req := httptest.NewRequest(http.MethodGet, "/v1/public/budgets/tok", nil)
	if remoteAddr != "" {
		req.RemoteAddr = remoteAddr
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	c.Request = req
	return c
}

func TestPublicClientIP(t *testing.T) {
	t.Run("prefers CF-Connecting-IP", func(t *testing.T) {
		c := newCtxWithHeaders("10.0.0.1:1111", map[string]string{
			"CF-Connecting-IP": "203.0.113.7",
			"X-Forwarded-For":  "198.51.100.9, 10.0.0.1",
		})
		if got := publicClientIP(c); got != "203.0.113.7" {
			t.Errorf("got %q want 203.0.113.7", got)
		}
	})

	t.Run("falls back to first valid XFF", func(t *testing.T) {
		c := newCtxWithHeaders("10.0.0.1:1111", map[string]string{
			"X-Forwarded-For": "198.51.100.9, 10.0.0.1",
		})
		if got := publicClientIP(c); got != "198.51.100.9" {
			t.Errorf("got %q want 198.51.100.9", got)
		}
	})

	t.Run("skips invalid XFF entries", func(t *testing.T) {
		c := newCtxWithHeaders("10.0.0.1:1111", map[string]string{
			"X-Forwarded-For": "garbage, 198.51.100.42",
		})
		if got := publicClientIP(c); got != "198.51.100.42" {
			t.Errorf("got %q want 198.51.100.42", got)
		}
	})

	t.Run("ignores invalid CF header and uses XFF", func(t *testing.T) {
		c := newCtxWithHeaders("10.0.0.1:1111", map[string]string{
			"CF-Connecting-IP": "not-an-ip",
			"X-Forwarded-For":  "198.51.100.9",
		})
		if got := publicClientIP(c); got != "198.51.100.9" {
			t.Errorf("got %q want 198.51.100.9", got)
		}
	})
}
