package service

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/IanShaw027/sub2api-plugin-market/ent"
	"github.com/IanShaw027/sub2api-plugin-market/ent/plugin"
	"github.com/IanShaw027/sub2api-plugin-market/ent/submission"
)

// ErrInvalidSubmissionRequest 提交请求参数错误
var ErrInvalidSubmissionRequest = errors.New("invalid submission request")

// pluginNameRegex 插件名只允许小写字母、数字和连字符，首尾必须是字母或数字，长度 2-64
var pluginNameRegex = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{0,62}[a-z0-9]$`)

// SubmissionService 开发者公开提交通道服务
type SubmissionService struct {
	client *ent.Client
}

// CreateSubmissionRequest 创建提交请求
type CreateSubmissionRequest struct {
	PluginName         string
	DisplayName        string
	Description        string
	Author             string
	SubmissionType     string
	SubmitterName      string
	SubmitterEmail     string
	Notes              string
	SourceType         string
	GithubRepoURL      string
	AutoUpgradeEnabled bool
}

// CreateSubmissionResponse 创建提交响应
type CreateSubmissionResponse struct {
	SubmissionID string `json:"submission_id"`
	PluginID     string `json:"plugin_id"`
	Status       string `json:"status"`
}

// NewSubmissionService 创建开发者提交服务
func NewSubmissionService(client *ent.Client) *SubmissionService {
	return &SubmissionService{client: client}
}

// CreateSubmission 创建开发者提交，MVP 仅落库元数据
func (s *SubmissionService) CreateSubmission(ctx context.Context, req *CreateSubmissionRequest) (*CreateSubmissionResponse, error) {
	pluginName := strings.TrimSpace(req.PluginName)
	if pluginName == "" {
		return nil, fmt.Errorf("%w: plugin_name 不能为空", ErrInvalidSubmissionRequest)
	}
	if len(pluginName) < 2 || !pluginNameRegex.MatchString(pluginName) {
		return nil, fmt.Errorf("%w: plugin_name 格式非法，仅允许小写字母、数字和连字符，长度 2-64", ErrInvalidSubmissionRequest)
	}

	displayName := strings.TrimSpace(req.DisplayName)
	if displayName == "" {
		return nil, fmt.Errorf("%w: display_name 不能为空", ErrInvalidSubmissionRequest)
	}

	author := strings.TrimSpace(req.Author)
	if author == "" {
		return nil, fmt.Errorf("%w: author 不能为空", ErrInvalidSubmissionRequest)
	}

	submitterName := strings.TrimSpace(req.SubmitterName)
	if submitterName == "" {
		return nil, fmt.Errorf("%w: submitter_name 不能为空", ErrInvalidSubmissionRequest)
	}

	submitterEmail := strings.TrimSpace(req.SubmitterEmail)
	if submitterEmail == "" {
		return nil, fmt.Errorf("%w: submitter_email 不能为空", ErrInvalidSubmissionRequest)
	}

	sourceType := submission.SourceType(strings.TrimSpace(req.SourceType))
	if sourceType != submission.SourceTypeUpload && sourceType != submission.SourceTypeGithub {
		return nil, fmt.Errorf("%w: source_type 仅支持 upload 或 github", ErrInvalidSubmissionRequest)
	}

	submissionType := submission.SubmissionType(strings.TrimSpace(req.SubmissionType))
	if submissionType != submission.SubmissionTypeNewPlugin &&
		submissionType != submission.SubmissionTypeNewVersion &&
		submissionType != submission.SubmissionTypeUpdateMetadata {
		return nil, fmt.Errorf("%w: submission_type 非法", ErrInvalidSubmissionRequest)
	}

	githubRepoURL := strings.TrimSpace(req.GithubRepoURL)
	if sourceType == submission.SourceTypeGithub && githubRepoURL == "" {
		return nil, fmt.Errorf("%w: source_type=github 时 github_repo_url 不能为空", ErrInvalidSubmissionRequest)
	}

	pluginRecord, err := s.client.Plugin.Query().
		Where(plugin.NameEQ(pluginName)).
		Only(ctx)
	if err != nil {
		if !ent.IsNotFound(err) {
			return nil, err
		}

		createBuilder := s.client.Plugin.Create().
			SetName(pluginName).
			SetDisplayName(displayName).
			SetDescription(strings.TrimSpace(req.Description)).
			SetAuthor(author).
			SetSourceType(plugin.SourceType(sourceType)).
			SetAutoUpgradeEnabled(req.AutoUpgradeEnabled)

		if githubRepoURL != "" {
			createBuilder = createBuilder.SetGithubRepoURL(githubRepoURL).
				SetGithubRepoNormalized(NormalizeGitHubRepoURL(githubRepoURL))
		}

		pluginRecord, err = createBuilder.Save(ctx)
		if err != nil {
			return nil, err
		}
	}

	pendingCount, err := s.client.Submission.Query().
		Where(
			submission.PluginIDEQ(pluginRecord.ID),
			submission.StatusEQ(submission.StatusPending),
		).
		Count(ctx)
	if err != nil {
		return nil, err
	}
	if pendingCount >= 3 {
		return nil, fmt.Errorf("%w: 该插件已有 %d 个待审核提交，请等待审核完成后再提交", ErrInvalidSubmissionRequest, pendingCount)
	}

	createSubmissionBuilder := s.client.Submission.Create().
		SetPluginID(pluginRecord.ID).
		SetSubmissionType(submissionType).
		SetSubmitterName(submitterName).
		SetSubmitterEmail(submitterEmail).
		SetSourceType(sourceType).
		SetAutoUpgradeEnabled(req.AutoUpgradeEnabled).
		SetStatus(submission.StatusPending)

	if notes := strings.TrimSpace(req.Notes); notes != "" {
		createSubmissionBuilder = createSubmissionBuilder.SetNotes(notes)
	}
	if githubRepoURL != "" {
		createSubmissionBuilder = createSubmissionBuilder.SetGithubRepoURL(githubRepoURL)
	}

	submissionRecord, err := createSubmissionBuilder.Save(ctx)
	if err != nil {
		return nil, err
	}

	return &CreateSubmissionResponse{
		SubmissionID: submissionRecord.ID.String(),
		PluginID:     pluginRecord.ID.String(),
		Status:       submissionRecord.Status.String(),
	}, nil
}
