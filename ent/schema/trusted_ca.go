package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/google/uuid"
)

// TrustedCA — user-imported root certificate authority appended to the system pool
// when building an HTTP transport. Stored as raw PEM so the app stays portable
// (copy DB to another machine, CAs travel with it).
type TrustedCA struct {
	ent.Schema
}

func (TrustedCA) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("id", uuid.UUID{}).Default(uuid.New),
		field.String("label").NotEmpty(),
		field.Text("pem_content").NotEmpty(),
		field.Bool("enabled").Default(true),
		field.Time("created_at").Default(time.Now).Immutable(),
	}
}

func (TrustedCA) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("label"),
	}
}
