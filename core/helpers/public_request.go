package helpers

import (
	"context"
	"net"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

// PublicClientIP resolves the real client IP for the public (no-auth) endpoints
// behind the Cloudflare -> Traefik -> pod chain. Global trusted proxies stay empty
// (so c.ClientIP() returns the ingress pod IP, which would collapse every visitor
// into one rate-limit bucket), so here we prefer the edge-provided headers:
//  1. CF-Connecting-IP (set by Cloudflare),
//  2. the first valid IP of X-Forwarded-For,
//  3. c.ClientIP() as a last resort.
//
// These headers are client-settable, so a caller can spoof them to pick its own
// bucket. That is no worse than the single-bucket behavior and never grants access:
// the share/webhook token remains the only credential.
func PublicClientIP(c *gin.Context) string {
	if ip := firstValidIP(c.GetHeader("CF-Connecting-IP")); ip != "" {
		return ip
	}
	if xff := c.GetHeader("X-Forwarded-For"); xff != "" {
		for _, part := range strings.Split(xff, ",") {
			if ip := firstValidIP(part); ip != "" {
				return ip
			}
		}
	}
	return c.ClientIP()
}

// firstValidIP trims s and returns it when it parses as an IP address, else "".
func firstValidIP(s string) string {
	s = strings.TrimSpace(s)
	if net.ParseIP(s) != nil {
		return s
	}
	return ""
}

// AllowRequest applies a per-IP fixed-window rate limit using Redis INCR + EXPIRE.
// It FAILS OPEN: when the client is nil or any command errors, the request is
// allowed, so a cache outage never blocks the public endpoints. keyPrefix groups a
// family of buckets (e.g. "public_budget_rl") and bucket distinguishes operations
// (e.g. "get"/"post") within it.
func AllowRequest(ctx context.Context, client *redis.Client, keyPrefix, bucket, ip string, window time.Duration, limit int) bool {
	if client == nil {
		return true
	}
	key := keyPrefix + ":" + bucket + ":" + ip
	count, err := client.Incr(ctx, key).Result()
	if err != nil {
		return true // fail-open
	}
	if count == 1 {
		// Best-effort expiry; ignore the error (fail-open).
		_ = client.Expire(ctx, key, window).Err()
	}
	return count <= int64(limit)
}
