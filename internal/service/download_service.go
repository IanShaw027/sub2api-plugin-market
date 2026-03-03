package service

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/IanShaw027/sub2api-storage"
	"github.com/sub2api/plugin-market/ent"
	"github.com/sub2api/plugin-market/internal/repository"
)

// DownloadService 下载业务逻辑层
type DownloadService struct {
	pluginRepo *repository.PluginRepository
	storage    storage.Storage
	client     *ent.Client
}

// NewDownloadService 创建下载服务
func NewDownloadService(pluginRepo *repository.PluginRepository, storage storage.Storage, client *ent.Client) *DownloadService {
	return &DownloadService{
		pluginRepo: pluginRepo,
		storage:    storage,
		client:     client,
	}
}

// DownloadPlugin 下载插件
func (s *DownloadService) DownloadPlugin(ctx context.Context, pluginName, version string) (*ent.PluginVersion, io.ReadCloser, error) {
	// 获取插件版本信息
	pv, err := s.pluginRepo.GetPluginVersion(ctx, pluginName, version)
	if err != nil {
		return nil, nil, fmt.Errorf("plugin version not found: %w", err)
	}

	// 从 Storage 获取 WASM 文件
	reader, err := s.storage.Download(ctx, pv.WasmURL)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get wasm file: %w", err)
	}

	// 记录下载日志
	if err := s.recordDownloadLog(ctx, pv, "", "", true, ""); err != nil {
		// 日志记录失败不影响下载
		fmt.Printf("failed to record download log: %v\n", err)
	}

	// 增加下载计数
	if err := s.pluginRepo.IncrementDownloadCount(ctx, pv.PluginID.String()); err != nil {
		// 计数失败不影响下载
		fmt.Printf("failed to increment download count: %v\n", err)
	}

	return pv, reader, nil
}

// recordDownloadLog 记录下载日志
func (s *DownloadService) recordDownloadLog(ctx context.Context, pv *ent.PluginVersion, clientIP, userAgent string, success bool, errorMsg string) error {
	_, err := s.client.DownloadLog.Create().
		SetPluginID(pv.PluginID).
		SetVersion(pv.Version).
		SetClientIP(clientIP).
		SetUserAgent(userAgent).
		SetSuccess(success).
		SetNillableErrorMessage(&errorMsg).
		Save(ctx)
	return err
}

// GetDownloadURL 获取下载 URL（预签名 URL）
func (s *DownloadService) GetDownloadURL(ctx context.Context, pluginName, version string) (string, error) {
	// 获取插件版本信息
	pv, err := s.pluginRepo.GetPluginVersion(ctx, pluginName, version)
	if err != nil {
		return "", fmt.Errorf("plugin version not found: %w", err)
	}

	// 生成预签名 URL
	url, err := s.storage.GetPresignedURL(ctx, pv.WasmURL, 15*time.Minute)
	if err != nil {
		return "", fmt.Errorf("failed to generate presigned url: %w", err)
	}

	// 记录下载日志
	if err := s.recordDownloadLog(ctx, pv, "", "", true, ""); err != nil {
		fmt.Printf("failed to record download log: %v\n", err)
	}

	// 增加下载计数
	if err := s.pluginRepo.IncrementDownloadCount(ctx, pv.PluginID.String()); err != nil {
		fmt.Printf("failed to increment download count: %v\n", err)
	}

	return url, nil
}
