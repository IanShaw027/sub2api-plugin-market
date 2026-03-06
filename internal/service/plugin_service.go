package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"sync"
	"time"

	"github.com/IanShaw027/sub2api-plugin-market/ent"
	"github.com/IanShaw027/sub2api-plugin-market/internal/repository"
)

type cacheEntry struct {
	data      *ListPluginsResponse
	expiresAt time.Time
}

// PluginService 插件业务逻辑层
type PluginService struct {
	repo  *repository.PluginRepository
	cache sync.Map // key: string (query hash), value: *cacheEntry
	ttl   time.Duration
}

// NewPluginService 创建插件服务
func NewPluginService(repo *repository.PluginRepository) *PluginService {
	return &PluginService{
		repo: repo,
		ttl:  1 * time.Minute,
	}
}

// ListPluginsRequest 插件列表请求
type ListPluginsRequest struct {
	Category   string
	Search     string
	PluginType string
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

	cacheKey := s.cacheKey(req)

	// Check cache
	if entry, ok := s.cache.Load(cacheKey); ok {
		ce := entry.(*cacheEntry)
		if time.Now().Before(ce.expiresAt) {
			return ce.data, nil
		}
		s.cache.Delete(cacheKey)
	}

	// Cache miss - query DB
	offset := (req.Page - 1) * req.PageSize
	plugins, total, err := s.repo.ListPlugins(ctx, req.Category, req.Search, req.PluginType, req.IsOfficial, offset, req.PageSize)
	if err != nil {
		return nil, err
	}

	resp := &ListPluginsResponse{
		Plugins:  plugins,
		Total:    total,
		Page:     req.Page,
		PageSize: req.PageSize,
	}

	// Store in cache
	s.cache.Store(cacheKey, &cacheEntry{
		data:      resp,
		expiresAt: time.Now().Add(s.ttl),
	})

	return resp, nil
}

func (s *PluginService) cacheKey(req *ListPluginsRequest) string {
	raw, _ := json.Marshal(req)
	h := sha256.Sum256(raw)
	return hex.EncodeToString(h[:8]) // 16-char hex key is sufficient
}

// InvalidateCache clears the plugin list cache (call after plugin create/update/delete)
func (s *PluginService) InvalidateCache() {
	s.cache.Range(func(key, _ interface{}) bool {
		s.cache.Delete(key)
		return true
	})
}

// GetPluginDetail 获取插件详情
func (s *PluginService) GetPluginDetail(ctx context.Context, name string) (*ent.Plugin, error) {
	return s.repo.GetPluginByName(ctx, name)
}

// GetPluginVersions 获取插件版本列表
func (s *PluginService) GetPluginVersions(ctx context.Context, name, compatibleWith string) ([]*ent.PluginVersion, error) {
	return s.repo.GetPluginVersions(ctx, name, compatibleWith)
}

// GetPluginVersion 获取指定版本
func (s *PluginService) GetPluginVersion(ctx context.Context, name, version string) (*ent.PluginVersion, error) {
	return s.repo.GetPluginVersion(ctx, name, version)
}
