package integration

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateSubmission_Success(t *testing.T) {
	tc := SetupTestContext(t)
	defer tc.Cleanup()

	body := `{
		"plugin_name": "test-submission-plugin",
		"display_name": "Test Plugin",
		"description": "A test plugin",
		"author": "test-author",
		"submission_type": "new_plugin",
		"submitter_name": "Test User",
		"submitter_email": "test@example.com",
		"source_type": "upload"
	}`

	w := tc.PerformRequestWithBody("POST", "/api/v1/submissions", bytes.NewBufferString(body), nil)

	assert.Equal(t, 200, w.Code)
	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, float64(0), resp["code"])
	data := resp["data"].(map[string]interface{})
	assert.NotEmpty(t, data["submission_id"])
	assert.NotEmpty(t, data["plugin_id"])
	assert.Equal(t, "pending", data["status"])
}

func TestCreateSubmission_MissingRequiredField(t *testing.T) {
	tc := SetupTestContext(t)
	defer tc.Cleanup()

	body := `{"display_name": "Test"}`
	w := tc.PerformRequestWithBody("POST", "/api/v1/submissions", bytes.NewBufferString(body), nil)

	assert.Equal(t, 400, w.Code)
	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, float64(1001), resp["code"])
}

func TestCreateSubmission_InvalidSourceType(t *testing.T) {
	tc := SetupTestContext(t)
	defer tc.Cleanup()

	body := `{
		"plugin_name": "test-plugin",
		"display_name": "Test",
		"author": "author",
		"submission_type": "new_plugin",
		"submitter_name": "User",
		"submitter_email": "u@e.com",
		"source_type": "invalid"
	}`
	w := tc.PerformRequestWithBody("POST", "/api/v1/submissions", bytes.NewBufferString(body), nil)

	assert.Equal(t, 400, w.Code)
	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, float64(1001), resp["code"])
}

func TestCreateSubmission_GitHubWithoutURL(t *testing.T) {
	tc := SetupTestContext(t)
	defer tc.Cleanup()

	body := `{
		"plugin_name": "test-plugin",
		"display_name": "Test",
		"author": "author",
		"submission_type": "new_plugin",
		"submitter_name": "User",
		"submitter_email": "u@e.com",
		"source_type": "github"
	}`
	w := tc.PerformRequestWithBody("POST", "/api/v1/submissions", bytes.NewBufferString(body), nil)

	assert.Equal(t, 400, w.Code)
	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, float64(1001), resp["code"])
}

func TestCreateSubmission_ExistingPlugin(t *testing.T) {
	tc := SetupTestContext(t)
	defer tc.Cleanup()

	tc.CreateTestPlugin(t, "existing-plugin", "proxy", false)

	body := `{
		"plugin_name": "existing-plugin",
		"display_name": "Existing Plugin",
		"description": "Already exists",
		"author": "author",
		"submission_type": "new_version",
		"submitter_name": "User",
		"submitter_email": "u@e.com",
		"source_type": "upload"
	}`
	w := tc.PerformRequestWithBody("POST", "/api/v1/submissions", bytes.NewBufferString(body), nil)

	assert.Equal(t, 200, w.Code)
	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, float64(0), resp["code"])
	data := resp["data"].(map[string]interface{})
	assert.Equal(t, "pending", data["status"])
}
