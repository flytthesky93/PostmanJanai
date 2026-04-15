package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/google/uuid"
)

type Environment struct {
	ent.Schema
}

func (Environment) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("id", uuid.UUID{}).Default(uuid.New),
		field.String("name").NotEmpty(),
		field.String("description").Default(""),
		field.Bool("is_active").Default(false),
		field.Time("created_at").Default(time.Now).Immutable(),
		field.Time("updated_at").Default(time.Now).UpdateDefault(time.Now),
	}
}

func (Environment) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("environment_variables", EnvironmentVariable.Type),
	}
}

func (Environment) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("name").Unique(),
	}
}
