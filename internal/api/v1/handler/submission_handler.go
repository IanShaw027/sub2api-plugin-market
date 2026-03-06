package handler

import (
	"encoding/json"
	"errors"
	"io"
	"mime/multipart"

	"github.com/IanShaw027/sub2api-plugin-market/internal/service"
	"github.com/gin-gonic/gin"
)

// SubmissionHandler 公开提交处理器
type SubmissionHandler struct {
	submissionService *service.SubmissionService
}

// CreateSubmissionRequest 提交请求体（JSON 或 form 字段）
type CreateSubmissionRequest struct {
	PluginName         string `json:"plugin_name" form:"plugin_name"`
	DisplayName        string `json:"display_name" form:"display_name"`
	Description        string `json:"description" form:"description"`
	Author             string `json:"author" form:"author"`
	SubmissionType     string `json:"submission_type" form:"submission_type"`
	SubmitterName      string `json:"submitter_name" form:"submitter_name"`
	SubmitterEmail     string `json:"submitter_email" form:"submitter_email"`
	Notes              string `json:"notes" form:"notes"`
	SourceType         string `json:"source_type" form:"source_type"`
	GithubRepoURL      string `json:"github_repo_url" form:"github_repo_url"`
	AutoUpgradeEnabled bool   `json:"auto_upgrade_enabled" form:"auto_upgrade_enabled"`
}

// NewSubmissionHandler 创建公开提交处理器
func NewSubmissionHandler(submissionService *service.SubmissionService) *SubmissionHandler {
	return &SubmissionHandler{submissionService: submissionService}
}

// CreateSubmission 创建公开提交
// POST /api/v1/submissions
// 支持 application/json 或 multipart/form-data
func (h *SubmissionHandler) CreateSubmission(c *gin.Context) {
	svcReq, hasWASM, err := h.parseSubmissionRequest(c)
	if err != nil {
		Error(c, ErrCodeInvalidParam, err.Error())
		return
	}

	resp, err := h.submissionService.CreateSubmission(c.Request.Context(), svcReq)
	if err != nil {
		if errors.Is(err, service.ErrPendingLimitExceeded) {
			Error(c, ErrCodePendingLimitExceeded, err.Error())
			return
		}
		if errors.Is(err, service.ErrInvalidSubmissionRequest) {
			Error(c, ErrCodeInvalidParam, err.Error())
			return
		}
		Error(c, ErrCodeDatabaseError, "创建提交失败")
		return
	}

	if hasWASM {
		SuccessCreated(c, resp)
	} else {
		Success(c, resp)
	}
}

// parseSubmissionRequest 解析请求：支持 JSON 或 multipart/form-data
func (h *SubmissionHandler) parseSubmissionRequest(c *gin.Context) (*service.CreateSubmissionRequest, bool, error) {
	contentType := c.GetHeader("Content-Type")
	if len(contentType) >= 19 && contentType[:19] == "multipart/form-data" {
		return h.parseMultipartSubmission(c)
	}
	return h.parseJSONSubmission(c)
}

func (h *SubmissionHandler) parseJSONSubmission(c *gin.Context) (*service.CreateSubmissionRequest, bool, error) {
	var req CreateSubmissionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		return nil, false, err
	}
	return h.buildServiceRequest(&req, nil, nil, "", ""), false, nil
}

func (h *SubmissionHandler) parseMultipartSubmission(c *gin.Context) (*service.CreateSubmissionRequest, bool, error) {
	if err := c.Request.ParseMultipartForm(32 << 20); err != nil {
		return nil, false, err
	}
	form := c.Request.MultipartForm

	req := &CreateSubmissionRequest{
		PluginName:         getFormValue(form, "plugin_name"),
		DisplayName:        getFormValue(form, "display_name"),
		Description:        getFormValue(form, "description"),
		Author:             getFormValue(form, "author"),
		SubmissionType:     getFormValue(form, "submission_type"),
		SubmitterName:      getFormValue(form, "submitter_name"),
		SubmitterEmail:     getFormValue(form, "submitter_email"),
		Notes:              getFormValue(form, "notes"),
		SourceType:         getFormValue(form, "source_type"),
		GithubRepoURL:      getFormValue(form, "github_repo_url"),
		AutoUpgradeEnabled: getFormValue(form, "auto_upgrade_enabled") == "true" || getFormValue(form, "auto_upgrade_enabled") == "1",
	}

	manifestStr := getFormValue(form, "manifest")
	signature := getFormValue(form, "signature")
	signKeyID := getFormValue(form, "sign_key_id")

	var wasmData []byte
	if fh, err := c.FormFile("wasm_file"); err == nil && fh != nil {
		f, err := fh.Open()
		if err != nil {
			return nil, false, err
		}
		defer f.Close()
		wasmData, err = io.ReadAll(f)
		if err != nil {
			return nil, false, err
		}
	}

	if len(wasmData) > 0 {
		if manifestStr == "" {
			return nil, false, errors.New("WASM 上传必须提供 manifest")
		}
		var m service.PluginManifest
		if err := json.Unmarshal([]byte(manifestStr), &m); err != nil {
			return nil, false, errors.New("manifest JSON 格式错误")
		}
		return h.buildServiceRequest(req, wasmData, &m, signature, signKeyID), true, nil
	}

	return h.buildServiceRequest(req, nil, nil, "", ""), false, nil
}

func getFormValue(form *multipart.Form, key string) string {
	if form == nil || form.Value == nil {
		return ""
	}
	if v, ok := form.Value[key]; ok && len(v) > 0 {
		return v[0]
	}
	return ""
}

func (h *SubmissionHandler) buildServiceRequest(req *CreateSubmissionRequest, wasmData []byte, manifest *service.PluginManifest, signature, signKeyID string) *service.CreateSubmissionRequest {
	svc := &service.CreateSubmissionRequest{
		PluginName:         req.PluginName,
		DisplayName:        req.DisplayName,
		Description:        req.Description,
		Author:             req.Author,
		SubmissionType:     req.SubmissionType,
		SubmitterName:      req.SubmitterName,
		SubmitterEmail:     req.SubmitterEmail,
		Notes:              req.Notes,
		SourceType:         req.SourceType,
		GithubRepoURL:      req.GithubRepoURL,
		AutoUpgradeEnabled: req.AutoUpgradeEnabled,
		WASMData:           wasmData,
		Manifest:           manifest,
		Signature:          signature,
		SignKeyID:          signKeyID,
	}
	return svc
}
