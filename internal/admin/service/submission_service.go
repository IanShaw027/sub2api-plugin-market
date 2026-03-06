package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/IanShaw027/sub2api-plugin-market/ent"
	"github.com/IanShaw027/sub2api-plugin-market/ent/plugin"
	"github.com/IanShaw027/sub2api-plugin-market/ent/submission"
	pubsvc "github.com/IanShaw027/sub2api-plugin-market/internal/service"
	"github.com/google/uuid"
)

// ErrForbiddenReview 权限不足：reviewer 不能审核官方插件
var ErrForbiddenReview = errors.New("权限不足：官方插件仅允许 admin 或 super_admin 审核")

// SubmissionService 提交审核服务
type SubmissionService struct {
	client *ent.Client
}

// NewSubmissionService 创建提交审核服务
func NewSubmissionService(client *ent.Client) *SubmissionService {
	return &SubmissionService{client: client}
}

// ListSubmissions 获取提交列表
func (s *SubmissionService) ListSubmissions(ctx context.Context, status string, page, pageSize int) ([]*ent.Submission, int, error) {
	query := s.client.Submission.Query().WithPlugin()

	// 状态筛选
	if status != "" {
		query = query.Where(submission.StatusEQ(submission.Status(status)))
	}

	// 获取总数
	total, err := query.Count(ctx)
	if err != nil {
		return nil, 0, err
	}

	// 分页查询
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

// GetSubmission 获取提交详情
func (s *SubmissionService) GetSubmission(ctx context.Context, id string) (*ent.Submission, error) {
	uid, err := uuid.Parse(id)
	if err != nil {
		return nil, err
	}
	return s.client.Submission.Query().Where(submission.IDEQ(uid)).WithPlugin().Only(ctx)
}

// ReviewSubmission 审核提交
func (s *SubmissionService) ReviewSubmission(ctx context.Context, id string, action, reviewerNotes, reviewerName, reviewerRole string) error {
	uid, err := uuid.Parse(id)
	if err != nil {
		return err
	}

	sub, err := s.client.Submission.Get(ctx, uid)
	if err != nil {
		return err
	}

	pluginRecord, err := s.client.Plugin.Get(ctx, sub.PluginID)
	if err != nil {
		return fmt.Errorf("查询关联插件失败: %w", err)
	}
	if pluginRecord.IsOfficial && reviewerRole == "reviewer" {
		return ErrForbiddenReview
	}

	tx, err := s.client.Tx(ctx)
	if err != nil {
		return err
	}
	defer func() {
		if v := recover(); v != nil {
			tx.Rollback()
			panic(v)
		}
	}()

	var newStatus submission.Status
	if action == "approve" {
		newStatus = submission.StatusApproved
	} else {
		newStatus = submission.StatusRejected
	}

	now := time.Now()
	n, err := tx.Submission.Update().
		Where(
			submission.IDEQ(uid),
			submission.StatusEQ(submission.StatusPending),
		).
		SetStatus(newStatus).
		SetReviewerNotes(reviewerNotes).
		SetReviewedBy(reviewerName).
		SetReviewedAt(now).
		Save(ctx)
	if err != nil {
		tx.Rollback()
		return err
	}
	if n == 0 {
		tx.Rollback()
		return fmt.Errorf("submission has already been reviewed or is no longer pending")
	}

	if action == "approve" {
		pluginUpdate := tx.Plugin.UpdateOneID(sub.PluginID).
			SetSourceType(plugin.SourceType(sub.SourceType)).
			SetAutoUpgradeEnabled(sub.AutoUpgradeEnabled)
		if sub.GithubRepoURL != "" {
			pluginUpdate = pluginUpdate.SetGithubRepoURL(sub.GithubRepoURL).
				SetGithubRepoNormalized(pubsvc.NormalizeGitHubRepoURL(sub.GithubRepoURL))
		}
		if _, err := pluginUpdate.Save(ctx); err != nil {
			tx.Rollback()
			return err
		}
	}

	return tx.Commit()
}

// ApproveSubmission 批准提交
func (s *SubmissionService) ApproveSubmission(ctx context.Context, id string, reviewerUsername, notes, reviewerRole string) error {
	return s.ReviewSubmission(ctx, id, "approve", notes, reviewerUsername, reviewerRole)
}

// RejectSubmission 拒绝提交
func (s *SubmissionService) RejectSubmission(ctx context.Context, id string, reviewerUsername, reason, reviewerRole string) error {
	return s.ReviewSubmission(ctx, id, "reject", reason, reviewerUsername, reviewerRole)
}

// Stats 审核统计
type Stats struct {
	Total    int `json:"total"`
	Pending  int `json:"pending"`
	Approved int `json:"approved"`
	Rejected int `json:"rejected"`
}

// GetStats 获取审核统计
func (s *SubmissionService) GetStats(ctx context.Context) (*Stats, error) {
	total, err := s.client.Submission.Query().Count(ctx)
	if err != nil {
		return nil, err
	}

	pending, err := s.client.Submission.Query().
		Where(submission.StatusEQ(submission.StatusPending)).
		Count(ctx)
	if err != nil {
		return nil, err
	}

	approved, err := s.client.Submission.Query().
		Where(submission.StatusEQ(submission.StatusApproved)).
		Count(ctx)
	if err != nil {
		return nil, err
	}

	rejected, err := s.client.Submission.Query().
		Where(submission.StatusEQ(submission.StatusRejected)).
		Count(ctx)
	if err != nil {
		return nil, err
	}

	return &Stats{
		Total:    total,
		Pending:  pending,
		Approved: approved,
		Rejected: rejected,
	}, nil
}
