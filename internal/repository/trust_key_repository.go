package repository

import (
	"context"

	"github.com/IanShaw027/sub2api-plugin-market/ent"
	"github.com/IanShaw027/sub2api-plugin-market/ent/trustkey"
)

// TrustKeyRepository 信任密钥数据访问层
type TrustKeyRepository struct {
	client *ent.Client
}

// NewTrustKeyRepository 创建信任密钥仓库
func NewTrustKeyRepository(client *ent.Client) *TrustKeyRepository {
	return &TrustKeyRepository{client: client}
}

// ListTrustKeys 查询信任密钥列表
func (r *TrustKeyRepository) ListTrustKeys(ctx context.Context, keyType string, isActive *bool) ([]*ent.TrustKey, error) {
	query := r.client.TrustKey.Query()

	// 密钥类型过滤
	if keyType != "" {
		query = query.Where(trustkey.KeyTypeEQ(trustkey.KeyType(keyType)))
	}

	// 激活状态过滤
	if isActive != nil {
		query = query.Where(trustkey.IsActiveEQ(*isActive))
	}

	return query.
		Order(ent.Desc(trustkey.FieldCreatedAt)).
		All(ctx)
}

// GetTrustKeyByKeyID 根据 key_id 获取密钥详情
func (r *TrustKeyRepository) GetTrustKeyByKeyID(ctx context.Context, keyID string) (*ent.TrustKey, error) {
	return r.client.TrustKey.Query().
		Where(trustkey.KeyIDEQ(keyID)).
		Only(ctx)
}

// ListActiveTrustKeys 获取所有激活的信任密钥
func (r *TrustKeyRepository) ListActiveTrustKeys(ctx context.Context) ([]*ent.TrustKey, error) {
	return r.client.TrustKey.Query().
		Where(trustkey.IsActiveEQ(true)).
		All(ctx)
}
