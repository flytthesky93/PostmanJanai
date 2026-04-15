package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"github.com/google/uuid"
)

// Request — tài nguyên request (root workspace hoặc trong collection).
type Request struct {
	ent.Schema
}

func (Request) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("id", uuid.UUID{}).Default(uuid.New),
		field.UUID("workspace_id", uuid.UUID{}),
		field.UUID("collection_id", uuid.UUID{}).Optional().Nillable(),
		field.String("name").NotEmpty(),
		field.String("method").Default("GET"),
		field.Text("url").NotEmpty(),
		field.String("body_mode").NotEmpty(),
		field.Text("raw_body").Optional().Nillable(),
		field.Time("created_at").Default(time.Now).Immutable(),
		field.Time("updated_at").Default(time.Now).UpdateDefault(time.Now),
	}
}

func (Request) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("workspace", Workspace.Type).
			Ref("requests").
			Field("workspace_id").
			Unique().
			Required(),
		edge.From("collection", Collection.Type).
			Ref("requests").
			Field("collection_id").
			Unique(),
		edge.To("request_headers", RequestHeader.Type),
		edge.To("request_query_params", RequestQueryParam.Type),
		edge.To("request_form_fields", RequestFormField.Type),
		edge.To("histories", History.Type),
	}
}
