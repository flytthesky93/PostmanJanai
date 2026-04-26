package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/google/uuid"
)

// RequestCapture — post-response capture rule attached to a saved Request (Phase 8).
//
// source values:
//
//	"json_body"  — JSONPath against the response body parsed as JSON.
//	"header"     — case-insensitive response header lookup; expression = header name.
//	"status"     — HTTP status code (expression ignored).
//	"regex_body" — Go regexp against the raw response body; capture group 1 wins (or full match).
//
// target_scope:
//
//	"environment" — write the captured value into the active environment as a plain variable.
//	"memory"      — write to per-run in-memory bag (used during runner only).
type RequestCapture struct {
	ent.Schema
}

func (RequestCapture) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("id", uuid.UUID{}).Default(uuid.New),
		field.UUID("request_id", uuid.UUID{}),
		field.String("name").NotEmpty(),
		field.String("source").NotEmpty(),
		field.Text("expression").Default(""),
		field.String("target_scope").Default("environment"),
		field.String("target_variable").NotEmpty(),
		field.Bool("enabled").Default(true),
		field.Int("sort_order").Default(0),
		field.Time("created_at").Default(time.Now).Immutable(),
		field.Time("updated_at").Default(time.Now).UpdateDefault(time.Now),
	}
}

func (RequestCapture) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("request", Request.Type).
			Ref("request_captures").
			Field("request_id").
			Unique().
			Required(),
	}
}

func (RequestCapture) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("request_id", "sort_order"),
	}
}
