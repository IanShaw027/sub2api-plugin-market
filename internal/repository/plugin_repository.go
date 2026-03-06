package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/IanShaw027/sub2api-plugin-market/ent"
	"github.com/IanShaw027/sub2api-plugin-market/ent/plugin"
	"github.com/IanShaw027/sub2api-plugin-market/ent/pluginversion"
	"github.com/IanShaw027/sub2api-plugin-market/ent/predicate"
	entsql "entgo.io/ent/dialect/sql"
)

// PluginRepository 插件数据访问层
type PluginRepository struct {
	client    *ent.Client
	listCache *TTLCache
}

// NewPluginRepository 创建插件仓库
func NewPluginRepository(client *ent.Client) *PluginRepository {
	return &PluginRepository{
		client:    client,
		listCache: NewTTLCache(3 * time.Minute),
	}
}

// InvalidateCache clears the list cache. Call after plugin create/update/delete.
func (r *PluginRepository) InvalidateCache() {
	r.listCache.Invalidate()
}

type listPluginsResult struct {
	plugins []*ent.Plugin
	total   int
}

// ListPlugins 查询插件列表
func (r *PluginRepository) ListPlugins(ctx context.Context, category, search, pluginType string, isOfficial *bool, offset, limit int) ([]*ent.Plugin, int, error) {
	officialStr := ""
	if isOfficial != nil {
		officialStr = fmt.Sprintf("%v", *isOfficial)
	}
	cacheKey := fmt.Sprintf("list:%s:%s:%s:%s:%d:%d", category, search, pluginType, officialStr, offset, limit)

	if cached, ok := r.listCache.Get(cacheKey); ok {
		res := cached.(*listPluginsResult)
		return res.plugins, res.total, nil
	}

	query := r.client.Plugin.Query().
		Where(plugin.StatusEQ(plugin.StatusActive))

	if category != "" {
		query = query.Where(plugin.CategoryEQ(plugin.Category(category)))
	}

	if pluginType != "" {
		query = query.Where(plugin.PluginTypeEQ(plugin.PluginType(pluginType)))
	}

	if isOfficial != nil {
		query = query.Where(plugin.IsOfficialEQ(*isOfficial))
	}

	if search != "" {
		query = query.Where(
			plugin.Or(
				plugin.NameContainsFold(search),
				plugin.DisplayNameContainsFold(search),
				plugin.DescriptionContainsFold(search),
				tagsContainFold(search),
			),
		)
	}

	total, err := query.Count(ctx)
	if err != nil {
		return nil, 0, err
	}

	plugins, err := query.
		Order(ent.Desc(plugin.FieldIsOfficial), ent.Desc(plugin.FieldDownloadCount)).
		Offset(offset).
		Limit(limit).
		All(ctx)
	if err != nil {
		return nil, 0, err
	}

	r.listCache.Set(cacheKey, &listPluginsResult{plugins: plugins, total: total})
	return plugins, total, err
}

// tagsContainFold matches plugins whose JSON tags array contains the search
// string (case-insensitive). Uses PostgreSQL cast + ILIKE.
func tagsContainFold(search string) predicate.Plugin {
	return func(s *entsql.Selector) {
		s.Where(entsql.P(func(b *entsql.Builder) {
			b.WriteString("tags::text ILIKE ")
			b.Arg("%" + search + "%")
		}))
	}
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

	allVersions, err := query.
		Order(ent.Desc(pluginversion.FieldPublishedAt)).
		All(ctx)
	if err != nil {
		return nil, err
	}

	if compatibleWith == "" {
		return allVersions, nil
	}

	var filtered []*ent.PluginVersion
	for _, v := range allVersions {
		if isVersionCompatible(v.MinAPIVersion, v.MaxAPIVersion, compatibleWith) {
			filtered = append(filtered, v)
		}
	}
	return filtered, nil
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

	err = r.client.Plugin.UpdateOneID(id).
		AddDownloadCount(1).
		Exec(ctx)
	if err == nil {
		r.listCache.Invalidate()
	}
	return err
}
