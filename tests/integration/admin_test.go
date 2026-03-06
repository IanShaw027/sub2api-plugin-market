package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/IanShaw027/sub2api-plugin-market/ent"
	"github.com/IanShaw027/sub2api-plugin-market/ent/enttest"
	"github.com/IanShaw027/sub2api-plugin-market/ent/plugin"
	"github.com/IanShaw027/sub2api-plugin-market/internal/admin"
	adminHandler "github.com/IanShaw027/sub2api-plugin-market/internal/admin/handler"
	adminService "github.com/IanShaw027/sub2api-plugin-market/internal/admin/service"
	"github.com/IanShaw027/sub2api-plugin-market/internal/auth"
	"github.com/IanShaw027/sub2api-plugin-market/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	_ "github.com/mattn/go-sqlite3"
)

const testJWTSecret = "test-secret-key-for-integration-tests"

type AdminTestContext struct {
	Client       *ent.Client
	Router       *gin.Engine
	JWTService   *auth.JWTService
	AdminService *auth.AdminService
}

func SetupAdminTestContext(t *testing.T) *AdminTestContext {
	client := enttest.Open(t, "sqlite3", fmt.Sprintf("file:admin_test_%d?mode=memory&cache=shared&_fk=1", time.Now().UnixNano()))

	jwtSvc := auth.NewJWTService(testJWTSecret, 2, 7)
	authSvc := auth.NewAdminService(client)
	adminSubSvc := adminService.NewSubmissionService(client)
	syncSvc := service.NewSyncService(client, &fakeStorage{})
	adminPluginHandler := adminHandler.NewAdminPluginHandler(client)
	adminVersionHandler := adminHandler.NewAdminVersionHandler(client)
	authHdl := adminHandler.NewAuthHandler(authSvc, jwtSvc)
	submissionHdl := adminHandler.NewSubmissionHandler(adminSubSvc)
	syncHdl := adminHandler.NewSyncHandler(syncSvc)

	gin.SetMode(gin.TestMode)
	router := gin.New()
	admin.RegisterRoutes(router, authHdl, submissionHdl, syncHdl, adminPluginHandler, adminVersionHandler, jwtSvc, authSvc)

	return &AdminTestContext{
		Client:       client,
		Router:       router,
		JWTService:   jwtSvc,
		AdminService: authSvc,
	}
}

func (atc *AdminTestContext) createTestAdmin(t *testing.T, username, role string) *ent.AdminUser {
	user, err := atc.AdminService.CreateAdmin(context.Background(), username, username+"@test.com", "testpass123", role)
	require.NoError(t, err)
	return user
}

func (atc *AdminTestContext) getAccessToken(t *testing.T, user *ent.AdminUser) string {
	token, err := atc.JWTService.GenerateToken(user.ID.String(), user.Username, string(user.Role))
	require.NoError(t, err)
	return token
}

func (atc *AdminTestContext) doRequest(method, path string, body []byte, token string) *httptest.ResponseRecorder {
	var bodyReader *bytes.Reader
	if body != nil {
		bodyReader = bytes.NewReader(body)
	}
	var req *http.Request
	if bodyReader != nil {
		req, _ = http.NewRequest(method, path, bodyReader)
		req.Header.Set("Content-Type", "application/json")
	} else {
		req, _ = http.NewRequest(method, path, nil)
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	w := httptest.NewRecorder()
	atc.Router.ServeHTTP(w, req)
	return w
}

// --- JWT Middleware Tests ---

func TestAdminAuth_MissingToken(t *testing.T) {
	atc := SetupAdminTestContext(t)
	defer atc.Client.Close()

	w := atc.doRequest("GET", "/admin/api/plugins", nil, "")
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestAdminAuth_InvalidBearerFormat(t *testing.T) {
	atc := SetupAdminTestContext(t)
	defer atc.Client.Close()

	req, _ := http.NewRequest("GET", "/admin/api/plugins", nil)
	req.Header.Set("Authorization", "Token some-token")
	w := httptest.NewRecorder()
	atc.Router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestAdminAuth_ExpiredToken(t *testing.T) {
	atc := SetupAdminTestContext(t)
	defer atc.Client.Close()

	expiredJWT := auth.NewJWTService(testJWTSecret, -1, 7)
	user := atc.createTestAdmin(t, "expired-user", "admin")
	token, err := expiredJWT.GenerateToken(user.ID.String(), user.Username, string(user.Role))
	require.NoError(t, err)

	w := atc.doRequest("GET", "/admin/api/plugins", nil, token)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestAdminAuth_RefreshTokenRejectedAsAccess(t *testing.T) {
	atc := SetupAdminTestContext(t)
	defer atc.Client.Close()

	user := atc.createTestAdmin(t, "refresh-user", "admin")
	refreshToken, err := atc.JWTService.GenerateRefreshToken(user.ID.String(), user.Username, string(user.Role))
	require.NoError(t, err)

	w := atc.doRequest("GET", "/admin/api/plugins", nil, refreshToken)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestAdminAuth_DisabledUser(t *testing.T) {
	atc := SetupAdminTestContext(t)
	defer atc.Client.Close()

	user := atc.createTestAdmin(t, "disabled-user", "admin")
	token := atc.getAccessToken(t, user)

	_, err := user.Update().SetIsActive(false).Save(context.Background())
	require.NoError(t, err)

	w := atc.doRequest("GET", "/admin/api/plugins", nil, token)
	assert.Equal(t, http.StatusForbidden, w.Code)
}

func TestAdminAuth_ValidToken_Success(t *testing.T) {
	atc := SetupAdminTestContext(t)
	defer atc.Client.Close()

	user := atc.createTestAdmin(t, "valid-user", "admin")
	token := atc.getAccessToken(t, user)

	w := atc.doRequest("GET", "/admin/api/plugins", nil, token)
	assert.Equal(t, http.StatusOK, w.Code)
}

// --- Login Integration Tests ---

func TestAdminLogin_Success(t *testing.T) {
	atc := SetupAdminTestContext(t)
	defer atc.Client.Close()

	atc.createTestAdmin(t, "loginuser", "admin")

	body := []byte(`{"username":"loginuser","password":"testpass123"}`)
	w := atc.doRequest("POST", "/admin/api/auth/login", body, "")

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]any
	json.Unmarshal(w.Body.Bytes(), &resp)
	data, ok := resp["data"].(map[string]any)
	require.True(t, ok, "response should have data field, got: %s", w.Body.String())
	assert.NotEmpty(t, data["token"])
	assert.NotEmpty(t, data["refresh_token"])
	user, ok := data["user"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "loginuser", user["username"])
}

func TestAdminLogin_WrongPassword(t *testing.T) {
	atc := SetupAdminTestContext(t)
	defer atc.Client.Close()

	atc.createTestAdmin(t, "wrongpassuser", "admin")

	body := []byte(`{"username":"wrongpassuser","password":"wrongpassword"}`)
	w := atc.doRequest("POST", "/admin/api/auth/login", body, "")

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestAdminLogin_NonExistentUser(t *testing.T) {
	atc := SetupAdminTestContext(t)
	defer atc.Client.Close()

	body := []byte(`{"username":"nouser","password":"pass"}`)
	w := atc.doRequest("POST", "/admin/api/auth/login", body, "")

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

// --- Admin Plugins API Integration ---

func TestAdminPlugins_ListWithAuth(t *testing.T) {
	atc := SetupAdminTestContext(t)
	defer atc.Client.Close()

	user := atc.createTestAdmin(t, "plugin-admin", "admin")
	token := atc.getAccessToken(t, user)

	_, err := atc.Client.Plugin.Create().
		SetName("test-plugin-1").
		SetDisplayName("Test Plugin 1").
		SetDescription("test").
		SetAuthor("tester").
		SetCategory(plugin.CategoryAnalytics).
		Save(context.Background())
	require.NoError(t, err)

	w := atc.doRequest("GET", "/admin/api/plugins", nil, token)
	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]any
	json.Unmarshal(w.Body.Bytes(), &resp)
	data, ok := resp["data"].(map[string]any)
	require.True(t, ok, "response should have data field, got: %s", w.Body.String())
	plugins, ok := data["plugins"].([]any)
	require.True(t, ok)
	assert.Len(t, plugins, 1)
}

func TestAdminPlugins_GetByID(t *testing.T) {
	atc := SetupAdminTestContext(t)
	defer atc.Client.Close()

	user := atc.createTestAdmin(t, "get-admin", "admin")
	token := atc.getAccessToken(t, user)

	p, err := atc.Client.Plugin.Create().
		SetName("get-plugin").
		SetDisplayName("Get Plugin").
		SetDescription("test").
		SetAuthor("tester").
		SetCategory(plugin.CategoryProxy).
		Save(context.Background())
	require.NoError(t, err)

	w := atc.doRequest("GET", fmt.Sprintf("/admin/api/plugins/%s", p.ID), nil, token)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestAdminPlugins_UpdatePlugin(t *testing.T) {
	atc := SetupAdminTestContext(t)
	defer atc.Client.Close()

	user := atc.createTestAdmin(t, "update-admin", "admin")
	token := atc.getAccessToken(t, user)

	p, err := atc.Client.Plugin.Create().
		SetName("update-plugin").
		SetDisplayName("Update Plugin").
		SetDescription("original").
		SetAuthor("tester").
		SetCategory(plugin.CategorySecurity).
		Save(context.Background())
	require.NoError(t, err)

	body := []byte(`{"description":"updated description"}`)
	w := atc.doRequest("PUT", fmt.Sprintf("/admin/api/plugins/%s", p.ID), body, token)
	assert.Equal(t, http.StatusOK, w.Code)
}

// --- Auth Me ---

func TestAdminAuth_GetMe(t *testing.T) {
	atc := SetupAdminTestContext(t)
	defer atc.Client.Close()

	user := atc.createTestAdmin(t, "me-user", "super_admin")
	token := atc.getAccessToken(t, user)

	w := atc.doRequest("GET", "/admin/api/auth/me", nil, token)
	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]any
	json.Unmarshal(w.Body.Bytes(), &resp)
	data, ok := resp["data"].(map[string]any)
	require.True(t, ok, "response should have data field, got: %s", w.Body.String())
	assert.Equal(t, "me-user", data["username"])
}
