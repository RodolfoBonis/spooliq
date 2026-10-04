package middlewares

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

// TestResponseWriterWrite tests that the custom responseWriter properly writes data to the client
func TestResponseWriterWrite(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("Write should propagate data to underlying ResponseWriter", func(t *testing.T) {
		// Create a test recorder (simulates the actual HTTP response)
		recorder := httptest.NewRecorder()

		// Create a gin context with the recorder
		c, _ := gin.CreateTestContext(recorder)

		// Create our custom responseWriter wrapping the recorder
		writer := &responseWriter{
			ResponseWriter: c.Writer,
			body:           make([]byte, 0),
			statusCode:     http.StatusOK,
		}

		// Write test data
		testData := []byte(`{"message": "test data"}`)
		n, err := writer.Write(testData)

		if err != nil {
			t.Errorf("Write returned error: %v", err)
		}

		if n != len(testData) {
			t.Errorf("Write returned wrong byte count: got %d, want %d", n, len(testData))
		}

		// Check that data was accumulated in body buffer (for caching)
		if !bytes.Equal(writer.body, testData) {
			t.Errorf("Data not accumulated in body buffer: got %s, want %s", writer.body, testData)
		}

		// THIS IS THE BUG CHECK: verify data was written to the underlying ResponseWriter
		recorderBody := recorder.Body.Bytes()
		if !bytes.Equal(recorderBody, testData) {
			t.Errorf("BUG CONFIRMED: Data NOT written to underlying ResponseWriter!\n"+
				"  Recorder body: %q (len=%d)\n"+
				"  Expected:      %q (len=%d)\n"+
				"  This means clients receive empty responses on cache miss.",
				recorderBody, len(recorderBody), testData, len(testData))
		} else {
			t.Log("Data correctly written to underlying ResponseWriter")
		}
	})
}

// TestResponseWriterWriteString tests that c.String-style responses (rendered via
// WriteString) are both streamed to the client and captured for caching.
func TestResponseWriterWriteString(t *testing.T) {
	gin.SetMode(gin.TestMode)

	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)

	writer := &responseWriter{
		ResponseWriter: c.Writer,
		body:           make([]byte, 0),
		statusCode:     http.StatusOK,
	}

	const testData = "plain text body"
	n, err := writer.WriteString(testData)
	if err != nil {
		t.Errorf("WriteString returned error: %v", err)
	}
	if n != len(testData) {
		t.Errorf("WriteString returned wrong byte count: got %d, want %d", n, len(testData))
	}
	if string(writer.body) != testData {
		t.Errorf("WriteString not captured for caching: got %q, want %q", writer.body, testData)
	}
	if recorder.Body.String() != testData {
		t.Errorf("WriteString not streamed to client: got %q, want %q", recorder.Body.String(), testData)
	}
}

// TestCacheMiddlewareCacheMiss simulates a full cache miss scenario
func TestCacheMiddlewareCacheMiss(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("Cache miss should return data to client", func(t *testing.T) {
		recorder := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(recorder)
		c.Request = httptest.NewRequest(http.MethodGet, "/test", nil)

		// Simulate what the cache middleware does on cache miss
		originalWriter := c.Writer
		writer := &responseWriter{
			ResponseWriter: originalWriter,
			body:           make([]byte, 0),
			statusCode:     http.StatusOK,
		}
		c.Writer = writer

		// Simulate handler writing response (like dashboard handler would)
		testResponse := []byte(`{"data": [{"id": 1, "name": "test"}]}`)
		c.Writer.WriteHeader(http.StatusOK)
		_, _ = c.Writer.Write(testResponse)

		// Check what the client would receive
		recorderBody := recorder.Body.Bytes()
		if len(recorderBody) == 0 {
			t.Errorf("BUG CONFIRMED: Client receives EMPTY response on cache miss!\n"+
				"  Response body sent to client: %q (len=%d)\n"+
				"  Data accumulated for cache:   %q (len=%d)\n"+
				"  This explains why dashboard is empty on first load.",
				recorderBody, len(recorderBody), writer.body, len(writer.body))
		} else if !bytes.Equal(recorderBody, testResponse) {
			t.Errorf("Client received incorrect data:\n  got:  %q\n  want: %q", recorderBody, testResponse)
		} else {
			t.Log("Client correctly received data on cache miss")
		}
	})
}
