package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/google/uuid"
)

// RunnerRunRequest — per-request result inside a RunnerRun (Phase 8).
//
// `status` reflects the row outcome rather than the HTTP status:
//
//	"passed"  — all enabled assertions passed (or none defined).
//	"failed"  — at least one assertion failed.
//	"errored" — transport / validation error before assertions could run.
//	"skipped" — request was skipped (disabled, etc.).
type RunnerRunRequest struct {
	ent.Schema
}

func (RunnerRunRequest) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("id", uuid.UUID{}).Default(uuid.New),
		field.UUID("run_id", uuid.UUID{}),
		field.UUID("request_id", uuid.UUID{}).Optional().Nillable(),
		field.String("request_name").Default(""),
		field.String("method").Default("GET"),
		field.Text("url").Default(""),
		field.String("status").Default("passed"),
		field.Int("status_code").Default(0),
		field.Int("duration_ms").Default(0),
		field.Int("response_size_bytes").Default(0),
		field.Text("error_message").Default(""),
		field.Text("assertions_json").Default(""),
		field.Text("captures_json").Default(""),
		// Phase 8.1 — store the resolved (post-substitution) request snapshot
		// and the response payload so the user can review what was actually
		// sent/received without re-running the request.
		field.Text("request_headers_json").Optional().Nillable(),
		field.Text("response_headers_json").Optional().Nillable(),
		field.Text("request_body").Optional().Nillable(),
		field.Text("response_body").Optional().Nillable(),
		field.Bool("body_truncated").Default(false),
		field.Int("sort_order").Default(0),
		field.Time("created_at").Default(time.Now).Immutable(),
	}
}

func (RunnerRunRequest) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("run", RunnerRun.Type).
			Ref("requests").
			Field("run_id").
			Unique().
			Required(),
	}
}

func (RunnerRunRequest) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("run_id", "sort_order"),
	}
}
