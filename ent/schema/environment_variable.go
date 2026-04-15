package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/google/uuid"
)

type EnvironmentVariable struct {
	ent.Schema
}

func (EnvironmentVariable) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("id", uuid.UUID{}).Default(uuid.New),
		field.UUID("environment_id", uuid.UUID{}),
		field.String("key").NotEmpty(),
		field.Text("value").NotEmpty(),
		field.Bool("enabled").Default(true),
		field.Int("sort_order").Default(0),
		field.Time("created_at").Default(time.Now).Immutable(),
		field.Time("updated_at").Default(time.Now).UpdateDefault(time.Now),
	}
}

func (EnvironmentVariable) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("environment", Environment.Type).
			Ref("environment_variables").
			Field("environment_id").
			Unique().
			Required(),
	}
}

func (EnvironmentVariable) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("environment_id", "key").Unique(),
		index.Fields("environment_id", "sort_order"),
	}
}
