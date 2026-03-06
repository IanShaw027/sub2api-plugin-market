package schema

import (
	"regexp"
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/google/uuid"
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
			Match(regexp.MustCompile(`^[a-z0-9][a-z0-9-]{0,62}[a-z0-9]$`)).
			Comment("插件唯一标识符，仅允许小写字母、数字和连字符，长度 2-64"),
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
		field.Enum("plugin_type").
			Values("interceptor", "transform", "provider").
			Optional().
			Comment("插件类型，对应 DispatchRuntime 的三个执行阶段"),
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
		field.Enum("source_type").
			Values("upload", "github").
			Default("upload").
			Comment("来源类型"),
		field.String("github_repo_url").
			Optional().
			Comment("GitHub 仓库地址"),
		field.String("github_repo_normalized").
			Optional().
			Comment("Normalized GitHub repo URL for index lookup"),
		field.Bool("auto_upgrade_enabled").
			Default(false).
			Comment("是否启用自动升级"),
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
		edge.To("sync_jobs", SyncJob.Type),
	}
}

// Indexes of the Plugin.
func (Plugin) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("name").Unique(),
		index.Fields("is_official", "status"),
		index.Fields("category", "status"),
		index.Fields("is_official", "status", "download_count"),
		index.Fields("github_repo_normalized"),
		index.Fields("plugin_type", "status"),
	}
}
