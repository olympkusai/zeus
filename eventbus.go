package thunderstorm

import (
	"context"
	"fmt"
	"log"
	"sort"
	"sync"
)

// EventBus manages hooks, actions, and filters
type EventBus struct {
	actions       map[string][]listener
	filters       map[string][]listener
	mu            sync.RWMutex
	sortedActions map[string]bool // track if actions are sorted
	sortedFilters map[string]bool // track if filters are sorted
}

// New creates a new event bus
func New() *EventBus {
	return &EventBus{
		actions:       make(map[string][]listener),
		filters:       make(map[string][]listener),
		sortedActions: make(map[string]bool),
		sortedFilters: make(map[string]bool),
	}
}

// AddAction registers an action listener
func (eb *EventBus) AddAction(hook string, priority int, handler ActionHandler) {
	eb.mu.Lock()
	defer eb.mu.Unlock()

	name := fmt.Sprintf("%T", handler) // func pointer as name
	eb.actions[hook] = append(eb.actions[hook], listener{
		priority: priority,
		name:     name,
		handler:  handler,
	})

	eb.sortedActions[hook] = false // mark as unsorted
}

// AddFilter registers a filter listener
func (eb *EventBus) AddFilter(hook string, priority int, handler FilterHandler) {
	eb.mu.Lock()
	defer eb.mu.Unlock()

	name := fmt.Sprintf("%T", handler)
	eb.filters[hook] = append(eb.filters[hook], listener{
		priority: priority,
		name:     name,
		handler:  handler,
	})

	eb.sortedFilters[hook] = false // mark as unsorted
}

// DoAction fires an action
func (eb *EventBus) DoAction(ctx context.Context, hook string, args ...interface{}) error {
	eb.mu.RLock()
	listeners, exists := eb.actions[hook]
	eb.mu.RUnlock()

	if !exists || len(listeners) == 0 {
		return nil
	}

	// Sort if not already sorted
	eb.ensureSorted(hook, "action")

	eb.mu.RLock()
	sortedListeners := eb.actions[hook]
	eb.mu.RUnlock()

	var errs []error

	for _, l := range sortedListeners {
		handler := l.handler.(ActionHandler)

		// Recover from panics
		func() {
			defer func() {
				if r := recover(); r != nil {
					err := fmt.Errorf("panic in listener: %v", r)
					errs = append(errs, err)
					log.Printf("Thunderstorm: %v", err)
				}
			}()

			if err := handler(ctx, args...); err != nil {
				errs = append(errs, fmt.Errorf("listener %s: %w", l.name, err))
			}
		}()
	}

	if len(errs) > 0 {
		return fmt.Errorf("action %s had %d errors", hook, len(errs))
	}

	return nil
}

// ApplyFilters applies all filters to a value
func (eb *EventBus) ApplyFilters(ctx context.Context, hook string, value interface{}, args ...interface{}) (interface{}, error) {
	eb.mu.RLock()
	listeners, exists := eb.filters[hook]
	eb.mu.RUnlock()

	if !exists || len(listeners) == 0 {
		return value, nil
	}

	// Sort if not already sorted
	eb.ensureSorted(hook, "filter")

	eb.mu.RLock()
	sortedListeners := eb.filters[hook]
	eb.mu.RUnlock()

	var errs []error
	result := value

	for _, l := range sortedListeners {
		handler := l.handler.(FilterHandler)

		func() {
			defer func() {
				if r := recover(); r != nil {
					err := fmt.Errorf("panic in filter: %v", r)
					errs = append(errs, err)
					log.Printf("Thunderstorm: %v", err)
					// Don't update result on panic
				}
			}()

			filtered, err := handler(result, args...)
			if err != nil {
				errs = append(errs, fmt.Errorf("filter %s: %w", l.name, err))
				// Keep previous value if error
				return
			}

			result = filtered
		}()
	}

	if len(errs) > 0 {
		return result, fmt.Errorf("filter %s had %d errors", hook, len(errs))
	}

	return result, nil
}

// RemoveAction removes specific action listener
func (eb *EventBus) RemoveAction(hook string, target ActionHandler) bool {
	eb.mu.Lock()
	defer eb.mu.Unlock()

	listeners, exists := eb.actions[hook]
	if !exists {
		return false
	}

	targetPtr := fmt.Sprintf("%p", target)
	for i, l := range listeners {
		if fmt.Sprintf("%p", l.handler) == targetPtr {
			eb.actions[hook] = append(listeners[:i], listeners[i+1:]...)
			eb.sortedActions[hook] = false // mark as unsorted
			return true
		}
	}

	return false
}

// RemoveFilter removes specific filter listener
func (eb *EventBus) RemoveFilter(hook string, target FilterHandler) bool {
	eb.mu.Lock()
	defer eb.mu.Unlock()

	listeners, exists := eb.filters[hook]
	if !exists {
		return false
	}

	targetPtr := fmt.Sprintf("%p", target)
	for i, l := range listeners {
		if fmt.Sprintf("%p", l.handler) == targetPtr {
			eb.filters[hook] = append(listeners[:i], listeners[i+1:]...)
			eb.sortedFilters[hook] = false // mark as unsorted
			return true
		}
	}

	return false
}

// RemoveAllActions removes all listeners for a hook
func (eb *EventBus) RemoveAllActions(hook string) int {
	eb.mu.Lock()
	defer eb.mu.Unlock()

	count := len(eb.actions[hook])
	delete(eb.actions, hook)
	delete(eb.sortedActions, hook)
	return count
}

// RemoveAllFilters removes all listeners for a hook
func (eb *EventBus) RemoveAllFilters(hook string) int {
	eb.mu.Lock()
	defer eb.mu.Unlock()

	count := len(eb.filters[hook])
	delete(eb.filters, hook)
	delete(eb.sortedFilters, hook)
	return count
}

// HasAction checks if hook has listeners
func (eb *EventBus) HasAction(hook string) bool {
	eb.mu.RLock()
	defer eb.mu.RUnlock()

	listeners, exists := eb.actions[hook]
	return exists && len(listeners) > 0
}

// HasFilter checks if hook has listeners
func (eb *EventBus) HasFilter(hook string) bool {
	eb.mu.RLock()
	defer eb.mu.RUnlock()

	listeners, exists := eb.filters[hook]
	return exists && len(listeners) > 0
}

// GetActionListeners returns all listeners for an action hook
func (eb *EventBus) GetActionListeners(hook string) []Listener {
	eb.mu.RLock()
	defer eb.mu.RUnlock()

	listeners, exists := eb.actions[hook]
	if !exists {
		return []Listener{}
	}

	result := make([]Listener, len(listeners))
	for i, l := range listeners {
		result[i] = Listener{
			Priority: l.priority,
			Name:     l.name,
			Type:     "action",
		}
	}

	return result
}

// GetFilterListeners returns all listeners for a filter hook
func (eb *EventBus) GetFilterListeners(hook string) []Listener {
	eb.mu.RLock()
	defer eb.mu.RUnlock()

	listeners, exists := eb.filters[hook]
	if !exists {
		return []Listener{}
	}

	result := make([]Listener, len(listeners))
	for i, l := range listeners {
		result[i] = Listener{
			Priority: l.priority,
			Name:     l.name,
			Type:     "filter",
		}
	}

	return result
}

// ensureSorted sorts listeners by priority if not already sorted
func (eb *EventBus) ensureSorted(hook string, hookType string) {
	eb.mu.Lock()
	defer eb.mu.Unlock()

	if hookType == "action" {
		if !eb.sortedActions[hook] {
			sort.Slice(eb.actions[hook], func(i, j int) bool {
				return eb.actions[hook][i].priority < eb.actions[hook][j].priority
			})
			eb.sortedActions[hook] = true
		}
	} else {
		if !eb.sortedFilters[hook] {
			sort.Slice(eb.filters[hook], func(i, j int) bool {
				return eb.filters[hook][i].priority < eb.filters[hook][j].priority
			})
			eb.sortedFilters[hook] = true
		}
	}
}
