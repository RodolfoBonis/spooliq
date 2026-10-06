// Package health exposes liveness and readiness probes. Liveness reports that
// the process is up; readiness reports that its critical dependencies
// (Postgres, and Redis when enabled) are reachable.
package health

import (
	"context"
	"database/sql"
	"net/http"
	"time"

	"github.com/RodolfoBonis/go-otel-agent/logger"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

// readyTimeout bounds each dependency probe so a hung dependency cannot stall
// the readiness endpoint.
// Kept below the k8s probe timeout (1s) so a slow dependency still yields a
// proper 503 body inside the probe window.
const readyTimeout = 800 * time.Millisecond

const (
	statusOK      = "ok"
	statusSkipped = "skipped"
)

// Pinger is the minimal contract a dependency must satisfy to be probed. Both
// *sql.DB and a thin Redis adapter implement it, which also makes the handler
// trivial to unit test with a fake.
type Pinger interface {
	PingContext(ctx context.Context) error
}

// redisPinger adapts a *redis.Client to the Pinger contract.
type redisPinger struct {
	client *redis.Client
}

func (r redisPinger) PingContext(ctx context.Context) error {
	return r.client.Ping(ctx).Err()
}

// Handler serves the health endpoints. Redis is informational only: the cache
// layer fails open, so a Redis outage is reported ("error" / "skipped") but never
// makes the API unready — only the database does.
type Handler struct {
	db    Pinger
	redis Pinger
	// redisClient resolves the client at probe time: the Redis service is
	// initialized after the handler is built, so capturing it eagerly would
	// always yield nil.
	redisClient func() *redis.Client
	logger      logger.Logger
}

// NewHandler builds a health Handler from a *sql.DB and a resolver for the
// optional Redis client (the resolver may return nil when Redis is disabled or
// not initialized yet).
func NewHandler(db *sql.DB, redisClient func() *redis.Client, log logger.Logger) *Handler {
	h := &Handler{logger: log, redisClient: redisClient}
	if db != nil {
		h.db = db
	}
	return h
}

// redisProbe returns the Redis pinger to use for this probe, or nil when Redis
// is disabled or not initialized.
func (h *Handler) redisProbe() Pinger {
	if h.redis != nil {
		return h.redis
	}
	if h.redisClient == nil {
		return nil
	}
	if client := h.redisClient(); client != nil {
		return redisPinger{client: client}
	}
	return nil
}

// Register wires the health routes onto the provided /v1 router group.
//
//	GET /v1/health/live  -> liveness (no dependencies)
//	GET /v1/health/ready -> readiness (pings Postgres and Redis)
//	GET /v1/health_check -> legacy liveness alias (text/plain)
func (h *Handler) Register(group *gin.RouterGroup) {
	healthGroup := group.Group("/health")
	healthGroup.GET("/live", h.Live)
	healthGroup.GET("/ready", h.Ready)

	// Legacy alias kept for backward compatibility with existing monitors.
	group.GET("/health_check", h.LegacyCheck)
}

// Live reports process liveness. It has no dependencies and always returns 200
// while the process can serve requests.
//
// @Summary Liveness probe
// @Description Reports that the process is up. No dependencies are checked.
// @Tags Health
// @Produce json
// @Success 200 {object} map[string]string
// @Router /health/live [get]
func (h *Handler) Live(c *gin.Context) {
	h.debug(c, "liveness probe")
	c.JSON(http.StatusOK, gin.H{"status": statusOK})
}

// Ready reports readiness by pinging Postgres and, when configured, Redis. It
// returns 503 only when the database is down. A Redis outage returns 200 with
// status "degraded" (the cache fails open); disabled Redis reports "skipped".
//
// @Summary Readiness probe
// @Description Pings Postgres and (when enabled) Redis. Returns 503 only if the database is down; a Redis outage is reported as degraded.
// @Tags Health
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Failure 503 {object} map[string]interface{}
// @Router /health/ready [get]
func (h *Handler) Ready(c *gin.Context) {
	h.debug(c, "readiness probe")

	checks := make(map[string]string, 2)
	healthy := true

	// Database is required.
	if h.db == nil {
		checks["database"] = "unavailable"
		healthy = false
	} else if err := h.ping(c.Request.Context(), h.db); err != nil {
		checks["database"] = "error"
		healthy = false
	} else {
		checks["database"] = statusOK
	}

	// Redis is informational (the cache fails open): an outage degrades the
	// service but must not take it out of rotation.
	degraded := false
	if redisPinger := h.redisProbe(); redisPinger == nil {
		checks["redis"] = statusSkipped
	} else if err := h.ping(c.Request.Context(), redisPinger); err != nil {
		checks["redis"] = "error"
		degraded = true
	} else {
		checks["redis"] = statusOK
	}

	status := http.StatusOK
	body := gin.H{"status": statusOK, "checks": checks}
	switch {
	case !healthy:
		status = http.StatusServiceUnavailable
		body["status"] = "unavailable"
	case degraded:
		body["status"] = "degraded"
	}
	c.JSON(status, body)
}

// LegacyCheck preserves the original /v1/health_check endpoint, including its
// text/plain body, so existing probes keep working unchanged.
//
// @Summary Legacy health check
// @Description Backward-compatible liveness alias returning a plain-text body.
// @Tags Health
// @Produce plain
// @Success 200 {string} string "This Service is Healthy"
// @Router /health_check [get]
func (h *Handler) LegacyCheck(c *gin.Context) {
	h.debug(c, "legacy health check")
	c.String(http.StatusOK, "This Service is Healthy")
}

func (h *Handler) ping(ctx context.Context, p Pinger) error {
	ctx, cancel := context.WithTimeout(ctx, readyTimeout)
	defer cancel()
	return p.PingContext(ctx)
}

func (h *Handler) debug(c *gin.Context, msg string) {
	if h.logger != nil {
		h.logger.Debug(c.Request.Context(), msg)
	}
}
