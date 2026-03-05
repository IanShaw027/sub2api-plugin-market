package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/IanShaw027/sub2api-plugin-market/internal/auth"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestRefresh_BadRequest(t *testing.T) {
	gin.SetMode(gin.TestMode)

	h := &AuthHandler{
		jwtService: auth.NewJWTService("test-secret", 2, 7),
	}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/admin/api/auth/refresh", bytes.NewBufferString("{}"))
	c.Request.Header.Set("Content-Type", "application/json")

	h.Refresh(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestRefresh_InvalidToken(t *testing.T) {
	gin.SetMode(gin.TestMode)

	h := &AuthHandler{
		jwtService: auth.NewJWTService("test-secret", 2, 7),
	}

	body, _ := json.Marshal(map[string]string{
		"refresh_token": "invalid-token",
	})

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/admin/api/auth/refresh", bytes.NewBuffer(body))
	c.Request.Header.Set("Content-Type", "application/json")

	h.Refresh(c)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestRefresh_WithAccessToken(t *testing.T) {
	gin.SetMode(gin.TestMode)

	jwtSvc := auth.NewJWTService("test-secret", 2, 7)
	accessToken, err := jwtSvc.GenerateToken("u1", "admin", "super_admin")
	assert.NoError(t, err)

	h := &AuthHandler{
		jwtService: jwtSvc,
	}

	body, _ := json.Marshal(map[string]string{
		"refresh_token": accessToken,
	})

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/admin/api/auth/refresh", bytes.NewBuffer(body))
	c.Request.Header.Set("Content-Type", "application/json")

	h.Refresh(c)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}
