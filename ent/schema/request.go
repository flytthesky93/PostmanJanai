package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/google/uuid"
)

// Request — saved HTTP request; scoped to exactly one folder.
type Request struct {
	ent.Schema
}

func (Request) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("id", uuid.UUID{}).Default(uuid.New),
		field.UUID("folder_id", uuid.UUID{}),
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
		edge.From("folder", Folder.Type).
			Ref("requests").
			Field("folder_id").
			Unique().
			Required(),
		edge.To("request_headers", RequestHeader.Type),
		edge.To("request_query_params", RequestQueryParam.Type),
		edge.To("request_form_fields", RequestFormField.Type),
		edge.To("histories", History.Type),
	}
}

func (Request) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("folder_id", "name").Unique(),
	}
}
