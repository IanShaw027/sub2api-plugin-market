package service

import (
	"context"
	"testing"

	"github.com/IanShaw027/sub2api-plugin-market/ent"
	"github.com/IanShaw027/sub2api-plugin-market/ent/enttest"
	"github.com/IanShaw027/sub2api-plugin-market/ent/plugin"
	"github.com/IanShaw027/sub2api-plugin-market/ent/submission"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	_ "github.com/mattn/go-sqlite3"
)

func createTestPlugin(t *testing.T, client *ent.Client, name string) *ent.Plugin {
	ctx := context.Background()
	p, err := client.Plugin.Create().
		SetName(name).
		SetDisplayName("Test " + name).
		SetDescription("test").
		SetAuthor("tester").
		SetCategory(plugin.CategoryOther).
		SetSourceType(plugin.SourceTypeUpload).
		SetStatus(plugin.StatusActive).
		Save(ctx)
	require.NoError(t, err)
	return p
}

func createTestSubmission(t *testing.T, client *ent.Client, pluginID uuid.UUID) *ent.Submission {
	ctx := context.Background()
	sub, err := client.Submission.Create().
		SetPluginID(pluginID).
		SetSubmissionType(submission.SubmissionTypeNewPlugin).
		SetSubmitterName("submitter").
		SetSubmitterEmail("submitter@example.com").
		SetSourceType(submission.SourceTypeUpload).
		SetStatus(submission.StatusPending).
		Save(ctx)
	require.NoError(t, err)
	return sub
}

func TestAdminSubmissionService_ListSubmissions_Empty(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:admin_sub_list_empty?mode=memory&cache=shared&_fk=1")
	defer client.Close()

	svc := NewSubmissionService(client)
	ctx := context.Background()

	subs, total, err := svc.ListSubmissions(ctx, "", 1, 10)
	require.NoError(t, err)
	assert.Empty(t, subs)
	assert.Equal(t, 0, total)
}

func TestAdminSubmissionService_ListSubmissions_WithStatusFilter(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:admin_sub_list_filter?mode=memory&cache=shared&_fk=1")
	defer client.Close()
	ctx := context.Background()

	p1 := createTestPlugin(t, client, "plugin-a")
	_ = createTestSubmission(t, client, p1.ID)

	p2 := createTestPlugin(t, client, "plugin-b")
	sub2 := createTestSubmission(t, client, p2.ID)
	_, err := sub2.Update().SetStatus(submission.StatusApproved).Save(ctx)
	require.NoError(t, err)

	svc := NewSubmissionService(client)

	pending, totalPending, err := svc.ListSubmissions(ctx, "pending", 1, 10)
	require.NoError(t, err)
	assert.Len(t, pending, 1)
	assert.Equal(t, 1, totalPending)
	assert.Equal(t, submission.StatusPending, pending[0].Status)

	approved, totalApproved, err := svc.ListSubmissions(ctx, "approved", 1, 10)
	require.NoError(t, err)
	assert.Len(t, approved, 1)
	assert.Equal(t, 1, totalApproved)
	assert.Equal(t, submission.StatusApproved, approved[0].Status)
}

func TestAdminSubmissionService_GetSubmission_Success(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:admin_sub_get?mode=memory&cache=shared&_fk=1")
	defer client.Close()

	p := createTestPlugin(t, client, "get-plugin")
	sub := createTestSubmission(t, client, p.ID)

	svc := NewSubmissionService(client)
	got, err := svc.GetSubmission(context.Background(), sub.ID.String())
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, sub.ID, got.ID)
	assert.NotNil(t, got.Edges.Plugin)
}

func TestAdminSubmissionService_GetSubmission_NotFound(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:admin_sub_get_nf?mode=memory&cache=shared&_fk=1")
	defer client.Close()

	svc := NewSubmissionService(client)

	_, err := svc.GetSubmission(context.Background(), "invalid-uuid")
	require.Error(t, err)

	_, err = svc.GetSubmission(context.Background(), uuid.New().String())
	require.Error(t, err)
}

func TestAdminSubmissionService_ReviewSubmission_Approve(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:admin_sub_approve?mode=memory&cache=shared&_fk=1")
	defer client.Close()
	ctx := context.Background()

	p := createTestPlugin(t, client, "approve-plugin")
	sub := createTestSubmission(t, client, p.ID)
	sub, _ = sub.Update().SetSourceType(submission.SourceTypeGithub).SetGithubRepoURL("https://github.com/org/repo").Save(ctx)

	svc := NewSubmissionService(client)
	err := svc.ReviewSubmission(ctx, sub.ID.String(), "approve", "looks good", "admin")
	require.NoError(t, err)

	updated, err := client.Submission.Get(ctx, sub.ID)
	require.NoError(t, err)
	assert.Equal(t, submission.StatusApproved, updated.Status)
	assert.Equal(t, "admin", updated.ReviewedBy)
	assert.Equal(t, "looks good", updated.ReviewerNotes)

	pluginUpdated, err := client.Plugin.Get(ctx, p.ID)
	require.NoError(t, err)
	assert.Equal(t, plugin.SourceTypeGithub, pluginUpdated.SourceType)
	assert.Equal(t, "https://github.com/org/repo", pluginUpdated.GithubRepoURL)
}

func TestAdminSubmissionService_ReviewSubmission_Reject(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:admin_sub_reject?mode=memory&cache=shared&_fk=1")
	defer client.Close()
	ctx := context.Background()

	p := createTestPlugin(t, client, "reject-plugin")
	sub := createTestSubmission(t, client, p.ID)

	svc := NewSubmissionService(client)
	err := svc.ReviewSubmission(ctx, sub.ID.String(), "reject", "needs more info", "admin")
	require.NoError(t, err)

	updated, err := client.Submission.Get(ctx, sub.ID)
	require.NoError(t, err)
	assert.Equal(t, submission.StatusRejected, updated.Status)
	assert.Equal(t, "admin", updated.ReviewedBy)
	assert.Equal(t, "needs more info", updated.ReviewerNotes)
}

func TestAdminSubmissionService_GetStats(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:admin_sub_stats?mode=memory&cache=shared&_fk=1")
	defer client.Close()
	ctx := context.Background()

	p1 := createTestPlugin(t, client, "stats-p1")
	p2 := createTestPlugin(t, client, "stats-p2")
	p3 := createTestPlugin(t, client, "stats-p3")

	sub1 := createTestSubmission(t, client, p1.ID)
	sub2 := createTestSubmission(t, client, p2.ID)
	_ = createTestSubmission(t, client, p3.ID)

	_, err := sub1.Update().SetStatus(submission.StatusApproved).Save(ctx)
	require.NoError(t, err)
	_, err = sub2.Update().SetStatus(submission.StatusRejected).Save(ctx)
	require.NoError(t, err)

	svc := NewSubmissionService(client)
	stats, err := svc.GetStats(ctx)
	require.NoError(t, err)
	assert.Equal(t, 3, stats.Total)
	assert.Equal(t, 1, stats.Pending)
	assert.Equal(t, 1, stats.Approved)
	assert.Equal(t, 1, stats.Rejected)
}
