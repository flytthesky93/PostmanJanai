package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"github.com/google/uuid"
)

type History struct {
	ent.Schema
}

func (History) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("id", uuid.UUID{}).Default(uuid.New),
		field.UUID("workspace_id", uuid.UUID{}).Optional().Nillable(),
		field.UUID("request_id", uuid.UUID{}).Optional().Nillable(),
		field.String("method").NotEmpty(),
		field.Text("url").NotEmpty(),
		field.Int("status_code"),
		field.Int("duration_ms").Optional().Nillable(),
		field.Int("response_size_bytes").Optional().Nillable(),
		field.Text("request_headers_json").Optional().Nillable(),
		field.Text("response_headers_json").Optional().Nillable(),
		field.Text("request_body").Optional().Nillable(),
		field.Text("response_body").Optional().Nillable(),
		field.Time("created_at").Default(time.Now).Immutable(),
	}
}

func (History) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("workspace", Workspace.Type).
			Ref("histories").
			Field("workspace_id").
			Unique(),
		edge.From("request", Request.Type).
			Ref("histories").
			Field("request_id").
			Unique(),
	}
}
