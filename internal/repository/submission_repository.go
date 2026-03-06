package repository

import (
	"context"
	"time"

	"github.com/IanShaw027/sub2api-plugin-market/ent"
	"github.com/IanShaw027/sub2api-plugin-market/ent/submission"
	"github.com/google/uuid"
)

// SubmissionRepository 提交审核数据访问层
type SubmissionRepository struct {
	client *ent.Client
}

// NewSubmissionRepository 创建提交审核仓库
func NewSubmissionRepository(client *ent.Client) *SubmissionRepository {
	return &SubmissionRepository{client: client}
}

// CreateSubmissionParams 创建提交参数
type CreateSubmissionParams struct {
	PluginID           uuid.UUID
	SubmissionType     submission.SubmissionType
	SubmitterName      string
	SubmitterEmail     string
	Notes              string
	SourceType         submission.SourceType
	GithubRepoURL      string
	AutoUpgradeEnabled bool
}

// Create 创建提交记录
func (r *SubmissionRepository) Create(ctx context.Context, params CreateSubmissionParams) (*ent.Submission, error) {
	create := r.client.Submission.Create().
		SetPluginID(params.PluginID).
		SetSubmissionType(params.SubmissionType).
		SetSubmitterName(params.SubmitterName).
		SetSubmitterEmail(params.SubmitterEmail).
		SetSourceType(params.SourceType).
		SetAutoUpgradeEnabled(params.AutoUpgradeEnabled).
		SetStatus(submission.StatusPending)

	if params.Notes != "" {
		create = create.SetNotes(params.Notes)
	}
	if params.GithubRepoURL != "" {
		create = create.SetGithubRepoURL(params.GithubRepoURL)
	}

	return create.Save(ctx)
}

// GetByID 根据 ID 获取提交
func (r *SubmissionRepository) GetByID(ctx context.Context, id uuid.UUID) (*ent.Submission, error) {
	return r.client.Submission.Get(ctx, id)
}

// GetByIDWithPlugin 根据 ID 获取提交（含关联插件）
func (r *SubmissionRepository) GetByIDWithPlugin(ctx context.Context, id uuid.UUID) (*ent.Submission, error) {
	return r.client.Submission.Query().
		Where(submission.IDEQ(id)).
		WithPlugin().
		Only(ctx)
}

// List 分页查询提交列表
func (r *SubmissionRepository) List(ctx context.Context, status string, page, pageSize int) ([]*ent.Submission, int, error) {
	query := r.client.Submission.Query().WithPlugin()

	if status != "" {
		query = query.Where(submission.StatusEQ(submission.Status(status)))
	}

	total, err := query.Count(ctx)
	if err != nil {
		return nil, 0, err
	}

	submissions, err := query.
		Order(ent.Desc(submission.FieldCreatedAt)).
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		All(ctx)
	if err != nil {
		return nil, 0, err
	}

	return submissions, total, nil
}

// UpdateStatus 更新提交状态
func (r *SubmissionRepository) UpdateStatus(ctx context.Context, id uuid.UUID, status, reviewerNotes, reviewedBy string) error {
	now := time.Now()
	_, err := r.client.Submission.UpdateOneID(id).
		SetStatus(submission.Status(status)).
		SetReviewerNotes(reviewerNotes).
		SetReviewedBy(reviewedBy).
		SetReviewedAt(now).
		Save(ctx)
	return err
}

// CountByStatus 按状态统计提交数量
func (r *SubmissionRepository) CountByStatus(ctx context.Context, status string) (int, error) {
	return r.client.Submission.Query().
		Where(submission.StatusEQ(submission.Status(status))).
		Count(ctx)
}

// CountAll 统计全部提交数量
func (r *SubmissionRepository) CountAll(ctx context.Context) (int, error) {
	return r.client.Submission.Query().Count(ctx)
}
