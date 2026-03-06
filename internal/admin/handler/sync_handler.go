package handler

import (
	"errors"
	"io"
	"strconv"
	"time"

	"github.com/IanShaw027/sub2api-plugin-market/ent"
	"github.com/IanShaw027/sub2api-plugin-market/ent/syncjob"
	"github.com/IanShaw027/sub2api-plugin-market/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// SyncHandler 同步任务处理器
type SyncHandler struct {
	service *service.SyncService
}

// NewSyncHandler 创建同步任务处理器
func NewSyncHandler(service *service.SyncService) *SyncHandler {
	return &SyncHandler{service: service}
}

// CreateSyncRequest 创建同步请求
type CreateSyncRequest struct {
	TargetRef string `json:"target_ref"`
}

// CreateManualSync 创建并执行手动同步任务
func (h *SyncHandler) CreateManualSync(c *gin.Context) {
	pluginID := c.Param("id")

	var req CreateSyncRequest
	if err := c.ShouldBindJSON(&req); err != nil && !errors.Is(err, io.EOF) {
		Error(c, ErrCodeInvalidParam, "请求参数错误")
		return
	}

	job, err := h.service.CreateAndRunManualSync(c.Request.Context(), pluginID, req.TargetRef)
	if err != nil {
		Error(c, ErrCodeInternalError, "创建同步任务失败")
		return
	}

	Success(c, buildSyncJobResponse(job))
}

// GetSyncJob 获取同步任务详情
func (h *SyncHandler) GetSyncJob(c *gin.Context) {
	id := c.Param("id")

	job, err := h.service.GetSyncJob(c.Request.Context(), id)
	if err != nil {
		Error(c, ErrCodeNotFound, "同步任务不存在")
		return
	}

	Success(c, buildSyncJobResponse(job))
}

// ListSyncJobs 获取同步任务列表
func (h *SyncHandler) ListSyncJobs(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	status := c.DefaultQuery("status", "")
	if status != "" {
		s := syncjob.Status(status)
		if err := syncjob.StatusValidator(s); err != nil {
			Error(c, ErrCodeInvalidParam, "status 参数非法")
			return
		}
	}

	triggerType := c.DefaultQuery("trigger_type", "")
	if triggerType != "" {
		t := syncjob.TriggerType(triggerType)
		if err := syncjob.TriggerTypeValidator(t); err != nil {
			Error(c, ErrCodeInvalidParam, "trigger_type 参数非法")
			return
		}
	}

	var fromPtr *time.Time
	from := c.DefaultQuery("from", "")
	if from != "" {
		fromTime, err := time.Parse(time.RFC3339, from)
		if err != nil {
			Error(c, ErrCodeInvalidParam, "from 参数必须为 RFC3339 时间")
			return
		}
		fromPtr = &fromTime
	}

	var toPtr *time.Time
	to := c.DefaultQuery("to", "")
	if to != "" {
		toTime, err := time.Parse(time.RFC3339, to)
		if err != nil {
			Error(c, ErrCodeInvalidParam, "to 参数必须为 RFC3339 时间")
			return
		}
		toPtr = &toTime
	}

	pluginID := c.DefaultQuery("plugin_id", "")
	if pluginID != "" {
		if _, err := uuid.Parse(pluginID); err != nil {
			Error(c, ErrCodeInvalidParam, "plugin_id 参数非法")
			return
		}
	}

	jobs, total, err := h.service.ListSyncJobs(c.Request.Context(), service.ListSyncJobsParams{
		Status:      status,
		PluginID:    pluginID,
		TriggerType: triggerType,
		Page:        page,
		PageSize:    pageSize,
		From:        fromPtr,
		To:          toPtr,
	})
	if err != nil {
		Error(c, ErrCodeInternalError, "获取同步任务列表失败")
		return
	}

	jobResponses := make([]gin.H, 0, len(jobs))
	for _, job := range jobs {
		jobResponses = append(jobResponses, buildSyncJobResponse(job))
	}

	Success(c, gin.H{
		"jobs": jobResponses,
		"pagination": gin.H{
			"page":        page,
			"page_size":   pageSize,
			"total":       total,
			"total_pages": (total + pageSize - 1) / pageSize,
		},
	})
}

func buildSyncJobResponse(job *ent.SyncJob) gin.H {
	return gin.H{
		"id":            job.ID,
		"plugin_id":     job.PluginID,
		"trigger_type":  job.TriggerType,
		"status":        job.Status,
		"target_ref":    job.TargetRef,
		"error_message": job.ErrorMessage,
		"started_at":    job.StartedAt,
		"finished_at":   job.FinishedAt,
		"created_at":    job.CreatedAt,
		"updated_at":    job.UpdatedAt,
	}
}
