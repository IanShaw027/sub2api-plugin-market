package handler

import (
	"fmt"
	"time"

	"github.com/IanShaw027/sub2api-plugin-market/ent"
	"github.com/IanShaw027/sub2api-plugin-market/ent/plugin"
	"github.com/IanShaw027/sub2api-plugin-market/ent/pluginversion"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// AdminVersionHandler 管理后台版本处理器
type AdminVersionHandler struct {
	client *ent.Client
}

// NewAdminVersionHandler 创建管理后台版本处理器
func NewAdminVersionHandler(client *ent.Client) *AdminVersionHandler {
	return &AdminVersionHandler{client: client}
}

// List 获取插件版本列表（包含所有状态）
func (h *AdminVersionHandler) List(c *gin.Context) {
	idStr := c.Param("id")
	pluginID, err := uuid.Parse(idStr)
	if err != nil {
		Error(c, ErrCodeInvalidParam, "无效的插件 ID")
		return
	}

	exists, err := h.client.Plugin.Query().
		Where(plugin.IDEQ(pluginID)).
		Exist(c.Request.Context())
	if err != nil {
		Error(c, ErrCodeDatabaseError, "查询插件失败")
		return
	}
	if !exists {
		Error(c, ErrCodeNotFound, "插件不存在")
		return
	}

	versions, err := h.client.PluginVersion.Query().
		Where(pluginversion.PluginIDEQ(pluginID)).
		Order(ent.Desc(pluginversion.FieldCreatedAt)).
		All(c.Request.Context())
	if err != nil {
		Error(c, ErrCodeDatabaseError, "查询版本列表失败")
		return
	}

	Success(c, gin.H{
		"versions": versions,
	})
}

// UpdateStatusRequest 更新版本状态请求
type UpdateStatusRequest struct {
	Status string `json:"status" binding:"required"`
}

// validTransitions defines allowed version status transitions.
var validTransitions = map[pluginversion.Status]map[pluginversion.Status]bool{
	pluginversion.StatusDraft:     {pluginversion.StatusPublished: true},
	pluginversion.StatusPublished: {pluginversion.StatusYanked: true},
	pluginversion.StatusYanked:    {pluginversion.StatusPublished: true},
}

// UpdateStatus 更新版本状态
func (h *AdminVersionHandler) UpdateStatus(c *gin.Context) {
	idStr := c.Param("id")
	if _, err := uuid.Parse(idStr); err != nil {
		Error(c, ErrCodeInvalidParam, "无效的插件 ID")
		return
	}

	vidStr := c.Param("vid")
	vid, err := uuid.Parse(vidStr)
	if err != nil {
		Error(c, ErrCodeInvalidParam, "无效的版本 ID")
		return
	}

	var req UpdateStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		Error(c, ErrCodeInvalidParam, "请求参数错误")
		return
	}

	newStatus := pluginversion.Status(req.Status)
	if err := pluginversion.StatusValidator(newStatus); err != nil {
		Error(c, ErrCodeInvalidParam, "status 参数非法")
		return
	}

	version, err := h.client.PluginVersion.Get(c.Request.Context(), vid)
	if err != nil {
		if ent.IsNotFound(err) {
			Error(c, ErrCodeNotFound, "版本不存在")
			return
		}
		Error(c, ErrCodeDatabaseError, "查询版本失败")
		return
	}

	allowed, ok := validTransitions[version.Status]
	if !ok || !allowed[newStatus] {
		Error(c, ErrCodeInvalidParam, fmt.Sprintf("不允许从 %s 转换到 %s", version.Status, newStatus))
		return
	}

	update := h.client.PluginVersion.UpdateOneID(vid).SetStatus(newStatus)
	if newStatus == pluginversion.StatusPublished && version.Status == pluginversion.StatusDraft {
		update = update.SetPublishedAt(time.Now())
	}

	updated, err := update.Save(c.Request.Context())
	if err != nil {
		Error(c, ErrCodeDatabaseError, "更新版本状态失败")
		return
	}

	Success(c, updated)
}
