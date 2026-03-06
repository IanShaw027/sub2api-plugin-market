package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/google/uuid"
	"time"
)

// PluginVersion holds the schema definition for the PluginVersion entity.
type PluginVersion struct {
	ent.Schema
}

// Fields of the PluginVersion.
func (PluginVersion) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("id", uuid.UUID{}).
			Default(uuid.New).
			Immutable(),
		field.UUID("plugin_id", uuid.UUID{}).
			Comment("关联的插件 ID"),
		field.String("version").
			NotEmpty().
			Comment("语义化版本号"),
		field.Text("changelog").
			Optional().
			Comment("版本更新日志"),
		field.String("wasm_url").
			NotEmpty().
			Comment("WASM 文件存储路径"),
		field.String("wasm_hash").
			NotEmpty().
			Comment("WASM 文件 SHA256 哈希"),
		field.String("signature").
			Optional().
			Comment("Ed25519 签名，draft 未签名版本可为空"),
		field.String("sign_key_id").
			Optional().
			Comment("签名密钥 ID，关联到 TrustKey 表的 key_id"),
		field.Int("file_size").
			Positive().
			Comment("文件大小（字节）"),
		field.String("min_api_version").
			NotEmpty().
			Comment("最低 API 版本要求"),
		field.String("plugin_api_version").
			NotEmpty().
			Comment("插件 API 版本，格式如 1.0.0"),
		field.String("max_api_version").
			Optional().
			Comment("最高 API 版本要求"),
		field.JSON("dependencies", []map[string]string{}).
			Optional().
			Comment("依赖列表"),
		field.JSON("capabilities", []string{}).
			Optional().
			Comment("所需 Host API 能力列表，如 host_http_fetch, host_kv_read"),
		field.Enum("status").
			Values("draft", "published", "yanked").
			Default("draft").
			Comment("版本状态"),
		field.Time("published_at").
			Optional().
			Comment("发布时间"),
		field.Time("created_at").
			Default(time.Now).
			Immutable(),
		field.Time("updated_at").
			Default(time.Now).
			UpdateDefault(time.Now),
	}
}

// Edges of the PluginVersion.
func (PluginVersion) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("plugin", Plugin.Type).
			Ref("versions").
			Field("plugin_id").
			Unique().
			Required(),
		edge.From("submission", Submission.Type).
			Ref("version").
			Unique(),
	}
}

// Indexes of the PluginVersion.
func (PluginVersion) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("plugin_id", "version").Unique(),
		index.Fields("plugin_id", "status", "published_at"),
		index.Fields("status", "published_at"),
	}
}
