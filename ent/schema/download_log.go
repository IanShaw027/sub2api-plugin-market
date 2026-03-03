package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/google/uuid"
	"time"
)

// DownloadLog holds the schema definition for the DownloadLog entity.
type DownloadLog struct {
	ent.Schema
}

// Fields of the DownloadLog.
func (DownloadLog) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("id", uuid.UUID{}).
			Default(uuid.New).
			Immutable(),
		field.UUID("plugin_id", uuid.UUID{}).
			Comment("关联的插件 ID"),
		field.String("version").
			NotEmpty().
			Comment("下载的版本号"),
		field.String("client_ip").
			NotEmpty().
			Comment("客户端 IP（哈希处理）"),
		field.String("user_agent").
			Optional().
			Comment("用户代理"),
		field.String("country_code").
			Optional().
			MaxLen(2).
			Comment("国家代码（ISO 3166-1 alpha-2）"),
		field.Bool("success").
			Default(true).
			Comment("是否下载成功"),
		field.String("error_message").
			Optional().
			Comment("错误信息"),
		field.Time("downloaded_at").
			Default(time.Now).
			Immutable().
			Comment("下载时间"),
	}
}

// Edges of the DownloadLog.
func (DownloadLog) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("plugin", Plugin.Type).
			Ref("download_logs").
			Field("plugin_id").
			Unique().
			Required(),
	}
}

// Indexes of the DownloadLog.
func (DownloadLog) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("plugin_id", "downloaded_at"),
		index.Fields("downloaded_at"),
		index.Fields("success", "downloaded_at"),
	}
}
