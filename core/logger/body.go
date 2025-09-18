package logger

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
)

var sensitiveFields = []string{"password", "token", "secret"}

func isDevelopment() bool {
	return os.Getenv("ENV") == "development" || os.Getenv("ENV") == "dev"
}

func maskSensitiveFields(body string) string {
	var data map[string]interface{}
	if err := json.Unmarshal([]byte(body), &data); err == nil {
		for _, field := range sensitiveFields {
			if _, ok := data[field]; ok {
				data[field] = "***MASKED***"
			}
		}
		if masked, err := json.Marshal(data); err == nil {
			return string(masked)
		}
	}
	return body
}

// HandleRequestBody processes the request body for logging.
func HandleRequestBody(req *http.Request) string {
	if !isDevelopment() {
		return ""
	}
	var requestBodyBytes []byte
	if req.Body == nil {
		return ""
	}

	requestBodyBytes, _ = io.ReadAll(req.Body)
	if len(requestBodyBytes) > 2048 {
		requestBodyBytes = requestBodyBytes[:2048]
	}
	req.Body = io.NopCloser(bytes.NewBuffer(requestBodyBytes))
	return maskSensitiveFields(string(requestBodyBytes))
}

// HandleResponseBody processes the response body for logging.
func HandleResponseBody(rw gin.ResponseWriter) *BodyLogWriter {
	return &BodyLogWriter{Body: bytes.NewBufferString(""), ResponseWriter: rw}
}

// FormatRequestAndResponse formats the request and response for logging.
func FormatRequestAndResponse(rw gin.ResponseWriter, req *http.Request, responseBody string, requestID string, requestBody string) string {
	if req.URL.String() == "/metrics" || strings.Contains(req.URL.String(), "/docs") {
		return ""
	}
	if !isDevelopment() {
		return ""
	}
	if len(requestBody) > 2048 {
		requestBody = requestBody[:2048] + "..."
	}
	if len(responseBody) > 2048 {
		responseBody = responseBody[:2048] + "..."
	}
	requestBody = maskSensitiveFields(requestBody)
	responseBody = maskSensitiveFields(responseBody)
	return fmt.Sprintf("[Request ID: %s], Status: [%d], Method: [%s], Url: %s Request Body: %s, Response Body: %s",
		requestID, rw.Status(), req.Method, req.URL.String(), requestBody, responseBody)
}
