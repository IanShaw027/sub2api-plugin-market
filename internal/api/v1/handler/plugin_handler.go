package handler

import (
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/sub2api/plugin-market/internal/service"
)

// PluginHandler 插件处理器
type PluginHandler struct {
	pluginService *service.PluginService
}

// NewPluginHandler 创建插件处理器
func NewPluginHandler(pluginService *service.PluginService) *PluginHandler {
	return &PluginHandler{
		pluginService: pluginService,
	}
}

// ListPlugins 插件列表接口
// GET /api/v1/plugins
func (h *PluginHandler) ListPlugins(c *gin.Context) {
	// 解析查询参数
	category := c.Query("category")
	search := c.Query("search")

	var isOfficial *bool
	if isOfficialStr := c.Query("is_official"); isOfficialStr != "" {
		val := isOfficialStr == "true"
		isOfficial = &val
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	// 调用服务层
	req := &service.ListPluginsRequest{
		Category:   category,
		Search:     search,
		IsOfficial: isOfficial,
		Page:       page,
		PageSize:   pageSize,
	}

	resp, err := h.pluginService.ListPlugins(c.Request.Context(), req)
	if err != nil {
		Error(c, ErrCodeDatabaseError, "查询插件列表失败")
		return
	}

	Success(c, resp)
}

// GetPluginDetail 插件详情接口
// GET /api/v1/plugins/:name
func (h *PluginHandler) GetPluginDetail(c *gin.Context) {
	name := c.Param("name")
	if name == "" {
		Error(c, ErrCodeInvalidParam, "插件名称不能为空")
		return
	}

	plugin, err := h.pluginService.GetPluginDetail(c.Request.Context(), name)
	if err != nil {
		Error(c, ErrCodeNotFound, "插件不存在")
		return
	}

	Success(c, plugin)
}

// GetPluginVersions 插件版本列表接口
// GET /api/v1/plugins/:name/versions
func (h *PluginHandler) GetPluginVersions(c *gin.Context) {
	name := c.Param("name")
	if name == "" {
		Error(c, ErrCodeInvalidParam, "插件名称不能为空")
		return
	}

	versions, err := h.pluginService.GetPluginVersions(c.Request.Context(), name)
	if err != nil {
		Error(c, ErrCodeNotFound, "插件不存在")
		return
	}

	Success(c, gin.H{
		"plugin_name": name,
		"versions":    versions,
		"total":       len(versions),
	})
}
