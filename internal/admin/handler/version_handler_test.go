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
	"github.com/IanShaw027/sub2api-plugin-market/ent/pluginversion"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	_ "github.com/mattn/go-sqlite3"
)

func setupVersionTestRouter(client *ent.Client) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	h := NewAdminVersionHandler(client)
	api := r.Group("/admin/api")
	{
		api.GET("/plugins/:id/versions", h.List)
		api.PUT("/plugins/:id/versions/:vid/status", h.UpdateStatus)
	}
	return r
}

func createTestVersion(t *testing.T, client *ent.Client, pluginID uuid.UUID, ver string, status pluginversion.Status) *ent.PluginVersion {
	t.Helper()
	builder := client.PluginVersion.Create().
		SetPluginID(pluginID).
		SetVersion(ver).
		SetWasmURL("/test/" + ver).
		SetWasmHash("hash-" + ver).
		SetFileSize(1024).
		SetMinAPIVersion("1.0.0").
		SetPluginAPIVersion("1.0.0").
		SetStatus(status)
	pv, err := builder.Save(context.Background())
	require.NoError(t, err)
	return pv
}

func TestAdminVersionHandler_List(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:version_list?mode=memory&_fk=1")
	defer client.Close()

	p := createAdminTestPlugin(t, client, "ver-list-plugin")
	createTestVersion(t, client, p.ID, "1.0.0", pluginversion.StatusPublished)
	createTestVersion(t, client, p.ID, "2.0.0-beta", pluginversion.StatusDraft)
	createTestVersion(t, client, p.ID, "0.9.0", pluginversion.StatusYanked)

	router := setupVersionTestRouter(client)

	t.Run("list all versions including draft and yanked", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/admin/api/plugins/"+p.ID.String()+"/versions", nil)
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var resp Response
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
		assert.Equal(t, 0, resp.Code)

		data := resp.Data.(map[string]interface{})
		versions := data["versions"].([]interface{})
		assert.Len(t, versions, 3)
	})

	t.Run("list versions for non-existent plugin", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/admin/api/plugins/00000000-0000-0000-0000-000000000000/versions", nil)
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})
}

func TestAdminVersionHandler_UpdateStatus_Publish(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:version_publish?mode=memory&_fk=1")
	defer client.Close()

	p := createAdminTestPlugin(t, client, "publish-plugin")
	v := createTestVersion(t, client, p.ID, "1.0.0", pluginversion.StatusDraft)

	router := setupVersionTestRouter(client)

	body, _ := json.Marshal(map[string]string{"status": "published"})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("PUT", "/admin/api/plugins/"+p.ID.String()+"/versions/"+v.ID.String()+"/status", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp Response
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, 0, resp.Code)

	data := resp.Data.(map[string]interface{})
	assert.Equal(t, "published", data["status"])
	assert.NotNil(t, data["published_at"], "published_at should be set on draft->published")

	updated, err := client.PluginVersion.Get(context.Background(), v.ID)
	require.NoError(t, err)
	assert.Equal(t, pluginversion.StatusPublished, updated.Status)
	assert.False(t, updated.PublishedAt.IsZero())
}

func TestAdminVersionHandler_UpdateStatus_Yank(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:version_yank?mode=memory&_fk=1")
	defer client.Close()

	p := createAdminTestPlugin(t, client, "yank-plugin")
	v := createTestVersion(t, client, p.ID, "1.0.0", pluginversion.StatusPublished)

	router := setupVersionTestRouter(client)

	body, _ := json.Marshal(map[string]string{"status": "yanked"})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("PUT", "/admin/api/plugins/"+p.ID.String()+"/versions/"+v.ID.String()+"/status", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp Response
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	data := resp.Data.(map[string]interface{})
	assert.Equal(t, "yanked", data["status"])

	updated, err := client.PluginVersion.Get(context.Background(), v.ID)
	require.NoError(t, err)
	assert.Equal(t, pluginversion.StatusYanked, updated.Status)
}

func TestAdminVersionHandler_UpdateStatus_Invalid(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:version_invalid?mode=memory&_fk=1")
	defer client.Close()

	p := createAdminTestPlugin(t, client, "invalid-plugin")
	v := createTestVersion(t, client, p.ID, "1.0.0", pluginversion.StatusDraft)

	router := setupVersionTestRouter(client)

	tests := []struct {
		name       string
		fromStatus pluginversion.Status
		toStatus   string
		versionID  uuid.UUID
		wantCode   int
	}{
		{
			name:       "draft -> yanked is not allowed",
			fromStatus: pluginversion.StatusDraft,
			toStatus:   "yanked",
			versionID:  v.ID,
			wantCode:   http.StatusBadRequest,
		},
		{
			name:       "invalid status value",
			fromStatus: pluginversion.StatusDraft,
			toStatus:   "bogus",
			versionID:  v.ID,
			wantCode:   http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, _ := json.Marshal(map[string]string{"status": tt.toStatus})
			w := httptest.NewRecorder()
			req, _ := http.NewRequest("PUT", "/admin/api/plugins/"+p.ID.String()+"/versions/"+tt.versionID.String()+"/status", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			router.ServeHTTP(w, req)

			assert.Equal(t, tt.wantCode, w.Code)
		})
	}
}
