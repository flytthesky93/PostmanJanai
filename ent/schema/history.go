package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"time"
)

// History holds the schema definition for the History entity.
type History struct {
	ent.Schema
}

// Fields of the History.
func (History) Fields() []ent.Field {
	return []ent.Field{
		field.String("method").Default("GET"),
		field.Text("url"),
		field.Int("status_code"),
		field.Text("request_body").Optional(),
		field.Text("response_body").Optional(),
		field.Time("created_at").Default(time.Now),
	}
}

// Edges of the History.
func (History) Edges() []ent.Edge {
	return nil
}
