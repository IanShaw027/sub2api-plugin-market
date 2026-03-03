package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/google/uuid"
	"time"
)

// Plugin holds the schema definition for the Plugin entity.
type Plugin struct {
	ent.Schema
}

// Fields of the Plugin.
func (Plugin) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("id", uuid.UUID{}).
			Default(uuid.New).
			Immutable(),
		field.String("name").
			Unique().
			NotEmpty().
			Comment("插件唯一标识符"),
		field.String("display_name").
			NotEmpty().
			Comment("插件显示名称"),
		field.Text("description").
			Comment("插件描述"),
		field.String("author").
			NotEmpty().
			Comment("作者名称"),
		field.String("repository_url").
			Optional().
			Comment("代码仓库地址"),
		field.String("homepage_url").
			Optional().
			Comment("主页地址"),
		field.String("license").
			Default("MIT").
			Comment("开源协议"),
		field.Enum("category").
			Values("proxy", "auth", "analytics", "security", "other").
			Default("other").
			Comment("插件分类"),
		field.JSON("tags", []string{}).
			Optional().
			Comment("标签列表"),
		field.Bool("is_official").
			Default(false).
			Comment("是否官方插件"),
		field.Bool("is_verified").
			Default(false).
			Comment("是否已验证"),
		field.Int("download_count").
			Default(0).
			NonNegative().
			Comment("下载次数"),
		field.Float("rating").
			Optional().
			Min(0).
			Max(5).
			Comment("评分"),
		field.Enum("status").
			Values("active", "deprecated", "suspended").
			Default("active").
			Comment("插件状态"),
		field.Time("created_at").
			Default(time.Now).
			Immutable(),
		field.Time("updated_at").
			Default(time.Now).
			UpdateDefault(time.Now),
	}
}

// Edges of the Plugin.
func (Plugin) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("versions", PluginVersion.Type),
		edge.To("submissions", Submission.Type),
		edge.To("download_logs", DownloadLog.Type),
	}
}

// Indexes of the Plugin.
func (Plugin) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("name").Unique(),
		index.Fields("is_official", "status"),
		index.Fields("category", "status"),
		index.Fields("is_official", "status", "download_count"),
	}
}
