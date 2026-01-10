package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"time"
)

// Workspace holds the schema definition for the Workspace entity.
type Workspace struct {
	ent.Schema
}

// Fields of the Workspace.
func (Workspace) Fields() []ent.Field {
	return []ent.Field{
		field.String("workspace_name"),
		field.String("workspace_description"),
		field.Time("created_at").Default(time.Now),
	}
}

// Edges of the Workspace.
func (Workspace) Edges() []ent.Edge {
	return nil
}
