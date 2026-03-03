package service

import (
	"context"

	"github.com/IanShaw027/sub2api-plugin-market/ent"
	"github.com/IanShaw027/sub2api-plugin-market/internal/repository"
)

// TrustKeyService 信任密钥业务逻辑层
type TrustKeyService struct {
	repo *repository.TrustKeyRepository
}

// NewTrustKeyService 创建信任密钥服务
func NewTrustKeyService(repo *repository.TrustKeyRepository) *TrustKeyService {
	return &TrustKeyService{repo: repo}
}

// ListTrustKeysRequest 信任密钥列表请求
type ListTrustKeysRequest struct {
	KeyType  string
	IsActive *bool
}

// ListTrustKeys 查询信任密钥列表
func (s *TrustKeyService) ListTrustKeys(ctx context.Context, req *ListTrustKeysRequest) ([]*ent.TrustKey, error) {
	return s.repo.ListTrustKeys(ctx, req.KeyType, req.IsActive)
}

// GetTrustKeyDetail 获取信任密钥详情
func (s *TrustKeyService) GetTrustKeyDetail(ctx context.Context, keyID string) (*ent.TrustKey, error) {
	return s.repo.GetTrustKeyByKeyID(ctx, keyID)
}
