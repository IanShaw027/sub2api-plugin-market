package service

import (
	"context"
	"testing"

	"github.com/IanShaw027/sub2api-plugin-market/ent"
	"github.com/IanShaw027/sub2api-plugin-market/ent/enttest"
	"github.com/IanShaw027/sub2api-plugin-market/ent/plugin"
	"github.com/IanShaw027/sub2api-plugin-market/ent/pluginversion"
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
	err := svc.ReviewSubmission(ctx, sub.ID.String(), "approve", "looks good", "admin", "admin")
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
	err := svc.ReviewSubmission(ctx, sub.ID.String(), "reject", "needs more info", "admin", "admin")
	require.NoError(t, err)

	updated, err := client.Submission.Get(ctx, sub.ID)
	require.NoError(t, err)
	assert.Equal(t, submission.StatusRejected, updated.Status)
	assert.Equal(t, "admin", updated.ReviewedBy)
	assert.Equal(t, "needs more info", updated.ReviewerNotes)
}

func TestAdminSubmissionService_ReviewSubmission_AlreadyApproved(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:admin_sub_already_approved?mode=memory&cache=shared&_fk=1")
	defer client.Close()
	ctx := context.Background()

	p := createTestPlugin(t, client, "already-approved-plugin")
	sub := createTestSubmission(t, client, p.ID)

	svc := NewSubmissionService(client)

	err := svc.ReviewSubmission(ctx, sub.ID.String(), "approve", "first review", "admin1", "admin")
	require.NoError(t, err)

	updated, err := client.Submission.Get(ctx, sub.ID)
	require.NoError(t, err)
	assert.Equal(t, submission.StatusApproved, updated.Status)

	err = svc.ReviewSubmission(ctx, sub.ID.String(), "approve", "second review", "admin2", "admin")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "already been reviewed")

	err = svc.ReviewSubmission(ctx, sub.ID.String(), "reject", "late reject", "admin3", "admin")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "already been reviewed")
}

func TestAdminSubmissionService_ReviewSubmission_AlreadyRejected(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:admin_sub_already_rejected?mode=memory&cache=shared&_fk=1")
	defer client.Close()
	ctx := context.Background()

	p := createTestPlugin(t, client, "already-rejected-plugin")
	sub := createTestSubmission(t, client, p.ID)

	svc := NewSubmissionService(client)

	err := svc.ReviewSubmission(ctx, sub.ID.String(), "reject", "not good enough", "admin1", "admin")
	require.NoError(t, err)

	err = svc.ReviewSubmission(ctx, sub.ID.String(), "approve", "actually approve", "admin2", "admin")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "already been reviewed")
}

func createTestOfficialPlugin(t *testing.T, client *ent.Client, name string) *ent.Plugin {
	ctx := context.Background()
	p, err := client.Plugin.Create().
		SetName(name).
		SetDisplayName("Official " + name).
		SetDescription("official plugin").
		SetAuthor("official-team").
		SetCategory(plugin.CategoryOther).
		SetSourceType(plugin.SourceTypeUpload).
		SetStatus(plugin.StatusActive).
		SetIsOfficial(true).
		Save(ctx)
	require.NoError(t, err)
	return p
}

func TestAdminSubmissionService_ReviewSubmission_OfficialPluginByReviewer(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:admin_sub_official_reviewer?mode=memory&cache=shared&_fk=1")
	defer client.Close()
	ctx := context.Background()

	p := createTestOfficialPlugin(t, client, "official-plugin")
	sub := createTestSubmission(t, client, p.ID)

	svc := NewSubmissionService(client)
	err := svc.ReviewSubmission(ctx, sub.ID.String(), "approve", "lgtm", "reviewer-user", "reviewer")
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrForbiddenReview)

	updated, err := client.Submission.Get(ctx, sub.ID)
	require.NoError(t, err)
	assert.Equal(t, submission.StatusPending, updated.Status)
}

func TestAdminSubmissionService_ReviewSubmission_OfficialPluginByAdmin(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:admin_sub_official_admin?mode=memory&cache=shared&_fk=1")
	defer client.Close()
	ctx := context.Background()

	p := createTestOfficialPlugin(t, client, "official-plugin-admin")
	sub := createTestSubmission(t, client, p.ID)

	svc := NewSubmissionService(client)
	err := svc.ReviewSubmission(ctx, sub.ID.String(), "approve", "approved by admin", "admin-user", "admin")
	require.NoError(t, err)

	updated, err := client.Submission.Get(ctx, sub.ID)
	require.NoError(t, err)
	assert.Equal(t, submission.StatusApproved, updated.Status)
	assert.Equal(t, "admin-user", updated.ReviewedBy)
}

func TestAdminSubmissionService_ReviewSubmission_AutoPublishVersion(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:admin_sub_autopublish?mode=memory&cache=shared&_fk=1")
	defer client.Close()
	ctx := context.Background()

	p := createTestPlugin(t, client, "auto-publish-test")

	ver, err := client.PluginVersion.Create().
		SetPluginID(p.ID).
		SetVersion("1.0.0").
		SetWasmURL("/test/1.0.0").
		SetWasmHash("hash123").
		SetSignature("sig123").
		SetFileSize(1024).
		SetMinAPIVersion("1.0.0").
		SetPluginAPIVersion("1.0.0").
		SetStatus("draft").
		Save(ctx)
	require.NoError(t, err)

	sub := createTestSubmission(t, client, p.ID)
	_, err = client.Submission.UpdateOneID(sub.ID).SetVersion(ver).Save(ctx)
	require.NoError(t, err)

	svc := NewSubmissionService(client)
	err = svc.ReviewSubmission(ctx, sub.ID.String(), "approve", "looks good", "admin-user", "admin")
	require.NoError(t, err)

	updatedVer, err := client.PluginVersion.Get(ctx, ver.ID)
	require.NoError(t, err)
	assert.Equal(t, pluginversion.StatusPublished, updatedVer.Status)
	assert.False(t, updatedVer.PublishedAt.IsZero())
}

func TestAdminSubmissionService_ReviewSubmission_RejectDoesNotPublishVersion(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:admin_sub_reject_nopublish?mode=memory&cache=shared&_fk=1")
	defer client.Close()
	ctx := context.Background()

	p := createTestPlugin(t, client, "reject-nopublish-test")

	ver, err := client.PluginVersion.Create().
		SetPluginID(p.ID).
		SetVersion("1.0.0").
		SetWasmURL("/test/1.0.0").
		SetWasmHash("hash123").
		SetSignature("sig123").
		SetFileSize(1024).
		SetMinAPIVersion("1.0.0").
		SetPluginAPIVersion("1.0.0").
		SetStatus("draft").
		Save(ctx)
	require.NoError(t, err)

	sub := createTestSubmission(t, client, p.ID)
	_, err = client.Submission.UpdateOneID(sub.ID).SetVersion(ver).Save(ctx)
	require.NoError(t, err)

	svc := NewSubmissionService(client)
	err = svc.ReviewSubmission(ctx, sub.ID.String(), "reject", "not ready", "admin-user", "admin")
	require.NoError(t, err)

	updatedVer, err := client.PluginVersion.Get(ctx, ver.ID)
	require.NoError(t, err)
	assert.Equal(t, pluginversion.StatusDraft, updatedVer.Status)
	assert.True(t, updatedVer.PublishedAt.IsZero())
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
