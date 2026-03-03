package v1

import (
	"github.com/gin-gonic/gin"
	"github.com/IanShaw027/sub2api-plugin-market/internal/api/v1/handler"
	"github.com/IanShaw027/sub2api-plugin-market/internal/api/v1/middleware"
)

// RegisterRoutes 注册 v1 版本的所有路由
func RegisterRoutes(r *gin.Engine,
	pluginHandler *handler.PluginHandler,
	downloadHandler *handler.DownloadHandler,
	trustKeyHandler *handler.TrustKeyHandler) {

	// 注册全局中间件
	r.Use(middleware.Recovery())
	r.Use(middleware.Logger())
	r.Use(middleware.CORS())

	v1 := r.Group("/api/v1")
	{
		// 插件相关路由
		plugins := v1.Group("/plugins")
		{
			plugins.GET("", pluginHandler.ListPlugins)
			plugins.GET("/:name", pluginHandler.GetPluginDetail)
			plugins.GET("/:name/versions", pluginHandler.GetPluginVersions)
			plugins.GET("/:name/versions/:version/download", downloadHandler.DownloadPlugin)
		}

		// 信任密钥相关路由
		trustKeys := v1.Group("/trust-keys")
		{
			trustKeys.GET("", trustKeyHandler.ListTrustKeys)
			trustKeys.GET("/:key_id", trustKeyHandler.GetTrustKeyDetail)
		}
	}
}
