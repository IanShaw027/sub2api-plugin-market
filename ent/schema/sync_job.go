package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/google/uuid"
	"time"
)

// SyncJob holds the schema definition for the SyncJob entity.
type SyncJob struct {
	ent.Schema
}

// Fields of the SyncJob.
func (SyncJob) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("id", uuid.UUID{}).
			Default(uuid.New).
			Immutable(),
		field.UUID("plugin_id", uuid.UUID{}).
			Comment("关联的插件 ID"),
		field.Enum("trigger_type").
			Values("manual", "auto").
			Default("manual").
			Comment("触发类型"),
		field.Enum("status").
			Values("pending", "running", "succeeded", "failed", "cancelled").
			Default("pending").
			Comment("同步任务状态"),
		field.String("target_ref").
			Optional().
			Comment("目标引用，如分支或标签"),
		field.Text("error_message").
			Optional().
			Comment("错误信息"),
		field.Time("started_at").
			Optional().
			Comment("开始时间"),
		field.Time("finished_at").
			Optional().
			Comment("结束时间"),
		field.Time("created_at").
			Default(time.Now).
			Immutable(),
		field.Time("updated_at").
			Default(time.Now).
			UpdateDefault(time.Now),
	}
}

// Edges of the SyncJob.
func (SyncJob) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("plugin", Plugin.Type).
			Ref("sync_jobs").
			Field("plugin_id").
			Unique().
			Required(),
	}
}

// Indexes of the SyncJob.
func (SyncJob) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("plugin_id", "created_at"),
		index.Fields("status", "created_at"),
		index.Fields("trigger_type", "created_at"),
		index.Fields("plugin_id", "status", "trigger_type", "created_at"),
	}
}
