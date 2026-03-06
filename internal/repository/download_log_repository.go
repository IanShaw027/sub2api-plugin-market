package repository

import (
	"context"
	"strings"

	"github.com/IanShaw027/sub2api-plugin-market/ent"
	"github.com/google/uuid"
)

// DownloadLogRepository 下载日志数据访问层
type DownloadLogRepository struct {
	client *ent.Client
}

// NewDownloadLogRepository 创建下载日志仓库
func NewDownloadLogRepository(client *ent.Client) *DownloadLogRepository {
	return &DownloadLogRepository{client: client}
}

// Create 创建下载日志
// clientIP 应为已哈希处理的客户端 IP
func (r *DownloadLogRepository) Create(ctx context.Context, pluginID uuid.UUID, version, clientIP, userAgent string, success bool, errorMsg string) error {
	userAgent = strings.TrimSpace(userAgent)
	errorMsg = strings.TrimSpace(errorMsg)

	create := r.client.DownloadLog.Create().
		SetPluginID(pluginID).
		SetVersion(version).
		SetClientIP(clientIP).
		SetSuccess(success)

	if userAgent != "" {
		create = create.SetNillableUserAgent(&userAgent)
	}
	if errorMsg != "" {
		create = create.SetNillableErrorMessage(&errorMsg)
	}

	_, err := create.Save(ctx)
	return err
}
