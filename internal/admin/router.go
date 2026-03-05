package admin

import (
	"github.com/IanShaw027/sub2api-plugin-market/internal/admin/handler"
	"github.com/IanShaw027/sub2api-plugin-market/internal/admin/middleware"
	"github.com/IanShaw027/sub2api-plugin-market/internal/auth"
	"github.com/gin-gonic/gin"
)

// RegisterRoutes 注册管理后台路由
func RegisterRoutes(r *gin.Engine, authHandler *handler.AuthHandler, submissionHandler *handler.SubmissionHandler, jwtService *auth.JWTService, adminService *auth.AdminService) {
	admin := r.Group("/admin")
	{
		// 静态文件
		admin.Static("/js", "./web/admin/js")
		admin.Static("/css", "./web/admin/css")
		admin.Static("/assets", "./web/assets")
		admin.StaticFile("/", "./web/admin/index.html")
		admin.StaticFile("/login", "./web/admin/login.html")

		// API 路由
		api := admin.Group("/api")
		{
			// 认证接口（无需 token）
			api.POST("/auth/login", authHandler.Login)
			api.POST("/auth/refresh", authHandler.Refresh)

			// 需要认证的接口
			authorized := api.Group("")
			authorized.Use(middleware.AdminAuth(jwtService, adminService))
			{
				// 认证相关
				authorized.GET("/auth/me", authHandler.GetMe)
				authorized.POST("/auth/logout", authHandler.Logout)

				// 审核管理
				authorized.GET("/submissions", submissionHandler.List)
				authorized.GET("/submissions/:id", submissionHandler.Get)
				authorized.PUT("/submissions/:id/review", submissionHandler.Review)
				authorized.GET("/submissions/stats", submissionHandler.Stats)
			}
		}
	}
}
