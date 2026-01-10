package delivery

import (
	"context"
	"fmt"
)

// App struct
type AppHandler struct {
	ctx context.Context
}

// NewApp creates a new App application struct
func NewAppHandler() *AppHandler {
	return &AppHandler{}
}

// startup is called when the app starts. The context is saved
// so we can call the runtime methods
func (a *AppHandler) SetContext(ctx context.Context) {
	a.ctx = ctx
}

// Greet returns a greeting for the given name
func (a *AppHandler) Greet(name string) string {
	return fmt.Sprintf("Hello %s, It's show time!", name)
}
