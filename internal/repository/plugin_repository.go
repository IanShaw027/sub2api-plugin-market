package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/IanShaw027/sub2api-plugin-market/ent"
	"github.com/IanShaw027/sub2api-plugin-market/ent/plugin"
	"github.com/IanShaw027/sub2api-plugin-market/ent/pluginversion"
)

// PluginRepository 插件数据访问层
type PluginRepository struct {
	client *ent.Client
}

// NewPluginRepository 创建插件仓库
func NewPluginRepository(client *ent.Client) *PluginRepository {
	return &PluginRepository{client: client}
}

// ListPlugins 查询插件列表
func (r *PluginRepository) ListPlugins(ctx context.Context, category, search, pluginType string, isOfficial *bool, offset, limit int) ([]*ent.Plugin, int, error) {
	query := r.client.Plugin.Query().
		Where(plugin.StatusEQ(plugin.StatusActive))

	// 分类过滤
	if category != "" {
		query = query.Where(plugin.CategoryEQ(plugin.Category(category)))
	}

	// 插件类型过滤
	if pluginType != "" {
		query = query.Where(plugin.PluginTypeEQ(plugin.PluginType(pluginType)))
	}

	// 官方插件过滤
	if isOfficial != nil {
		query = query.Where(plugin.IsOfficialEQ(*isOfficial))
	}

	// 搜索过滤
	if search != "" {
		query = query.Where(
			plugin.Or(
				plugin.NameContains(search),
				plugin.DisplayNameContains(search),
				plugin.DescriptionContains(search),
			),
		)
	}

	// 获取总数
	total, err := query.Count(ctx)
	if err != nil {
		return nil, 0, err
	}

	// 排序：官方插件优先，下载量降序
	plugins, err := query.
		Order(ent.Desc(plugin.FieldIsOfficial), ent.Desc(plugin.FieldDownloadCount)).
		Offset(offset).
		Limit(limit).
		All(ctx)

	return plugins, total, err
}

// GetPluginByName 根据名称获取插件详情
func (r *PluginRepository) GetPluginByName(ctx context.Context, name string) (*ent.Plugin, error) {
	return r.client.Plugin.Query().
		Where(
			plugin.NameEQ(name),
			plugin.StatusEQ(plugin.StatusActive),
		).
		WithVersions(func(q *ent.PluginVersionQuery) {
			q.Where(pluginversion.StatusEQ(pluginversion.StatusPublished)).
				Order(ent.Desc(pluginversion.FieldPublishedAt))
		}).
		Only(ctx)
}

// GetPluginVersions 获取插件的所有版本
// compatibleWith 非空时，仅返回 min_api_version <= compatibleWith 的版本。
func (r *PluginRepository) GetPluginVersions(ctx context.Context, pluginName, compatibleWith string) ([]*ent.PluginVersion, error) {
	p, err := r.client.Plugin.Query().
		Where(
			plugin.NameEQ(pluginName),
			plugin.StatusEQ(plugin.StatusActive),
		).
		Only(ctx)
	if err != nil {
		return nil, err
	}

	query := r.client.PluginVersion.Query().
		Where(
			pluginversion.PluginIDEQ(p.ID),
			pluginversion.StatusEQ(pluginversion.StatusPublished),
		)

	if compatibleWith != "" {
		query = query.Where(pluginversion.MinAPIVersionLTE(compatibleWith))
	}

	return query.
		Order(ent.Desc(pluginversion.FieldPublishedAt)).
		All(ctx)
}

// GetPluginVersion 获取指定版本
func (r *PluginRepository) GetPluginVersion(ctx context.Context, pluginName, version string) (*ent.PluginVersion, error) {
	p, err := r.client.Plugin.Query().
		Where(
			plugin.NameEQ(pluginName),
			plugin.StatusEQ(plugin.StatusActive),
		).
		Only(ctx)
	if err != nil {
		return nil, err
	}

	return r.client.PluginVersion.Query().
		Where(
			pluginversion.PluginIDEQ(p.ID),
			pluginversion.VersionEQ(version),
			pluginversion.StatusEQ(pluginversion.StatusPublished),
		).
		Only(ctx)
}

// IncrementDownloadCount 增加下载计数
func (r *PluginRepository) IncrementDownloadCount(ctx context.Context, pluginID string) error {
	id, err := uuid.Parse(pluginID)
	if err != nil {
		return err
	}

	// 使用原子操作增加计数
	return r.client.Plugin.UpdateOneID(id).
		AddDownloadCount(1).
		Exec(ctx)
}
