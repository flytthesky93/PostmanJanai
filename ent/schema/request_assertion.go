package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/google/uuid"
)

// RequestAssertion — post-response assertion rule attached to a saved Request (Phase 8).
//
// source values mirror RequestCapture plus duration/size:
//
//	"status"             — HTTP status code; expression ignored.
//	"header"             — header name in `expression`.
//	"json_body"          — JSONPath in `expression`.
//	"regex_body"         — regex pattern in `expression`.
//	"duration_ms"        — total request duration in ms; expression ignored.
//	"response_size_bytes"— response body byte count; expression ignored.
//
// operator values:
//
//	"eq" / "neq"         — string or numeric equality.
//	"contains" / "not_contains".
//	"gt" / "lt" / "gte" / "lte" — numeric comparison.
//	"regex"              — `expected` is a regex matched against the actual value.
//	"exists" / "not_exists" — only the JSONPath / header has to resolve.
type RequestAssertion struct {
	ent.Schema
}

func (RequestAssertion) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("id", uuid.UUID{}).Default(uuid.New),
		field.UUID("request_id", uuid.UUID{}),
		field.String("name").NotEmpty(),
		field.String("source").NotEmpty(),
		field.Text("expression").Default(""),
		field.String("operator").NotEmpty(),
		field.Text("expected").Default(""),
		field.Bool("enabled").Default(true),
		field.Int("sort_order").Default(0),
		field.Time("created_at").Default(time.Now).Immutable(),
		field.Time("updated_at").Default(time.Now).UpdateDefault(time.Now),
	}
}

func (RequestAssertion) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("request", Request.Type).
			Ref("request_assertions").
			Field("request_id").
			Unique().
			Required(),
	}
}

func (RequestAssertion) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("request_id", "sort_order"),
	}
}
