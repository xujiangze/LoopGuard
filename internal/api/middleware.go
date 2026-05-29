package api

import (
	"net/http"
	"strings"

	"LoopGuard/internal/auth"
	"LoopGuard/internal/store"

	"github.com/gin-gonic/gin"
)

func APIKeyAuth(s *store.Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		plain := c.GetHeader("X-API-Key")
		if plain == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "缺少 X-API-Key"})
			return
		}
		k, err := s.GetAPIKeyByHash(auth.HashAPIKey(plain))
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "无效 API Key"})
			return
		}
		c.Set("api_key_id", k.ID)
		c.Next()
	}
}

func JWTAuth(secret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		h := c.GetHeader("Authorization")
		if !strings.HasPrefix(h, "Bearer ") {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "缺少 Bearer token"})
			return
		}
		claims, err := auth.ParseJWT(secret, strings.TrimPrefix(h, "Bearer "))
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "无效 token"})
			return
		}
		c.Set("user_id", claims.UserID)
		c.Set("role", claims.Role)
		c.Next()
	}
}

func APIKeyOrJWTAuth(s *store.Store, secret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Try API Key first
		if plain := c.GetHeader("X-API-Key"); plain != "" {
			k, err := s.GetAPIKeyByHash(auth.HashAPIKey(plain))
			if err == nil {
				c.Set("api_key_id", k.ID)
				c.Next()
				return
			}
		}
		// Fall back to JWT
		JWTAuth(secret)(c)
	}
}

func AdminOnly() gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.GetString("role") != "admin" {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "需要管理员权限"})
			return
		}
		c.Next()
	}
}
