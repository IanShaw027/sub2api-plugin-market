package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/IanShaw027/sub2api-plugin-market/ent"
	"github.com/IanShaw027/sub2api-plugin-market/ent/enttest"
	"github.com/IanShaw027/sub2api-plugin-market/ent/plugin"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	_ "github.com/mattn/go-sqlite3"
)

func setupPluginTestRouter(client *ent.Client) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	h := NewAdminPluginHandler(client)
	api := r.Group("/admin/api")
	{
		api.GET("/plugins", h.List)
		api.GET("/plugins/:id", h.Get)
		api.PUT("/plugins/:id", h.Update)
	}
	return r
}

func createAdminTestPlugin(t *testing.T, client *ent.Client, name string, opts ...func(*ent.PluginCreate)) *ent.Plugin {
	t.Helper()
	builder := client.Plugin.Create().
		SetName(name).
		SetDisplayName(name + " Display").
		SetDescription("Test plugin: " + name).
		SetAuthor("test-author").
		SetDownloadCount(0)
	for _, opt := range opts {
		opt(builder)
	}
	p, err := builder.Save(context.Background())
	require.NoError(t, err)
	return p
}

func TestAdminPluginHandler_List(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:plugin_list?mode=memory&_fk=1")
	defer client.Close()

	createAdminTestPlugin(t, client, "plugin-a", func(c *ent.PluginCreate) {
		c.SetStatus(plugin.StatusActive).SetCategory(plugin.CategoryProxy)
	})
	createAdminTestPlugin(t, client, "plugin-b", func(c *ent.PluginCreate) {
		c.SetStatus(plugin.StatusDeprecated).SetCategory(plugin.CategorySecurity)
	})
	createAdminTestPlugin(t, client, "plugin-c", func(c *ent.PluginCreate) {
		c.SetStatus(plugin.StatusActive).SetCategory(plugin.CategoryProxy)
	})

	router := setupPluginTestRouter(client)

	t.Run("list all plugins", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/admin/api/plugins", nil)
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var resp Response
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
		assert.Equal(t, 0, resp.Code)

		data := resp.Data.(map[string]interface{})
		assert.Equal(t, float64(3), data["total"])
		plugins := data["plugins"].([]interface{})
		assert.Len(t, plugins, 3)
	})

	t.Run("filter by status", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/admin/api/plugins?status=deprecated", nil)
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var resp Response
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
		data := resp.Data.(map[string]interface{})
		assert.Equal(t, float64(1), data["total"])
	})

	t.Run("filter by category", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/admin/api/plugins?category=proxy", nil)
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var resp Response
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
		data := resp.Data.(map[string]interface{})
		assert.Equal(t, float64(2), data["total"])
	})

	t.Run("search by name", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/admin/api/plugins?search=plugin-b", nil)
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var resp Response
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
		data := resp.Data.(map[string]interface{})
		assert.Equal(t, float64(1), data["total"])
	})

	t.Run("invalid status returns 400", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/admin/api/plugins?status=invalid", nil)
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestAdminPluginHandler_Get(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:plugin_get?mode=memory&_fk=1")
	defer client.Close()

	p := createAdminTestPlugin(t, client, "get-test-plugin")

	router := setupPluginTestRouter(client)

	t.Run("get existing plugin", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/admin/api/plugins/"+p.ID.String(), nil)
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var resp Response
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
		assert.Equal(t, 0, resp.Code)

		data := resp.Data.(map[string]interface{})
		assert.Equal(t, p.ID.String(), data["id"])
	})

	t.Run("get non-existent plugin", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/admin/api/plugins/00000000-0000-0000-0000-000000000000", nil)
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("get with invalid uuid", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/admin/api/plugins/not-a-uuid", nil)
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestAdminPluginHandler_Update(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:plugin_update?mode=memory&_fk=1")
	defer client.Close()

	p := createAdminTestPlugin(t, client, "update-test-plugin", func(c *ent.PluginCreate) {
		c.SetStatus(plugin.StatusActive)
	})

	router := setupPluginTestRouter(client)

	t.Run("update display_name and status", func(t *testing.T) {
		body, _ := json.Marshal(map[string]interface{}{
			"display_name": "New Display Name",
			"status":       "deprecated",
		})
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("PUT", "/admin/api/plugins/"+p.ID.String(), bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var resp Response
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
		assert.Equal(t, 0, resp.Code)

		data := resp.Data.(map[string]interface{})
		assert.Equal(t, "New Display Name", data["display_name"])
		assert.Equal(t, "deprecated", data["status"])
	})

	t.Run("update with invalid status", func(t *testing.T) {
		body, _ := json.Marshal(map[string]interface{}{
			"status": "invalid",
		})
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("PUT", "/admin/api/plugins/"+p.ID.String(), bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("update non-existent plugin", func(t *testing.T) {
		body, _ := json.Marshal(map[string]interface{}{
			"display_name": "nope",
		})
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("PUT", "/admin/api/plugins/00000000-0000-0000-0000-000000000000", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("update with empty body", func(t *testing.T) {
		body, _ := json.Marshal(map[string]interface{}{})
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("PUT", "/admin/api/plugins/"+p.ID.String(), bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}
