package thunderstorm

import "context"

type ActionHandler func(ctx context.Context, args ...interface{}) error
type FilterHandler func(value interface{}, args ...interface{}) (interface{}, error)

type listener struct {
	priority int
	name     string
	handler  interface{} // ActionHandler or FilterHandler
}

// Listener represents a registered event listener (for debugging)
type Listener struct {
	Priority int
	Name     string
	Type     string // "action" or "filter"
}

// HookNotFoundError when trying to access non-existent hook
type HookNotFoundError struct {
	Hook string
}

func (e HookNotFoundError) Error() string {
	return "hook not found: " + e.Hook
}
