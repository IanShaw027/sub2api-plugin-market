package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/google/uuid"
	"time"
)

// Submission holds the schema definition for the Submission entity.
type Submission struct {
	ent.Schema
}

// Fields of the Submission.
func (Submission) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("id", uuid.UUID{}).
			Default(uuid.New).
			Immutable(),
		field.UUID("plugin_id", uuid.UUID{}).
			Comment("关联的插件 ID"),
		field.Enum("submission_type").
			Values("new_plugin", "new_version", "update_metadata").
			Comment("提交类型"),
		field.String("submitter_email").
			NotEmpty().
			Comment("提交者邮箱"),
		field.String("submitter_name").
			NotEmpty().
			Comment("提交者姓名"),
		field.Text("notes").
			Optional().
			Comment("提交说明"),
		field.Enum("source_type").
			Values("upload", "github").
			Default("upload").
			Comment("来源类型"),
		field.String("github_repo_url").
			Optional().
			Comment("GitHub 仓库地址"),
		field.Bool("auto_upgrade_enabled").
			Default(false).
			Comment("是否启用自动升级"),
		field.Enum("status").
			Values("pending", "approved", "rejected", "cancelled").
			Default("pending").
			Comment("审核状态"),
		field.Text("reviewer_notes").
			Optional().
			Comment("审核意见"),
		field.String("reviewed_by").
			Optional().
			Comment("审核人"),
		field.Time("reviewed_at").
			Optional().
			Comment("审核时间"),
		field.Time("created_at").
			Default(time.Now).
			Immutable(),
		field.Time("updated_at").
			Default(time.Now).
			UpdateDefault(time.Now),
	}
}

// Edges of the Submission.
func (Submission) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("plugin", Plugin.Type).
			Ref("submissions").
			Field("plugin_id").
			Unique().
			Required(),
		edge.To("version", PluginVersion.Type).
			Unique(),
	}
}

// Indexes of the Submission.
func (Submission) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("status", "created_at"),
		index.Fields("plugin_id", "status"),
	}
}
