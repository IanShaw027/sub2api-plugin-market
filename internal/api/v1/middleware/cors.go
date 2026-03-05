package middleware

import (
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
)

const corsAllowedOriginsEnv = "CORS_ALLOWED_ORIGINS"

type corsPolicy struct {
	allowLocalDev bool
	allowed       map[string]struct{}
}

// CORS 中间件
func CORS() gin.HandlerFunc {
	policy := loadCORSPolicy()

	return func(c *gin.Context) {
		origin := strings.TrimSpace(c.GetHeader("Origin"))
		allowedOrigin, isAllowed := matchAllowedOrigin(origin, policy)

		if isAllowed {
			c.Writer.Header().Set("Access-Control-Allow-Origin", allowedOrigin)
			c.Writer.Header().Set("Vary", "Origin")
		}

		c.Writer.Header().Set("Access-Control-Allow-Credentials", "false")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, DELETE, PATCH")

		if c.Request.Method == http.MethodOptions {
			if origin != "" && !isAllowed {
				c.AbortWithStatus(http.StatusForbidden)
				return
			}

			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}

// loadCORSPolicy 加载 CORS 白名单策略
func loadCORSPolicy() corsPolicy {
	value, exists := os.LookupEnv(corsAllowedOriginsEnv)
	if !exists {
		// 默认仅允许本地开发来源，避免线上默认放开。
		return corsPolicy{allowLocalDev: true, allowed: map[string]struct{}{}}
	}

	allowed := make(map[string]struct{})
	for _, item := range strings.Split(value, ",") {
		normalized, ok := normalizeOrigin(item)
		if !ok {
			continue
		}
		allowed[normalized] = struct{}{}
	}

	return corsPolicy{allowLocalDev: false, allowed: allowed}
}

// matchAllowedOrigin 判断来源是否在白名单
func matchAllowedOrigin(origin string, policy corsPolicy) (string, bool) {
	normalized, ok := normalizeOrigin(origin)
	if !ok {
		return "", false
	}

	if _, exists := policy.allowed[normalized]; exists {
		return normalized, true
	}

	if policy.allowLocalDev && isLocalDevOrigin(normalized) {
		return normalized, true
	}

	return "", false
}

// normalizeOrigin 将来源统一为可比较的 origin（scheme://host[:port]）
func normalizeOrigin(origin string) (string, bool) {
	trimmed := strings.TrimSpace(origin)
	if trimmed == "" {
		return "", false
	}

	parsed, err := url.Parse(trimmed)
	if err != nil {
		return "", false
	}

	if (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" {
		return "", false
	}

	host := strings.ToLower(parsed.Hostname())
	if host == "" {
		return "", false
	}

	if port := parsed.Port(); port != "" {
		return parsed.Scheme + "://" + host + ":" + port, true
	}

	return parsed.Scheme + "://" + host, true
}

// isLocalDevOrigin 判断是否为本地开发来源
func isLocalDevOrigin(origin string) bool {
	parsed, err := url.Parse(origin)
	if err != nil {
		return false
	}

	host := strings.ToLower(parsed.Hostname())
	return host == "localhost" || host == "127.0.0.1" || host == "::1"
}
