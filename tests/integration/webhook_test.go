package integration

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGitHubWebhook_MissingEventHeader(t *testing.T) {
	tc := SetupTestContext(t)
	defer tc.Cleanup()

	w := tc.PerformRequestWithBody("POST", "/api/v1/integrations/github/webhook",
		bytes.NewBufferString(`{}`), nil)

	assert.Equal(t, 400, w.Code)
	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, float64(1001), resp["code"])
}

func TestGitHubWebhook_NonReleaseEvent(t *testing.T) {
	tc := SetupTestContext(t)
	defer tc.Cleanup()

	w := tc.PerformRequestWithBody("POST", "/api/v1/integrations/github/webhook",
		bytes.NewBufferString(`{}`),
		map[string]string{"X-GitHub-Event": "push"})

	assert.Equal(t, 200, w.Code)
	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, float64(0), resp["code"])
	assert.Equal(t, "ignored", resp["message"])
}

func TestGitHubWebhook_ReleaseNotPublished(t *testing.T) {
	tc := SetupTestContext(t)
	defer tc.Cleanup()

	payload := `{
		"action": "created",
		"repository": {"html_url": "https://github.com/test/repo"},
		"release": {"tag_name": "v1.0.0"}
	}`
	w := tc.PerformRequestWithBody("POST", "/api/v1/integrations/github/webhook",
		bytes.NewBufferString(payload),
		map[string]string{"X-GitHub-Event": "release"})

	assert.Equal(t, 200, w.Code)
	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, "ignored", resp["message"])
}

func TestGitHubWebhook_NoMatchingPlugin(t *testing.T) {
	tc := SetupTestContext(t)
	defer tc.Cleanup()

	payload := `{
		"action": "published",
		"repository": {"html_url": "https://github.com/unknown/repo"},
		"release": {"tag_name": "v1.0.0"}
	}`
	w := tc.PerformRequestWithBody("POST", "/api/v1/integrations/github/webhook",
		bytes.NewBufferString(payload),
		map[string]string{"X-GitHub-Event": "release"})

	assert.Equal(t, 200, w.Code)
	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, "ignored", resp["message"])
}
