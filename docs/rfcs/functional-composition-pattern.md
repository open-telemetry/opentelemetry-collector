# Functional Composition Pattern

## Overview

This codebase uses a **Functional Composition** pattern for its core
interfaces. This pattern decomposes individual methods from interface
types into corresponding function types, then recomposes them into a
concrete implementation type.  This approach provides flexibility,
testability, and safe interface evolution for the OpenTelemetry
Collector.

When an interface type is exported for users outside of this
repository, the type MUST follow these guidelines. Interface types
exposed in internal packages may decide to export internal interfaces,
of course.

For every method in the public interface, a corresponding `type
<Method>Func func(...) ...` declaration in the same package will
exist, having the matching signature.

For every interface type, there is a corresponding functional
constructor `func New<Type>(<Method1>Func, <Method2>Func, ...) Type`
in the same package for constructing a functional composition of
interface methods.

Interface stability for exported interface types is our primary
objective. The Functional Composition pattern supports safe interface
evolution, first by "sealing" the type with an unexported interface
method. This means all implementations of an interface must use
constructors provided in the package.

These "sealed concrete" implementation objects support adding new
methods in future releases, _without changing the major version
number_, because public interface types are always provided through a
package-provided implementation. As a key requirement, every function
must have a simple "no-op" implementation correspodning with the zero
value of the `<Method>Func`.

Additional methods may be provided using the widely-known [Functional
Option pattern](https://commandcenter.blogspot.com/2014/01/self-referential-functions-and-design.html).

## Key concepts

### 1. Decompose Interfaces into Function Types

Instead of implementing interfaces directly on structs, we create
function types for each method:

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

Users of the `net/http` package have seen this pattern before. The
`http.HandlerFunc` type can be seen as the prototype for functional
composition, in this case for HTTP handlers. Interestingly, the
single-method `http.RoundTripper` interface, which represents the same
interaction on the client-side, does not have a `RoundTripperFunc`. In
this codebase, the pattern is applied exstensively.

### 2. Compose Functions into Interface Implementations

Create concrete implementations embed the corresponding function type:

```go  
type rateReservationImpl struct {
    WaitTimeFunc
    CancelFunc
}

// Constructor for an instance
func NewRateReservation(wf WaitTimeFunc, cf CancelFunc) RateReservation {
    return rateReservationImpl{
        WaitTimeFunc: wf,
        CancelFunc:   cf,
    }
}
```

[The Go language automatically converts function literal
expressions](https://go.dev/doc/effective_go#conversions) into the
correct named type (i.e., `<Method>Func`), so we can write:

```go
    return NewRateReservation(
        // Wait time 1 second
        func() time.Duration { return time.Second },
	// Cancel is a no-op.
	nil,
    )
```

### 3. Use Constructors for Interface Values

Provide constructor functions rather than exposing concrete types:

```go
// Good: Constructor function
func NewRateLimiter(f ReserveRateFunc) RateLimiter {
    return rateLimiterImpl{ReserveRateFunc: f}
}

// Avoid: Direct struct instantiation inside the package
// rateLimiterImpl{ReserveRateFunc: f} // Don't do this, use the constructor
```

This will help maintainers upgrade your callsites, should the
interface gain a new method.  For more complicated interfaces, this
pattern can be combined with the Functional Option pattern in Golang,
shown in the next example.

Taken from `receiver/receiver.go`, here we setup signal-specific
functions using a functional-option argument passed to
`receiver.NewFactory`:

```go
// Setup optional signal-specific functions (e.g., logs)
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

### 4. Constant-value Function Implementations

For types defined by simple values, especially for enumerated types,
define a `Self()` method to act as the corresponding functional
constant:

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
    // ... Modify the config object
    // Pass cfg.Self as the default config function.
    return NewFactoryImpl(factory.Type().Self, cfg.Self)
```

## Rationale

### Flexibility and Composition

This pattern enables composition scenarios by making it easy to
compose and decompose interface values. For example, to wrap a
`receiver.Factory` with a limiter of some sort:

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

This is sometimes called aspect-oriented programming, for example it
is easy to add logging to an existing function:

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
constructor functions, and instead of breaking stability for function
definitions, we can add alternative constructors.

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
