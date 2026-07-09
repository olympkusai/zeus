# Thunderstorm - Event Bus

**Problem**: Coupling business logic creates brittle systems. When user registers, you need to send email, log event, update analytics, create welcome sequence. Hardcoding these creates spaghetti.

**Solution**: WordPress-style event bus (Actions/Filters) that decouples event producers from consumers. Register listeners that fire when events happen.

## When to Use

- **Multi-step workflows** - E.g., "when order placed" → invoice, email, analytics, fulfillment
- **Plugin architectures** - Let plugins react to core events without modifying core
- **Data transformation** - Run values through filter chains (sanitize, format, validate)
- **Async processing** - Trigger background jobs from synchronous code
- **Logging/monitoring** - Inject cross-cutting concerns without modifying business logic

## What It Solves

### ❌ Without Event Bus (Tightly Coupled)
```go
func CreateUser(email, name string) error {
    user, err := db.InsertUser(email, name)
    if err != nil {
        return err
    }
    
    // Hardcoded: email service
    email.SendWelcome(user)
    
    // Hardcoded: analytics
    analytics.Track("user_created", user.ID)
    
    // Hardcoded: onboarding
    onboarding.CreateFlow(user.ID)
    
    return nil
}
// Hard to test. Hard to disable/enable features. Hard to add new reactions.
```

### ✅ With Event Bus (Decoupled)
```go
func CreateUser(email, name string) error {
    user, err := db.InsertUser(email, name)
    if err != nil {
        return err
    }
    
    // Fire event, let listeners react
    bus.DoAction(ctx, "user:created", user)
    return nil
}

// Elsewhere, registered once:
bus.AddAction("user:created", 10, func(ctx context.Context, args ...interface{}) error {
    user := args[0].(*User)
    return email.SendWelcome(user)  // Email listener
})

bus.AddAction("user:created", 20, func(ctx context.Context, args ...interface{}) error {
    user := args[0].(*User)
    return analytics.Track("user_created", user.ID)  // Analytics listener
})
```

## Features

- ⚡ **Actions** - Fire events, run handlers sequentially
- 🔄 **Filters** - Transform data through handler chains
- 🎯 **Priorities** - Control execution order (lower = earlier)
- 🧹 **Panic Recovery** - One handler crashes, others continue
- 🔒 **Thread-safe** - Safe for concurrent listeners
- 🪝 **Full Control** - Add/remove handlers at runtime
- 📊 **Debugging** - Inspect registered listeners

## Installation

```bash
go get github.com/olympkusai/thunderstorm
```

## Quick Start

### 1. Create Bus

```go
import "github.com/olympkusai/thunderstorm"

bus := thunderstorm.New()
ctx := context.Background()
```

### 2. Actions (Fire-and-Forget)

```go
// Register action
bus.AddAction("order:placed", 10, func(ctx context.Context, args ...interface{}) error {
    orderID := args[0].(uint)
    log.Printf("Order %d placed", orderID)
    return email.SendConfirmation(orderID)
})

// Fire action
err := bus.DoAction(ctx, "order:placed", 42)
// Calls all "order:placed" handlers in priority order
```

### 3. Filters (Data Transformation)

```go
// Register filters
bus.AddFilter("post:title", 5, func(value interface{}, args ...interface{}) (interface{}, error) {
    title := value.(string)
    return strings.ToLower(title), nil
})

bus.AddFilter("post:title", 10, func(value interface{}, args ...interface{}) (interface{}, error) {
    title := value.(string)
    return strings.TrimSpace(title), nil
})

// Apply filters
result, _ := bus.ApplyFilters(ctx, "post:title", "  HELLO WORLD  ")
// Result: "hello world" (lowercased, trimmed)
```

## Real-World Examples

### E-Commerce: Order Workflow

```go
// Step 1: Register listeners (on startup)
bus.AddAction("order:created", 10, func(ctx context.Context, args ...interface{}) error {
    order := args[0].(*Order)
    return sendOrderConfirmationEmail(order)
})

bus.AddAction("order:created", 20, func(ctx context.Context, args ...interface{}) error {
    order := args[0].(*Order)
    return warehouse.QueuePick(order.ID)  // Fulfillment
})

bus.AddAction("order:created", 30, func(ctx context.Context, args ...interface{}) error {
    order := args[0].(*Order)
    return analytics.TrackPurchase(order)
})

// Step 2: Create order (calls all listeners)
func CreateOrder(items []Item, user User) error {
    order, err := db.InsertOrder(user.ID, items)
    if err != nil {
        return err
    }
    
    // Fire event → email + fulfillment + analytics
    return bus.DoAction(ctx, "order:created", order)
}
```

### Content Moderation: Filter Chain

```go
// Clean user comments: lowercase → trim → remove profanity → escape HTML
bus.AddFilter("comment:content", 10, func(value interface{}, args ...interface{}) (interface{}, error) {
    return strings.ToLower(value.(string)), nil
})

bus.AddFilter("comment:content", 20, func(value interface{}, args ...interface{}) (interface{}, error) {
    return strings.TrimSpace(value.(string)), nil
})

bus.AddFilter("comment:content", 30, func(value interface{}, args ...interface{}) (interface{}, error) {
    content := value.(string)
    return profanity.Remove(content), nil
})

bus.AddFilter("comment:content", 40, func(value interface{}, args ...interface{}) (interface{}, error) {
    return html.EscapeString(value.(string)), nil
})

// Apply all filters
cleaned, _ := bus.ApplyFilters(ctx, "comment:content", userInput)
// Input:  "  HeLLo@#$!  "
// Output: "hello@#$!" (safe to store)
```

### Plugin System: Extensibility

```go
// Core app fires events
bus.DoAction(ctx, "app:startup")       // Let plugins initialize
bus.DoAction(ctx, "request:received", request)  // Let plugins process
bus.DoAction(ctx, "response:sending", response) // Let plugins modify

// Plugin 1: Logging
bus.AddAction("request:received", 10, func(ctx context.Context, args ...interface{}) error {
    req := args[0]
    log.Printf("Request: %v", req)
    return nil
})

// Plugin 2: Authentication
bus.AddAction("request:received", 5, func(ctx context.Context, args ...interface{}) error {
    req := args[0]
    return auth.Verify(req)
})

// Plugin 3: Rate limiting
bus.AddAction("request:received", 15, func(ctx context.Context, args ...interface{}) error {
    req := args[0]
    return ratelimit.Check(req)
})
```

## API Reference

### Actions

```go
// Register action
bus.AddAction(hook string, priority int, handler ActionHandler)

// Fire action (calls all handlers in priority order)
err := bus.DoAction(ctx context.Context, hook string, args ...interface{}) error

// Check if hook has listeners
exists := bus.HasAction(hook string) bool

// Remove specific handler
removed := bus.RemoveAction(hook string, handler ActionHandler) bool

// Remove all handlers
count := bus.RemoveAllActions(hook string) int

// Debug: list all listeners
listeners := bus.GetActionListeners(hook string) []Listener
```

### Filters

```go
// Register filter
bus.AddFilter(hook string, priority int, handler FilterHandler)

// Apply all filters to value
result, err := bus.ApplyFilters(ctx, hook string, value interface{}, args ...interface{}) (interface{}, error)

// Check if hook has filters
exists := bus.HasFilter(hook string) bool

// Remove specific filter
removed := bus.RemoveFilter(hook string, handler FilterHandler) bool

// Remove all filters
count := bus.RemoveAllFilters(hook string) int

// Debug: list all filters
filters := bus.GetFilterListeners(hook string) []Listener
```

## Execution Model

### Actions: Sequential Fire

```
bus.DoAction("user:created", user)

Priority 5  → Handler 1 ✓
Priority 10 → Handler 2 ✓
Priority 15 → Handler 3 ✗ (error, but others still run)
Priority 20 → Handler 4 ✓

Returns: error from Handler 3 (first error)
```

### Filters: Sequential Transformation

```
bus.ApplyFilters("comment:content", "HELLO")

Priority 5  → Lowercase    → "hello"
Priority 10 → TrimSpace    → "hello"
Priority 15 → RemoveBadWords → "hello"
Priority 20 → HTMLEscape   → "hello"

Returns: "hello"
```

## Best Practices

### ✅ Do

```go
// Use specific hook names
bus.AddAction("user:created", ...)
bus.AddAction("user:updated", ...)
bus.AddAction("user:deleted", ...)

// Keep handlers focused and fast
bus.AddAction("order:created", 10, func(...) error {
    return sendEmail(order)  // One job
})

// Use priorities to control order
bus.AddAction("log", 5, ...)      // Log first
bus.AddAction("validate", 10, ...) // Validate second
bus.AddAction("execute", 20, ...)  // Execute third
```

### ❌ Don't

```go
// Don't do everything in one handler
bus.AddAction("user:created", 10, func(...) error {
    sendEmail()
    updateAnalytics()
    createWorkflow()
    notifySlack()
    // Too much, hard to test, hard to disable
})

// Don't block on slow operations (consider async)
bus.AddAction("user:created", 10, func(...) error {
    return slowThirdPartyAPI.Sync()  // Will block
})
```

## Performance

- **Actions**: O(n) where n = number of handlers
- **Filters**: O(n) where n = number of filters
- **Handler execution**: Sequential (one at a time)
- **Memory**: O(n) stored listeners per hook

For high-throughput systems, consider async workers:
```go
bus.AddAction("event", 10, func(ctx context.Context, args ...interface{}) error {
    queue.Enqueue("async_job", args)  // Fire and forget
    return nil
})
```

## Status

✅ Production-ready  
✅ Thread-safe  
✅ Zero dependencies  
✅ Fully tested

## Next Steps

- See `integration_example.go` for setup with database events
- Check `.specs/INTEGRACAO_ADB_THUNDERSTORM.md` for ADB integration
- Start with Actions, add Filters as needed
