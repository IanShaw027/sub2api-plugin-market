package handler

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/IanShaw027/sub2api-plugin-market/internal/admin/service"
	"github.com/gin-gonic/gin"
)

// SubmissionHandler 提交审核处理器
type SubmissionHandler struct {
	service *service.SubmissionService
}

// NewSubmissionHandler 创建提交审核处理器
func NewSubmissionHandler(service *service.SubmissionService) *SubmissionHandler {
	return &SubmissionHandler{service: service}
}

// List 获取提交列表
func (h *SubmissionHandler) List(c *gin.Context) {
	// 获取查询参数
	status := c.DefaultQuery("status", "")
	pageStr := c.DefaultQuery("page", "1")
	pageSizeStr := c.DefaultQuery("page_size", "20")

	page, _ := strconv.Atoi(pageStr)
	pageSize, _ := strconv.Atoi(pageSizeStr)

	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	submissions, total, err := h.service.ListSubmissions(c.Request.Context(), status, page, pageSize)
	if err != nil {
		Error(c, ErrCodeInternalError, "获取提交列表失败")
		return
	}

	Success(c, gin.H{
		"submissions": submissions,
		"pagination": gin.H{
			"page":        page,
			"page_size":   pageSize,
			"total":       total,
			"total_pages": (total + pageSize - 1) / pageSize,
		},
	})
}

// Get 获取提交详情
func (h *SubmissionHandler) Get(c *gin.Context) {
	id := c.Param("id")

	submission, err := h.service.GetSubmission(c.Request.Context(), id)
	if err != nil {
		Error(c, ErrCodeNotFound, "提交不存在")
		return
	}

	Success(c, submission)
}

// ReviewRequest 审核请求
type ReviewRequest struct {
	Action        string `json:"action" binding:"required"`
	ReviewerNotes string `json:"reviewer_notes"`
	Comment       string `json:"comment"`
}

func normalizeReviewRequest(req *ReviewRequest) error {
	action := strings.ToLower(strings.TrimSpace(req.Action))

	switch action {
	case "approved":
		req.Action = "approve"
	case "rejected":
		req.Action = "reject"
	case "approve", "reject":
		req.Action = action
	default:
		return fmt.Errorf("invalid action: %s", req.Action)
	}

	if req.ReviewerNotes == "" {
		req.ReviewerNotes = req.Comment
	}

	return nil
}

// Review 审核提交
func (h *SubmissionHandler) Review(c *gin.Context) {
	id := c.Param("id")

	var req ReviewRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		Error(c, ErrCodeInvalidParam, "请求参数错误")
		return
	}

	if err := normalizeReviewRequest(&req); err != nil {
		Error(c, ErrCodeInvalidParam, "请求参数错误")
		return
	}

	// 获取当前管理员
	username, _ := c.Get("admin_username")

	err := h.service.ReviewSubmission(c.Request.Context(), id, req.Action, req.ReviewerNotes, username.(string))
	if err != nil {
		Error(c, ErrCodeInternalError, "审核失败")
		return
	}

	Success(c, nil)
}

// Stats 获取审核统计
func (h *SubmissionHandler) Stats(c *gin.Context) {
	stats, err := h.service.GetStats(c.Request.Context())
	if err != nil {
		Error(c, ErrCodeInternalError, "获取统计数据失败")
		return
	}

	Success(c, stats)
}
