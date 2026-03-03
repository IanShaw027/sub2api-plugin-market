package service

import (
	"context"

	"github.com/sub2api/plugin-market/ent"
	"github.com/sub2api/plugin-market/internal/repository"
)

// PluginService 插件业务逻辑层
type PluginService struct {
	repo *repository.PluginRepository
}

// NewPluginService 创建插件服务
func NewPluginService(repo *repository.PluginRepository) *PluginService {
	return &PluginService{repo: repo}
}

// ListPluginsRequest 插件列表请求
type ListPluginsRequest struct {
	Category   string
	Search     string
	IsOfficial *bool
	Page       int
	PageSize   int
}

// ListPluginsResponse 插件列表响应
type ListPluginsResponse struct {
	Plugins []*ent.Plugin `json:"plugins"`
	Total   int           `json:"total"`
	Page    int           `json:"page"`
	PageSize int          `json:"page_size"`
}

// ListPlugins 查询插件列表
func (s *PluginService) ListPlugins(ctx context.Context, req *ListPluginsRequest) (*ListPluginsResponse, error) {
	// 参数校验
	if req.Page < 1 {
		req.Page = 1
	}
	if req.PageSize < 1 || req.PageSize > 100 {
		req.PageSize = 20
	}

	offset := (req.Page - 1) * req.PageSize

	plugins, total, err := s.repo.ListPlugins(ctx, req.Category, req.Search, req.IsOfficial, offset, req.PageSize)
	if err != nil {
		return nil, err
	}

	return &ListPluginsResponse{
		Plugins:  plugins,
		Total:    total,
		Page:     req.Page,
		PageSize: req.PageSize,
	}, nil
}

// GetPluginDetail 获取插件详情
func (s *PluginService) GetPluginDetail(ctx context.Context, name string) (*ent.Plugin, error) {
	return s.repo.GetPluginByName(ctx, name)
}

// GetPluginVersions 获取插件版本列表
func (s *PluginService) GetPluginVersions(ctx context.Context, name string) ([]*ent.PluginVersion, error) {
	return s.repo.GetPluginVersions(ctx, name)
}

// GetPluginVersion 获取指定版本
func (s *PluginService) GetPluginVersion(ctx context.Context, name, version string) (*ent.PluginVersion, error) {
	return s.repo.GetPluginVersion(ctx, name, version)
}
