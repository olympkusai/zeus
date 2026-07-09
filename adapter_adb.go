package thunderstorm

import (
	"context"
)

// ADBAdapter adapts EventBus to adb.EventDispatcher
type ADBAdapter struct {
	bus *EventBus
}

// NewADBAdapter creates a new adapter
func NewADBAdapter(bus *EventBus) *ADBAdapter {
	return &ADBAdapter{bus: bus}
}

// OnBeforeCreate fires before_create hook
func (a *ADBAdapter) OnBeforeCreate(ctx context.Context, table string, model interface{}) error {
	hook := table + ":before_create"
	return a.bus.DoAction(ctx, hook, model)
}

// OnAfterCreate fires created hook
func (a *ADBAdapter) OnAfterCreate(ctx context.Context, table string, id uint, model interface{}) error {
	hook := table + ":created"
	return a.bus.DoAction(ctx, hook, id, model)
}

// OnBeforeUpdate fires before_update hook
func (a *ADBAdapter) OnBeforeUpdate(ctx context.Context, table string, model interface{}) error {
	hook := table + ":before_update"
	return a.bus.DoAction(ctx, hook, model)
}

// OnAfterUpdate fires updated hook
func (a *ADBAdapter) OnAfterUpdate(ctx context.Context, table string, model interface{}) error {
	hook := table + ":updated"
	return a.bus.DoAction(ctx, hook, model)
}

// OnBeforeDelete fires before_delete hook
func (a *ADBAdapter) OnBeforeDelete(ctx context.Context, table string, id uint) error {
	hook := table + ":before_delete"
	return a.bus.DoAction(ctx, hook, id)
}

// OnAfterDelete fires deleted hook
func (a *ADBAdapter) OnAfterDelete(ctx context.Context, table string, id uint) error {
	hook := table + ":deleted"
	return a.bus.DoAction(ctx, hook, id)
}
