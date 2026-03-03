package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/google/uuid"
	"time"
)

// TrustKey holds the schema definition for the TrustKey entity.
type TrustKey struct {
	ent.Schema
}

// Fields of the TrustKey.
func (TrustKey) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("id", uuid.UUID{}).
			Default(uuid.New).
			Immutable(),
		field.String("key_id").
			Unique().
			NotEmpty().
			Comment("密钥短标识符"),
		field.String("public_key").
			NotEmpty().
			Comment("Ed25519 公钥（base64 编码）"),
		field.Enum("key_type").
			Values("official", "verified_publisher", "community").
			Default("community").
			Comment("密钥类型"),
		field.String("owner_name").
			NotEmpty().
			Comment("所有者姓名"),
		field.String("owner_email").
			NotEmpty().
			Comment("所有者邮箱"),
		field.Text("description").
			Optional().
			Comment("密钥描述"),
		field.Bool("is_active").
			Default(true).
			Comment("是否激活"),
		field.Time("expires_at").
			Optional().
			Comment("过期时间"),
		field.Time("created_at").
			Default(time.Now).
			Immutable(),
		field.Time("updated_at").
			Default(time.Now).
			UpdateDefault(time.Now),
	}
}

// Indexes of the TrustKey.
func (TrustKey) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("key_id").Unique(),
		index.Fields("key_type", "is_active"),
		index.Fields("is_active", "expires_at"),
	}
}
