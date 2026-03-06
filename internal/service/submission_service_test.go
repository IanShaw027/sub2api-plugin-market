package service

import (
	"context"
	"testing"

	"github.com/IanShaw027/sub2api-plugin-market/ent"
	"github.com/IanShaw027/sub2api-plugin-market/ent/enttest"
	"github.com/IanShaw027/sub2api-plugin-market/ent/plugin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	_ "github.com/mattn/go-sqlite3"
)

func setupSubmissionTest(t *testing.T) (*SubmissionService, *ent.Client) {
	client := enttest.Open(t, "sqlite3", "file:ent?mode=memory&cache=shared&_fk=1")
	return NewSubmissionService(client), client
}

func validCreateRequest() *CreateSubmissionRequest {
	return &CreateSubmissionRequest{
		PluginName:         "my-plugin",
		DisplayName:       "My Plugin",
		Description:       "A test plugin",
		Author:            "tester",
		SubmissionType:    "new_plugin",
		SubmitterName:     "submitter",
		SubmitterEmail:    "submitter@example.com",
		Notes:             "",
		SourceType:        "upload",
		GithubRepoURL:     "",
		AutoUpgradeEnabled: false,
	}
}

func TestSubmissionService_CreateSubmission_Success(t *testing.T) {
	svc, client := setupSubmissionTest(t)
	defer client.Close()
	ctx := context.Background()

	resp, err := svc.CreateSubmission(ctx, validCreateRequest())
	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.NotEmpty(t, resp.SubmissionID)
	assert.NotEmpty(t, resp.PluginID)
	assert.Equal(t, "pending", resp.Status)

	plugins, err := client.Plugin.Query().All(ctx)
	require.NoError(t, err)
	assert.Len(t, plugins, 1)
	assert.Equal(t, "my-plugin", plugins[0].Name)

	subs, err := client.Submission.Query().All(ctx)
	require.NoError(t, err)
	assert.Len(t, subs, 1)
}

func TestSubmissionService_CreateSubmission_EmptyPluginName(t *testing.T) {
	svc, client := setupSubmissionTest(t)
	defer client.Close()
	ctx := context.Background()

	req := validCreateRequest()
	req.PluginName = ""

	_, err := svc.CreateSubmission(ctx, req)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrInvalidSubmissionRequest)
}

func TestSubmissionService_CreateSubmission_InvalidSourceType(t *testing.T) {
	svc, client := setupSubmissionTest(t)
	defer client.Close()
	ctx := context.Background()

	req := validCreateRequest()
	req.SourceType = "invalid"

	_, err := svc.CreateSubmission(ctx, req)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrInvalidSubmissionRequest)
}

func TestSubmissionService_CreateSubmission_GithubWithoutURL(t *testing.T) {
	svc, client := setupSubmissionTest(t)
	defer client.Close()
	ctx := context.Background()

	req := validCreateRequest()
	req.SourceType = "github"
	req.GithubRepoURL = ""

	_, err := svc.CreateSubmission(ctx, req)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrInvalidSubmissionRequest)
}

func TestSubmissionService_CreateSubmission_InvalidPluginName_PathTraversal(t *testing.T) {
	svc, client := setupSubmissionTest(t)
	defer client.Close()
	ctx := context.Background()

	req := validCreateRequest()
	req.PluginName = "../hack"

	_, err := svc.CreateSubmission(ctx, req)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrInvalidSubmissionRequest)
}

func TestSubmissionService_CreateSubmission_InvalidPluginName_Slash(t *testing.T) {
	svc, client := setupSubmissionTest(t)
	defer client.Close()
	ctx := context.Background()

	req := validCreateRequest()
	req.PluginName = "plugin/../../etc"

	_, err := svc.CreateSubmission(ctx, req)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrInvalidSubmissionRequest)
}

func TestSubmissionService_CreateSubmission_InvalidPluginName_LeadingHyphen(t *testing.T) {
	svc, client := setupSubmissionTest(t)
	defer client.Close()
	ctx := context.Background()

	req := validCreateRequest()
	req.PluginName = "-invalid"

	_, err := svc.CreateSubmission(ctx, req)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrInvalidSubmissionRequest)
}

func TestSubmissionService_CreateSubmission_InvalidPluginName_TooShort(t *testing.T) {
	svc, client := setupSubmissionTest(t)
	defer client.Close()
	ctx := context.Background()

	req := validCreateRequest()
	req.PluginName = "a"

	_, err := svc.CreateSubmission(ctx, req)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrInvalidSubmissionRequest)
}

func TestSubmissionService_CreateSubmission_ValidPluginName_MinLength(t *testing.T) {
	svc, client := setupSubmissionTest(t)
	defer client.Close()
	ctx := context.Background()

	req := validCreateRequest()
	req.PluginName = "ab"

	resp, err := svc.CreateSubmission(ctx, req)
	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.NotEmpty(t, resp.SubmissionID)
	assert.Equal(t, "pending", resp.Status)
}

func TestSubmissionService_CreateSubmission_ExistingPlugin(t *testing.T) {
	svc, client := setupSubmissionTest(t)
	defer client.Close()
	ctx := context.Background()

	existing, err := client.Plugin.Create().
		SetName("existing-plugin").
		SetDisplayName("Existing").
		SetDescription("desc").
		SetAuthor("author").
		SetCategory(plugin.CategoryOther).
		SetSourceType(plugin.SourceTypeUpload).
		SetStatus(plugin.StatusActive).
		Save(ctx)
	require.NoError(t, err)

	req := validCreateRequest()
	req.PluginName = "existing-plugin"
	req.DisplayName = "Updated Display"

	resp, err := svc.CreateSubmission(ctx, req)
	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, existing.ID.String(), resp.PluginID)

	plugins, err := client.Plugin.Query().Where(plugin.NameEQ("existing-plugin")).All(ctx)
	require.NoError(t, err)
	assert.Len(t, plugins, 1)

	subs, err := client.Submission.Query().All(ctx)
	require.NoError(t, err)
	assert.Len(t, subs, 1)
}
