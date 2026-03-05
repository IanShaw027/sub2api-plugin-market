package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/google/uuid"
)

// AdminUser holds the schema definition for the AdminUser entity.
type AdminUser struct {
	ent.Schema
}

// Fields of the AdminUser.
func (AdminUser) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("id", uuid.UUID{}).
			Default(uuid.New).
			StorageKey("id").
			Unique().
			Immutable(),
		field.String("username").
			NotEmpty().
			Unique().
			Comment("管理员用户名"),
		field.String("email").
			NotEmpty().
			Unique().
			Comment("管理员邮箱"),
		field.String("password_hash").
			NotEmpty().
			Sensitive().
			Comment("密码哈希"),
		field.Enum("role").
			Values("super_admin", "admin", "reviewer").
			Default("reviewer").
			Comment("角色：super_admin=超级管理员, admin=管理员, reviewer=审核员"),
		field.Bool("is_active").
			Default(true).
			Comment("是否激活"),
		field.Time("last_login_at").
			Optional().
			Nillable().
			Comment("最后登录时间"),
		field.Time("created_at").
			Default(time.Now).
			Immutable().
			Comment("创建时间"),
		field.Time("updated_at").
			Default(time.Now).
			UpdateDefault(time.Now).
			Comment("更新时间"),
	}
}

// Indexes of the AdminUser.
func (AdminUser) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("username").Unique(),
		index.Fields("email").Unique(),
		index.Fields("is_active"),
	}
}
