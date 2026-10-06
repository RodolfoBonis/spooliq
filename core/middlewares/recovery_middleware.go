package middlewares

import (
	"fmt"
	"runtime/debug"

	"github.com/RodolfoBonis/go-otel-agent/logger"
	"github.com/RodolfoBonis/spooliq/core/errors"
	"github.com/gin-gonic/gin"
)

// Recovery returns a gin middleware that recovers from panics, logs the panic
// with its stack trace, and responds with the standard 500 error envelope. It
// replaces gin.Recovery() so panics produce the same JSON shape as every other
// error and never leak internal detail to clients.
func Recovery(log logger.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if rec := recover(); rec != nil {
				ctx := c.Request.Context()
				if log != nil {
					log.Error(ctx, "Recovered from panic", logger.Fields{
						"panic": fmt.Sprintf("%v", rec),
						"stack": string(debug.Stack()),
						"path":  c.Request.URL.Path,
					})
				}
				if !c.Writer.Written() {
					errors.AbortWith(c, errors.Internal())
				} else {
					c.Abort()
				}
			}
		}()
		c.Next()
	}
}
