package handler

import (
	"errors"

	"github.com/gin-gonic/gin"
	"github.com/IanShaw027/sub2api-plugin-market/internal/service"
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

	// TODO: 可以选择直接返回文件流或重定向到预签名 URL
	// 方案 1: 重定向到预签名 URL
	c.Redirect(302, url)

	// 方案 2: 直接返回文件流
	// pv, reader, err := h.downloadService.DownloadPlugin(c.Request.Context(), name, version)
	// if err != nil {
	//     Error(c, ErrCodeNotFound, "插件版本不存在")
	//     return
	// }
	// defer reader.Close()
	//
	// c.Header("Content-Type", "application/wasm")
	// c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%s-%s.wasm", name, version))
	// c.Header("Content-Length", strconv.Itoa(pv.FileSize))
	// io.Copy(c.Writer, reader)
}
