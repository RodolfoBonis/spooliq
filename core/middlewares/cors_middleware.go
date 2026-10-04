package middlewares

import (
	"net/http"
	"time"

	"github.com/RodolfoBonis/spooliq/core/config"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

// Cors returns a middleware that enables CORS support.
//
// Origins are read from the CORS_ALLOWED_ORIGINS environment variable
// (comma-separated). Two modes:
//   - unset/empty: allow all origins WITHOUT credentials (safe default; the
//     wildcard origin combined with credentials is forbidden by the CORS spec).
//   - explicit list: restrict to those origins AND enable credentials, so
//     browsers may send cookies / Authorization on cross-origin requests.
func Cors() gin.HandlerFunc {
	corsConfig := cors.Config{
		AllowMethods: []string{
			http.MethodGet,
			http.MethodPost,
			http.MethodPut,
			http.MethodPatch,
			http.MethodDelete,
			http.MethodOptions,
		},
		AllowHeaders: []string{
			"Content-Type",
			"Content-Length",
			"Accept-Encoding",
			"X-CSRF-Token",
			"Authorization",
			"Accept",
			"Origin",
			"Cache-Control",
			"X-Requested-With",
		},
		ExposeHeaders: []string{"Content-Length"},
		MaxAge:        12 * time.Hour,
	}

	origins := config.EnvCORSAllowedOrigins()
	if len(origins) == 0 {
		// Permissive default for local/dev or unconfigured environments: any
		// origin, but credentials disabled (wildcard + credentials is invalid).
		corsConfig.AllowAllOrigins = true
		corsConfig.AllowCredentials = false
	} else {
		corsConfig.AllowOrigins = origins
		corsConfig.AllowCredentials = true
	}

	return cors.New(corsConfig)
}
