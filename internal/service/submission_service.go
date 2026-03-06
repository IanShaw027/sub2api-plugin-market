package service

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/IanShaw027/sub2api-plugin-market/ent"
	"github.com/IanShaw027/sub2api-plugin-market/ent/plugin"
	"github.com/IanShaw027/sub2api-plugin-market/ent/pluginversion"
	"github.com/IanShaw027/sub2api-plugin-market/ent/submission"
	"github.com/IanShaw027/sub2api-plugin-market/ent/trustkey"
	"github.com/IanShaw027/sub2api-storage"
)

// ErrInvalidSubmissionRequest 提交请求参数错误
var ErrInvalidSubmissionRequest = errors.New("invalid submission request")

// ErrPendingLimitExceeded 同一插件待审核提交数量超限
var ErrPendingLimitExceeded = errors.New("pending submission limit exceeded")

// pluginNameRegex 插件名只允许小写字母、数字和连字符，首尾必须是字母或数字，长度 2-64
var pluginNameRegex = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{0,62}[a-z0-9]$`)

// PluginManifest 插件清单（来自 manifest JSON）
type PluginManifest struct {
	Name             string   `json:"name"`
	Version          string   `json:"version"`
	PluginType       string   `json:"plugin_type"`
	PluginAPIVersion string   `json:"plugin_api_version"`
	Capabilities     []string `json:"capabilities"`
	MinAPIVersion    string   `json:"min_api_version"`
	MaxAPIVersion    string   `json:"max_api_version"`
}

// SubmissionService 开发者公开提交通道服务
type SubmissionService struct {
	client  *ent.Client
	storage storage.Storage
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

	// WASM 上传相关（可选）
	WASMData   []byte
	Manifest   *PluginManifest
	Signature  string
	SignKeyID  string
}

// CreateSubmissionResponse 创建提交响应
type CreateSubmissionResponse struct {
	SubmissionID string `json:"submission_id"`
	PluginID     string `json:"plugin_id"`
	Status       string `json:"status"`
}

// NewSubmissionService 创建开发者提交服务
func NewSubmissionService(client *ent.Client, storage storage.Storage) *SubmissionService {
	return &SubmissionService{client: client, storage: storage}
}

// CreateSubmission 创建开发者提交；支持纯元数据或 WASM 上传
func (s *SubmissionService) CreateSubmission(ctx context.Context, req *CreateSubmissionRequest) (*CreateSubmissionResponse, error) {
	// WASM 上传路径：需要 wasm 文件、manifest、signature、sign_key_id
	if len(req.WASMData) > 0 {
		return s.createSubmissionWithWASM(ctx, req)
	}
	// 纯元数据路径
	return s.createSubmissionMetadataOnly(ctx, req)
}

// createSubmissionWithWASM 处理 WASM 上传的完整流程
func (s *SubmissionService) createSubmissionWithWASM(ctx context.Context, req *CreateSubmissionRequest) (*CreateSubmissionResponse, error) {
	if req.Manifest == nil {
		return nil, fmt.Errorf("%w: WASM 上传必须提供 manifest", ErrInvalidSubmissionRequest)
	}
	m := req.Manifest

	pluginName := strings.TrimSpace(m.Name)
	if pluginName == "" {
		pluginName = strings.TrimSpace(req.PluginName)
	}
	if pluginName == "" {
		return nil, fmt.Errorf("%w: plugin_name 或 manifest.name 不能为空", ErrInvalidSubmissionRequest)
	}
	if len(pluginName) < 2 || !pluginNameRegex.MatchString(pluginName) {
		return nil, fmt.Errorf("%w: plugin_name 格式非法，仅允许小写字母、数字和连字符，长度 2-64", ErrInvalidSubmissionRequest)
	}
	if strings.TrimSpace(req.PluginName) != "" && strings.TrimSpace(req.PluginName) != pluginName {
		return nil, fmt.Errorf("%w: plugin_name 与 manifest.name 不一致", ErrInvalidSubmissionRequest)
	}

	version := strings.TrimSpace(m.Version)
	if version == "" {
		return nil, fmt.Errorf("%w: manifest.version 不能为空", ErrInvalidSubmissionRequest)
	}

	minAPIVersion := strings.TrimSpace(m.MinAPIVersion)
	if minAPIVersion == "" {
		minAPIVersion = "1.0.0"
	}
	pluginAPIVersion := strings.TrimSpace(m.PluginAPIVersion)
	if pluginAPIVersion == "" {
		pluginAPIVersion = "1.0.0"
	}

	displayName := strings.TrimSpace(req.DisplayName)
	if displayName == "" {
		displayName = pluginName
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
	signature := strings.TrimSpace(req.Signature)
	signKeyID := strings.TrimSpace(req.SignKeyID)
	if signature == "" || signKeyID == "" {
		return nil, fmt.Errorf("%w: WASM 上传必须提供 signature 和 sign_key_id", ErrInvalidSubmissionRequest)
	}

	// 计算 WASM SHA-256 哈希
	hash := sha256.Sum256(req.WASMData)
	wasmHash := "sha256-" + hex.EncodeToString(hash[:])
	fileSize := len(req.WASMData)
	if fileSize <= 0 {
		return nil, fmt.Errorf("%w: WASM 文件为空", ErrInvalidSubmissionRequest)
	}

	// 验签：用 sign_key_id 查找 trust_key，Ed25519 验证
	trustKey, err := s.client.TrustKey.Query().
		Where(trustkey.KeyIDEQ(signKeyID), trustkey.IsActiveEQ(true)).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, fmt.Errorf("%w: 签名验证失败，密钥不存在或已失效", ErrInvalidSubmissionRequest)
		}
		return nil, err
	}

	pubKeyBytes, err := base64.StdEncoding.DecodeString(trustKey.PublicKey)
	if err != nil {
		return nil, fmt.Errorf("%w: 签名验证失败，公钥格式错误", ErrInvalidSubmissionRequest)
	}
	if len(pubKeyBytes) != ed25519.PublicKeySize {
		return nil, fmt.Errorf("%w: 签名验证失败，公钥长度非法", ErrInvalidSubmissionRequest)
	}

	sigBytes, err := base64.StdEncoding.DecodeString(signature)
	if err != nil {
		return nil, fmt.Errorf("%w: 签名验证失败，签名格式错误", ErrInvalidSubmissionRequest)
	}

	if !ed25519.Verify(pubKeyBytes, []byte(wasmHash), sigBytes) {
		return nil, fmt.Errorf("%w: 签名验证失败", ErrInvalidSubmissionRequest)
	}

	// 检查版本是否已存在
	pluginRecord, err := s.client.Plugin.Query().
		Where(plugin.NameEQ(pluginName)).
		Only(ctx)
	isNewPlugin := false
	if err != nil {
		if !ent.IsNotFound(err) {
			return nil, err
		}
		// 新建 Plugin
		createBuilder := s.client.Plugin.Create().
			SetName(pluginName).
			SetDisplayName(displayName).
			SetDescription(strings.TrimSpace(req.Description)).
			SetAuthor(author).
			SetSourceType(plugin.SourceTypeUpload).
			SetAutoUpgradeEnabled(req.AutoUpgradeEnabled)
		if m.PluginType != "" {
			if err := plugin.PluginTypeValidator(plugin.PluginType(m.PluginType)); err == nil {
				createBuilder = createBuilder.SetPluginType(plugin.PluginType(m.PluginType))
			}
		}
		pluginRecord, err = createBuilder.Save(ctx)
		if err != nil {
			return nil, err
		}
		isNewPlugin = true
	} else {
		// 插件已存在，检查版本并可选更新 plugin_type
		exists, err := s.client.PluginVersion.Query().
			Where(
				pluginversion.PluginIDEQ(pluginRecord.ID),
				pluginversion.VersionEQ(version),
			).
			Exist(ctx)
		if err != nil {
			return nil, err
		}
		if exists {
			return nil, fmt.Errorf("%w: 版本 %s 已存在", ErrInvalidSubmissionRequest, version)
		}
		if m.PluginType != "" {
			if err := plugin.PluginTypeValidator(plugin.PluginType(m.PluginType)); err == nil {
				_, _ = pluginRecord.Update().SetPluginType(plugin.PluginType(m.PluginType)).Save(ctx)
			}
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
		return nil, fmt.Errorf("%w: 该插件已有 %d 个待审核提交，请等待审核完成后再提交", ErrPendingLimitExceeded, pendingCount)
	}

	// 上传 WASM 到 storage
	storageKey := fmt.Sprintf("plugins/%s/%s/plugin.wasm", pluginName, version)
	if _, err := s.storage.Upload(ctx, storageKey, bytes.NewReader(req.WASMData)); err != nil {
		return nil, fmt.Errorf("上传 WASM 失败: %w", err)
	}

	// 创建 PluginVersion
	pvBuilder := s.client.PluginVersion.Create().
		SetPluginID(pluginRecord.ID).
		SetVersion(version).
		SetStatus(pluginversion.StatusDraft).
		SetWasmURL(storageKey).
		SetWasmHash(wasmHash).
		SetSignature(signature).
		SetSignKeyID(signKeyID).
		SetFileSize(fileSize).
		SetMinAPIVersion(minAPIVersion).
		SetPluginAPIVersion(pluginAPIVersion)
	if m.MaxAPIVersion != "" {
		pvBuilder = pvBuilder.SetMaxAPIVersion(strings.TrimSpace(m.MaxAPIVersion))
	}
	if len(m.Capabilities) > 0 {
		pvBuilder = pvBuilder.SetCapabilities(m.Capabilities)
	}

	pv, err := pvBuilder.Save(ctx)
	if err != nil {
		if delErr := s.storage.Delete(ctx, storageKey); delErr != nil {
			// log cleanup failure but return original error
		}
		return nil, fmt.Errorf("创建插件版本失败: %w", err)
	}

	// 创建 Submission 并关联 PluginVersion
	submissionType := submission.SubmissionTypeNewVersion
	if isNewPlugin {
		submissionType = submission.SubmissionTypeNewPlugin
	}
	createSubmissionBuilder := s.client.Submission.Create().
		SetPluginID(pluginRecord.ID).
		SetVersionID(pv.ID).
		SetSubmissionType(submissionType).
		SetSubmitterName(submitterName).
		SetSubmitterEmail(submitterEmail).
		SetSourceType(submission.SourceTypeUpload).
		SetAutoUpgradeEnabled(req.AutoUpgradeEnabled).
		SetStatus(submission.StatusPending)
	if notes := strings.TrimSpace(req.Notes); notes != "" {
		createSubmissionBuilder = createSubmissionBuilder.SetNotes(notes)
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

// createSubmissionMetadataOnly 纯元数据提交（无 WASM）
func (s *SubmissionService) createSubmissionMetadataOnly(ctx context.Context, req *CreateSubmissionRequest) (*CreateSubmissionResponse, error) {
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
		return nil, fmt.Errorf("%w: 该插件已有 %d 个待审核提交，请等待审核完成后再提交", ErrPendingLimitExceeded, pendingCount)
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
