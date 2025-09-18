package logger

import (
	"bytes"

	"github.com/gin-gonic/gin"
)

// BodyLogWriter is a custom writer for logging HTTP body content.
type BodyLogWriter struct {
	gin.ResponseWriter
	Body *bytes.Buffer
}

func (w BodyLogWriter) Write(b []byte) (int, error) {
	w.Body.Write(b)
	return w.ResponseWriter.Write(b)
}
