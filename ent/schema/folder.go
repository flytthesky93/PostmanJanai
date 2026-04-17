package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/google/uuid"
)

// Folder replaces workspace + collection: nested folders; requests live in a folder.
type Folder struct {
	ent.Schema
}

func (Folder) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("id", uuid.UUID{}).Default(uuid.New),
		field.UUID("parent_id", uuid.UUID{}).Optional().Nillable(),
		field.String("name").NotEmpty(),
		field.String("description").Default(""),
		field.Time("created_at").Default(time.Now).Immutable(),
	}
}

func (Folder) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("children", Folder.Type),
		edge.From("parent", Folder.Type).
			Ref("children").
			Field("parent_id").
			Unique(),
		edge.To("requests", Request.Type),
		edge.To("histories", History.Type),
	}
}

func (Folder) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("parent_id", "name").Unique(),
	}
}
