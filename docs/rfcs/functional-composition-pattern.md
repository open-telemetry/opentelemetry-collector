# Functional Composition Pattern

## Overview

This codebase uses a **Functional Composition** pattern for its core
interfaces. This pattern decomposes the individual methods from
interface types into corresponding function types, then recomposes
them into a concrete implementation type. This approach provides
flexibility, testability, and safe interface evolution for the
OpenTelemetry Collector.

When an interface type is exported for users outside of this
repository, the type MUST follow these guidelines. Interface types
exposed from internal packages may opt-out of this recommendation.

For every method in the public interface, a corresponding `type
<Method>Func func(...) ...` declaration in the same package will
exist, having the matching signature.

For every interface type, there is a corresponding functional
constructor to enable a functional composition of interface methods,
`func New<Type>(<Method1>Func, {...}, ...Option) Type` which accepts
the initially-required method set and uses a Functional Option pattern
for forwards compatibility, even when there are no options at the
start.

Interface stability for exported interface types is our primary
objective. The Functional Composition pattern supports safe interface
evolution, first by "sealing" the type with an unexported interface
method. This means all implementations of an interface must embed or
use constructors provided in the package.

These "sealed concrete" implementation objects support adding new
methods in future releases, _without changing the major version
number_, because public interface types are always provided through a
package-provided implementation.

As a key requirement, every function must have a simple "no-op"
behavior corresponding with the zero value of the `<Method>Func`. The
expression `New<Type>(nil, nil, ...)` is the "empty" do-nothing
implementation for each type.

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
        return 0 // No-op behavior
    }
    return f()
}

func (f CancelFunc) Cancel() {
    if f == nil {
        return // No-op behavior
    }
    f()
}
```

Users of the [`net/http` package have seen this
pattern](https://pkg.go.dev/net/http#HandlerFunc). `http.HandlerFunc`
can be seen as a prototype for the Functional Composition pattern, in
this case for HTTP servers. Interestingly, the single-method
`http.RoundTripper` interface, representing the same interaction for
HTTP clients, does not have a `RoundTripperFunc` in the base library
(consequently, this codebase defines [it for testing middleware
extensions](https://github.com/open-telemetry/opentelemetry-collector/blob/64088871efb1b873c3d53ed3b7f0ce7140c0d7e2/extension/extensionmiddleware/extensionmiddlewaretest/nop.go#L23)). In
this codebase, the pattern is applied extensively.

### 2. Compose Function Types into Interface Implementations

Create concrete implementations embedding the function type
corresponding with each interface method:

```go
// A struct embedding a <Method>Func for each method.
type rateReservationImpl struct {
    WaitTimeFunc
    CancelFunc
}
```

This pattern applies even for single-method interfaces, where the
`<Method>Func` would implement the interface, were it not sealed. 

```go
// Single-method interface type
type RateLimiter interface {
    ReserveRate(context.Context, int) RateReservation
}

// Single function type
ReserveRateFunc func(context.Context, int) RateReservation

func (f ReserveRateFunc) ReserveRate(ctx context.Context, value int) RateReservation {
    if f == nil {
        return rateReservationImpl{} // Composite no-op behavior
    }
    f(ctx, value)
}

// The matching concrete type.
type rateLimiterImpl struct {
    ReserveRateFunc
}
```

### 3. Use Constructors for Interface Values

Provide constructor functions rather than exposing concrete types. By
default, each interface should provide a `New<Type>` constructor for
all which returns the corresponding concrete implementation
struct. Methods are "required" when they are explicitly listed as
parameters methods are arguments in the constructor. In addition, a
Functional Optional pattern is provided for use by future optional
methods.

For example:

```go
func NewRateReservation(wf WaitTimeFunc, cf CancelFunc, _ ...RateReservationOption) RateReservation {
    return rateReservationImpl{
        WaitTimeFunc: wf,
        CancelFunc:   cf,
    }
}

func NewRateLimiter(f ReserveRateFunc, _ ...RateLimiterOption) RateLimiter {
    return rateLimiterImpl{ReserveRateFunc: f}
}
```

[The Go language automatically converts function literal
expressions](https://go.dev/doc/effective_go#conversions) into the
correct named type (i.e., `<Method>Func`), so we can pass function
literals to these constructors without an explicit conversion:

```go
    return NewRateReservation(
        // Wait time 1 second
        func() time.Duration { return time.Second },
	// Cancel is a no-op.
	nil,
    )
```

For more complicated interfaces, this pattern can be combined with the
[Functional Option
pattern](https://commandcenter.blogspot.com/2014/01/self-referential-functions-and-design.html)
in Golang, shown in the next example.

Taken from `receiver/receiver.go`, here we setup signal-specific
functions using a functional-option argument passed to
`receiver.NewFactory`:

```go
// Setup optional signal-specific functions (e.g., logs)
func WithLogs(createLogs CreateLogsFunc, sl component.StabilityLevel) FactoryOption {
	return factoryOptionFunc(func(o *factoryImpl, cfgType component.Type) {
		o.CreateLogsFunc = createLogs
		o.LogsStabilityFunc = sl.Self // See (5) below
	})
}

// Accept options to configure various aspects of the interface
func NewFactory(cfgType component.Type, createDefaultConfig component.CreateDefaultConfigFunc, options ...FactoryOption) Factory {
	f := factoryImpl{
		Factory: component.NewFactory(cfgType.Self, createDefaultConfig),
	}
	for _, opt := range options {
		opt.applyOption(&f, cfgType)
	}
	return f
}
```

### 4. Seal Public Interface Types

Using an unexported method "seals" the interface type so external
packages can only use, not implement the interface. This allows
interfaces to evolve safely because users are forced to use
constructor functions.

```go
// Public interfaces must include at least one private method
type RateLimiter interface {
    ReserveRate(context.Context, int) (RateReservation, error)

    // Prevents external implementations
    private()
}

// Concrete implementations are sealed with this method.
type rateLimiterImpl struct {
    ReserveRateFunc
}

func (rateLimiterImpl) private() {}
```

This practice enables safely evolving interfaces. A new method can be
added to a public interface type because public constructor functions
force the user to obtain the new type and the new type is guaranteed
to implement the old interface. If the functional option pattern is
already being used, then new interface methods will not require new
constructors, only new options. If the functional option pattern is
not in use, backwards compatibility can be maintained by adding new
constructors, for example:

```go
type RateLimiter interface {
    // Original method
    ReserveRate(context.Context, int) RateReservation

    // New method (optional support)
    ExtraFeature()
}

// Original constructor
func NewRateLimiter(f ReserveRateFunc) RateLimiter { ... }

// New constructor
func NewRateLimiterWithOptions(rf ReserveRateFunc, opts ...Option) RateLimiter { ... }

// New option
func WithExtraFeature(...) Option { ... }
```

### 5. Constant-value Function Implementations

For types defined by simple values, especially for enumerated types,
define a `Self()` method to act as the corresponding value:

```go
// Self returns itself.
func (t Config) Self() Config {
     return t
}

// ConfigFunc is ...
type ConfigFunc func() Config

// Config gets the default configuration for this factory.
func (f ConfigFunc) Config() Config {
	if f == nil {
	}
	return f()
}
```

For example, we can decompose, modify, and recompose a
`component.Factory` easily using Self instead of the inline `func()
Config { return cfg }` to capture the constant-valued Config function:

```go
// Copy a factory from somepackage, modify its default config.
func modifiedFactory() Factory {
    original := somepackage.NewFactory()
    otype := original.Type()
    cfg := original.CreateDefaultConfig()

    // Modify the config object
    ...

    // Here, otype.Self equals original.Type.
    return component.NewFactory(otype.Self, cfg.Self)
}    
```

### 6. How to apply the Functional Option pattern

The functional option pattern is well known. For the Functional
Composition pattern, we require the use of Functional Option arguments

## Examples

This pattern enables composition by making it easy to compose and
decompose interface values. For example, to wrap a `receiver.Factory`
with a limiter of some sort:

```go
// Transform existing factories with cross-cutting concerns
func NewLimitedFactory(fact receiver.Factory, cfg LimiterConfigurator) receiver.Factory {
    return receiver.NewFactoryImpl(
        fact.Type,
        fact.CreateDefaultConfig,
        receiver.CreateTracesFunc(limitReceiver(fact.CreateTraces, traceTraits{}, cfg)),
        fact.TracesStability,
        receiver.CreateMetricsFunc(limitReceiver(fact.CreateMetrics, metricTraits{}, cfg)),
        fact.MetricsStability,
        receiver.CreateLogsFunc(limitReceiver(fact.CreateLogs, logTraits{}, cfg)),
        fact.LogsStability,
    )
}
```

For example, it is easy to add logging to an existing function:

```go
func addLogging(f ReserveRateFunc) ReserveRateFunc {
    return func(ctx context.Context, n int) (RateReservation, error) {
        log.Printf("Reserving rate for %d", n)
        return f(ctx, n)
    }
}
```
