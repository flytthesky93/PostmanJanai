package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/google/uuid"
)

type Workspace struct {
	ent.Schema
}

func (Workspace) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("id", uuid.UUID{}).Default(uuid.New),
		field.String("workspace_name").NotEmpty(),
		field.String("workspace_description").Default(""),
		field.Time("created_at").Default(time.Now).Immutable(),
	}
}

func (Workspace) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("collections", Collection.Type),
		edge.To("requests", Request.Type),
		edge.To("histories", History.Type),
	}
}

func (Workspace) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("workspace_name").Unique(),
	}
}
