# Functional Composition Pattern

## Overview

This codebase uses a **Functional Composition** pattern for its core
interfaces. This pattern decomposes individual methods from single-
and multi-method interface types into individual function types, then
recomposes them into a concrete implementations, which may be sealed
(or not), and exported (or not) depending on the context.

This approach provides flexibility, testability, and safe interface
evolution.

## Core Principles

### 1. Decompose Interfaces into Function Types

Instead of implementing interfaces directly on structs, we create function types for each method:

```go
// Interface definition
type RateReservation interface {
    WaitTime() time.Duration
    Cancel()
}

// Function types for each method
type WaitTimeFunc func() time.Duration
type CancelFunc func()

// Function types implement their corresponding methods
func (f WaitTimeFunc) WaitTime() time.Duration {
    if f == nil {
        return 0
    }
    return f()
}

func (f CancelFunc) Cancel() {
    if f == nil {
        return
    }
    f()
}
```

Note that each function type always implements no-op functionality,
such that passing `nil` corresponds with the no-op implementation for
a that function.

### 2. Compose Functions into Interface Implementations

Create concrete implementations embed the corresponding function type:

```go  
type rateReservationImpl struct {
    WaitTimeFunc
    CancelFunc
}

// Constructor for an instance
func NewRateReservationImpl(wf WaitTimeFunc, cf CancelFunc) RateReservation {
    return rateReservationImpl{
        WaitTimeFunc: wf,
        CancelFunc:   cf,
    }
}
```

Note that because the argument to NewXyzImpl() is a list of
`<...>Func` objects, and the Go compiler automatically converts func
values, so we can write:

```go
    return NewRateReservationImpl(
        // Wait time 1 second
        func() time.Duration { return time.Second },
	// Cancel is a no-op.
	nil,
    )
```

For constant values and enumerated types, define a `Self()` method to
act as the corresponding function implementation: 

```go
// Self returns itself.
func (t Type) Self() Type {
     return t
}

// TypeFunc is ...
type TypeFunc func() Type

// Type gets the type of the component created by this factory.
func (f TypeFunc) Type() Type {
	if f == nil {
	}
	return f()
}
```

For example, we can decompose, modify, and recompose a
`component.Factory`:

```go
    // Construct a factory with a new default Config:
    factory := sometype.NewFactory()
    cfg := factory.CreateDefaultConfig()
    // modify the config object
    return NewFactoryImpl(factory.Type().Self, cfg.Self)
```

### 3. Use Constructors for Interface Values

Provide constructor functions rather than exposing concrete types:

```go
// Good: Constructor function
func NewRateLimiterImpl(f ReserveRateFunc) RateLimiter {
    return rateLimiterImpl{ReserveRateFunc: f}
}

// Avoid: Direct struct instantiation
// rateLimiterImpl{ReserveRateFunc: f} // Don't do this
```

Rarely should the implementation type be exposed. This pattern can be
combined with the Functional Option pattern (from `receiver/receiver.go`):

```go
// Setup optional logging-related functions
func WithLogs(createLogs CreateLogsFunc, sl component.StabilityLevel) FactoryOption {
	return factoryOptionFunc(func(o *factoryImpl, cfgType component.Type) {
		o.CreateLogsFunc = createLogs
		o.LogsStabilityFunc = sl.Self
	})
}

// Accept options to configure various aspects of the interface
func NewFactory(cfgType component.Type, createDefaultConfig component.CreateDefaultConfigFunc, options ...FactoryOption) Factory {
	f := factoryImpl{
		Factory: component.NewFactoryImpl(cfgType.Self, createDefaultConfig),
	}
	for _, opt := range options {
		opt.applyOption(&f, cfgType)
	}
	return f
}
```

## Rationale

### Flexibility and Composition

This pattern enables composition scenarios by making it easy to
compose and decompose interface values.

```go
// Transform existing factories with cross-cutting concerns
func NewLimitedFactory(fact receiver.Factory, cfgf LimiterConfigurator) receiver.Factory {
    return receiver.NewFactoryImpl(
        fact.Type,
        fact.CreateDefaultConfig,
        receiver.CreateTracesFunc(limitReceiver(fact.CreateTraces, traceTraits{}, cfgf)),
        fact.TracesStability,
        receiver.CreateMetricsFunc(limitReceiver(fact.CreateMetrics, metricTraits{}, cfgf)),
        fact.MetricsStability,
        receiver.CreateLogsFunc(limitReceiver(fact.CreateLogs, logTraits{}, cfgf)),
        fact.LogsStability,
    )
}
```

This is sometimes called aspect-oriented programming, for example to
add logging to an existing function:

```go
func addLogging(f ReserveRateFunc) ReserveRateFunc {
    return func(ctx context.Context, n int) (RateReservation, error) {
        log.Printf("Reserving rate for %d", n)
        return f(ctx, n)
    }
}
```

### Safe Interface Evolution

Using a private method allows sealing the interface type, which forces
external users to use functional constructor methods. This allows
interfaces to evolve safely because users are forced to use
constructor functions.

```go
type RateLimiter interface {
    ReserveRate(context.Context, int) (RateReservation, error)

    // Can add new methods without breaking existing code
    private() // Prevents external implementations
}
```

### Enhanced Testability

Individual methods can be tested independently:

```go
func TestWaitTimeFunction(t *testing.T) {
    waitFn := WaitTimeFunc(func() time.Duration { return time.Second })
    assert.Equal(t, time.Second, waitFn.WaitTime())
}
```

## Implementation Guidelines

### 1. Interface Design

- Define interfaces with clear method signatures
- Include a `private()` method if you need to control implementations
- Keep interfaces focused and cohesive

### 2. Function Type Naming

- Use `<MethodName>Func` naming convention
- Ensure function signatures match the interface method exactly
- Always implement the corresponding interface method on the function type

### 3. Nil Handling

Always handle nil function types gracefully to act as a no-op implementation.

```go
func (f ReserveRateFunc) ReserveRate(ctx context.Context, value int) (RateReservation, error) {
    if f == nil {
        return NewRateReservationImpl(nil, nil), nil // No-op implementation
    }
    return f(ctx, value)
}
```

### 4. Constructor Patterns

Follow consistent constructor naming for building an implementation
of each interface:

```go
// For interfaces: New<InterfaceName>Impl
func NewRateLimiterImpl(f ReserveRateFunc) RateLimiter
```

Follow consistent type naming for each method-function type:

```go
// For function types: <MethodName>Func
type ReserveRateFunc func(context.Context, int) (RateReservation, error)
```

### 5. Implementation Structs

- Use unexported names for implementation structs
- Embed function types directly
- Implement private methods for sealing the interface

```go
type RateLimiter interface {
     // ...

     // Must use functional constructors outside this package
     private()
}

type rateLimiterImpl struct {
    ReserveRateFunc
}

func (rateLimiterImpl) private() {}
```

## When to Use This Pattern

### Appropriate Use Cases

- **Interfaces with single or multiple methods** that benefit from composition
- **Cross-cutting concerns** that need to be applied selectively
- **Interface evolution** where you need to add methods over time
- **Factory patterns** where you're assembling behavior from components
- **Middleware/decorator scenarios** where you're transforming behavior

### When to Avoid

- **Stateful objects** where methods need to share significant state
- **Performance-critical code** where function call overhead matters
- **Simple implementations** where the pattern adds unnecessary complexity
