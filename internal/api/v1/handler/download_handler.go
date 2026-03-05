package handler

import (
	"errors"
	"net/http"

	"github.com/IanShaw027/sub2api-plugin-market/internal/service"
	"github.com/gin-gonic/gin"
)

// DownloadHandler 下载处理器
type DownloadHandler struct {
	downloadService *service.DownloadService
}

// NewDownloadHandler 创建下载处理器
func NewDownloadHandler(downloadService *service.DownloadService) *DownloadHandler {
	return &DownloadHandler{
		downloadService: downloadService,
	}
}

// DownloadPlugin 下载插件接口
// GET /api/v1/plugins/:name/versions/:version/download
func (h *DownloadHandler) DownloadPlugin(c *gin.Context) {
	name := c.Param("name")
	version := c.Param("version")

	if name == "" || version == "" {
		Error(c, ErrCodeInvalidParam, "插件名称和版本号不能为空")
		return
	}

	clientIP := c.ClientIP()
	userAgent := c.GetHeader("User-Agent")

	// 获取下载 URL（预签名 URL）
	url, err := h.downloadService.GetDownloadURL(c.Request.Context(), name, version, clientIP, userAgent)
	if err != nil {
		if errors.Is(err, service.ErrPluginVersionNotFound) {
			Error(c, ErrCodeNotFound, "插件版本不存在")
			return
		}
		Error(c, ErrCodeInternalError, "插件校验失败")
		return
	}

	// 下载策略定版：统一返回 302 跳转到预签名 URL，由对象存储负责文件传输。
	c.Redirect(http.StatusFound, url)
}
