package handler

import (
	"errors"

	"github.com/IanShaw027/sub2api-plugin-market/internal/service"
	"github.com/gin-gonic/gin"
)

// SubmissionHandler 公开提交处理器
type SubmissionHandler struct {
	submissionService *service.SubmissionService
}

// CreateSubmissionRequest 提交请求体
type CreateSubmissionRequest struct {
	PluginName         string `json:"plugin_name" binding:"required"`
	DisplayName        string `json:"display_name" binding:"required"`
	Description        string `json:"description"`
	Author             string `json:"author" binding:"required"`
	SubmissionType     string `json:"submission_type" binding:"required"`
	SubmitterName      string `json:"submitter_name" binding:"required"`
	SubmitterEmail     string `json:"submitter_email" binding:"required"`
	Notes              string `json:"notes"`
	SourceType         string `json:"source_type" binding:"required"`
	GithubRepoURL      string `json:"github_repo_url"`
	AutoUpgradeEnabled bool   `json:"auto_upgrade_enabled"`
}

// NewSubmissionHandler 创建公开提交处理器
func NewSubmissionHandler(submissionService *service.SubmissionService) *SubmissionHandler {
	return &SubmissionHandler{submissionService: submissionService}
}

// CreateSubmission 创建公开提交
// POST /api/v1/submissions
func (h *SubmissionHandler) CreateSubmission(c *gin.Context) {
	var req CreateSubmissionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		Error(c, ErrCodeInvalidParam, "请求参数格式错误")
		return
	}

	resp, err := h.submissionService.CreateSubmission(c.Request.Context(), &service.CreateSubmissionRequest{
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
	})
	if err != nil {
		if errors.Is(err, service.ErrInvalidSubmissionRequest) {
			Error(c, ErrCodeInvalidParam, err.Error())
			return
		}
		Error(c, ErrCodeDatabaseError, "创建提交失败")
		return
	}

	Success(c, resp)
}
