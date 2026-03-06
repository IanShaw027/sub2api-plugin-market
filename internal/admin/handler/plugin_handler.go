package handler

import (
	"strconv"

	"github.com/IanShaw027/sub2api-plugin-market/ent"
	"github.com/IanShaw027/sub2api-plugin-market/ent/plugin"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// AdminPluginHandler 管理后台插件处理器
type AdminPluginHandler struct {
	client *ent.Client
}

// NewAdminPluginHandler 创建管理后台插件处理器
func NewAdminPluginHandler(client *ent.Client) *AdminPluginHandler {
	return &AdminPluginHandler{client: client}
}

// List 获取插件列表（包含所有状态）
func (h *AdminPluginHandler) List(c *gin.Context) {
	status := c.DefaultQuery("status", "")
	category := c.DefaultQuery("category", "")
	pluginType := c.DefaultQuery("plugin_type", "")
	search := c.DefaultQuery("search", "")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	query := h.client.Plugin.Query()

	if status != "" {
		s := plugin.Status(status)
		if err := plugin.StatusValidator(s); err != nil {
			Error(c, ErrCodeInvalidParam, "status 参数非法")
			return
		}
		query = query.Where(plugin.StatusEQ(s))
	}

	if category != "" {
		cat := plugin.Category(category)
		if err := plugin.CategoryValidator(cat); err != nil {
			Error(c, ErrCodeInvalidParam, "category 参数非法")
			return
		}
		query = query.Where(plugin.CategoryEQ(cat))
	}

	if pluginType != "" {
		pt := plugin.PluginType(pluginType)
		if err := plugin.PluginTypeValidator(pt); err != nil {
			Error(c, ErrCodeInvalidParam, "plugin_type 参数非法")
			return
		}
		query = query.Where(plugin.PluginTypeEQ(pt))
	}

	if search != "" {
		query = query.Where(
			plugin.Or(
				plugin.NameContains(search),
				plugin.DisplayNameContains(search),
				plugin.DescriptionContains(search),
			),
		)
	}

	total, err := query.Count(c.Request.Context())
	if err != nil {
		Error(c, ErrCodeDatabaseError, "查询插件总数失败")
		return
	}

	offset := (page - 1) * pageSize
	plugins, err := query.
		Order(ent.Desc(plugin.FieldCreatedAt)).
		Offset(offset).
		Limit(pageSize).
		All(c.Request.Context())
	if err != nil {
		Error(c, ErrCodeDatabaseError, "查询插件列表失败")
		return
	}

	Success(c, gin.H{
		"plugins":   plugins,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	})
}

// Get 获取插件详情（含所有版本）
func (h *AdminPluginHandler) Get(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		Error(c, ErrCodeInvalidParam, "无效的插件 ID")
		return
	}

	p, err := h.client.Plugin.Query().
		Where(plugin.IDEQ(id)).
		WithVersions(func(q *ent.PluginVersionQuery) {
			q.Order(ent.Desc("created_at"))
		}).
		Only(c.Request.Context())
	if err != nil {
		if ent.IsNotFound(err) {
			Error(c, ErrCodeNotFound, "插件不存在")
			return
		}
		Error(c, ErrCodeDatabaseError, "查询插件详情失败")
		return
	}

	Success(c, p)
}

// UpdatePluginRequest 更新插件请求
type UpdatePluginRequest struct {
	DisplayName *string `json:"display_name"`
	Category    *string `json:"category"`
	IsOfficial  *bool   `json:"is_official"`
	Status      *string `json:"status"`
	PluginType  *string `json:"plugin_type"`
	Description *string `json:"description"`
}

// Update 更新插件字段
func (h *AdminPluginHandler) Update(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		Error(c, ErrCodeInvalidParam, "无效的插件 ID")
		return
	}

	var req UpdatePluginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		Error(c, ErrCodeInvalidParam, "请求参数错误")
		return
	}

	update := h.client.Plugin.UpdateOneID(id)
	hasUpdate := false

	if req.DisplayName != nil && *req.DisplayName != "" {
		update = update.SetDisplayName(*req.DisplayName)
		hasUpdate = true
	}

	if req.Description != nil {
		update = update.SetDescription(*req.Description)
		hasUpdate = true
	}

	if req.Category != nil {
		cat := plugin.Category(*req.Category)
		if err := plugin.CategoryValidator(cat); err != nil {
			Error(c, ErrCodeInvalidParam, "category 参数非法")
			return
		}
		update = update.SetCategory(cat)
		hasUpdate = true
	}

	if req.PluginType != nil {
		pt := plugin.PluginType(*req.PluginType)
		if err := plugin.PluginTypeValidator(pt); err != nil {
			Error(c, ErrCodeInvalidParam, "plugin_type 参数非法")
			return
		}
		update = update.SetPluginType(pt)
		hasUpdate = true
	}

	if req.IsOfficial != nil {
		update = update.SetIsOfficial(*req.IsOfficial)
		hasUpdate = true
	}

	if req.Status != nil {
		s := plugin.Status(*req.Status)
		if err := plugin.StatusValidator(s); err != nil {
			Error(c, ErrCodeInvalidParam, "status 参数非法")
			return
		}
		update = update.SetStatus(s)
		hasUpdate = true
	}

	if !hasUpdate {
		Error(c, ErrCodeInvalidParam, "未提供任何更新字段")
		return
	}

	p, err := update.Save(c.Request.Context())
	if err != nil {
		if ent.IsNotFound(err) {
			Error(c, ErrCodeNotFound, "插件不存在")
			return
		}
		Error(c, ErrCodeDatabaseError, "更新插件失败")
		return
	}

	Success(c, p)
}
