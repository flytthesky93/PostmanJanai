package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/google/uuid"
)

// RunnerRun — metadata for one folder runner execution (Phase 8).
//
// status values:
//
//	"running"   — in-flight (set on insert).
//	"completed" — finished; passed_count + failed_count == total_count.
//	"failed"    — terminated early due to error or cancel; counts may be partial.
//	"cancelled" — user-cancelled; counts may be partial.
type RunnerRun struct {
	ent.Schema
}

func (RunnerRun) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("id", uuid.UUID{}).Default(uuid.New),
		field.UUID("folder_id", uuid.UUID{}).Optional().Nillable(),
		field.UUID("environment_id", uuid.UUID{}).Optional().Nillable(),
		field.String("folder_name").Default(""),
		field.String("environment_name").Default(""),
		field.String("status").Default("running"),
		field.Int("total_count").Default(0),
		field.Int("passed_count").Default(0),
		field.Int("failed_count").Default(0),
		field.Int("error_count").Default(0),
		field.Int("duration_ms").Default(0),
		field.Text("notes").Default(""),
		field.Time("started_at").Default(time.Now).Immutable(),
		field.Time("finished_at").Optional().Nillable(),
	}
}

func (RunnerRun) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("requests", RunnerRunRequest.Type),
	}
}

func (RunnerRun) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("started_at"),
	}
}
