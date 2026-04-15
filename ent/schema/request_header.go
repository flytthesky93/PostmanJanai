package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/google/uuid"
)

type RequestHeader struct {
	ent.Schema
}

func (RequestHeader) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("id", uuid.UUID{}).Default(uuid.New),
		field.UUID("request_id", uuid.UUID{}),
		field.String("key").NotEmpty(),
		field.Text("value").NotEmpty(),
		field.Bool("enabled").Default(true),
		field.Int("sort_order").Default(0),
	}
}

func (RequestHeader) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("request", Request.Type).
			Ref("request_headers").
			Field("request_id").
			Unique().
			Required(),
	}
}

func (RequestHeader) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("request_id", "sort_order"),
	}
}
