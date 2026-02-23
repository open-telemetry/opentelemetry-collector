# Component Interface Guidelines

## Overview

The OpenTelemetry Collector has a number of public interfaces and
extension points that require careful attention to avoid breaking
changes for users. 

These guidelines describe how to achieve safe interface evolution in
Golang. This approach is recommended for all Golang modules that want
a safe approach to interface evolution.

When an interface type is exported for users outside of this
repository, the type MUST follow these guidelines. This does not
apply to internal packages, by definition.

## Background

As its most prominent feature, for every method in the public
interface, a corresponding `type <Method>Func func(...) ...`
declaration in the same package will exist. The method and the
function have the same signature, by design.

Users of the [`net/http` package have seen this
pattern](https://pkg.go.dev/net/http#HandlerFunc). `http.HandlerFunc`
can be seen as a prototype for this pattern, in this case for HTTP
servers. The public interface type `http.Handler`:

```
// A Handler responds to an HTTP request.
type Handler interface {
	ServeHTTP(ResponseWriter, *Request)
}
```

has a corresponding function type:

```
// The HandlerFunc type is an adapter to allow the use of
// ordinary functions as HTTP handlers. If f is a function
// with the appropriate signature, HandlerFunc(f) is a
// [Handler] that calls f.
type HandlerFunc func(ResponseWriter, *Request)
// ServeHTTP calls f(w, r).
func (f HandlerFunc) ServeHTTP(w ResponseWriter, r *Request) {
	f(w, r)
}
```

We use this pattern extensively, however there are important
differences with ours and this specific case:

- Public interfaces can only refer to other public interfaces, not 
  concrete types. The `*Request` (pointer-to-struct) is not compatible
  with safe interface evolution.
- Every function type must have a simple "no-op" behavior
  corresponding with its nil value. The Golang `HandlerFunc`
  implementation does not have a no-op implementation (callers are
  required to pass `func(http.ResponseWriter, *http.Request) {}` for
  no-op behavior).

In our version of this, the implementation is required to check for
nil and expose only other interfaces for parameter- and return-types,
themselves subject to the same safety requirements.

```
// SAFE VERSION satisfies requirements because arguments are interfaces
// and nil is checked.
//
// ServeHTTP calls f(w, r).
func (f HandlerFunc) ServeHTTP(w ResponseWriter, r RequestIface) {
    if f == nil {
        return
    }
	f(w, r)
}
```

The function type, in this pattern, corresponds with a single method
of an exposed `interface` type and has several uses:

- When a sealed-interface constructor takes `<Method>Func` as its argument,
  callers can pass `nil` for the no-op.
- When an object embeds the `<Method>Func`, they inherit an
  implementation of the interface. Uninitialized types that embed `<Method>Func`
  automatically gain a no-op implementation of the associated method.

## Interface Consumers and Providers

Extension interfaces fall into two categories: **sealed interfaces**
and **open interfaces**.

Sealed interfaces are provided by a package with safe-evolution in
mind, not for external implementations. The use of sealed interfaces
allows a repository to stabilize public methods while ensuring the
ability to add future methods by preventing external
construction. External implementations are prohibited in this
case. Examples include the `Factory` interface and `NewFactory`
constructor used to register Collector components in each of the
`receiver`, `processor`, `exporter`, `connector`, and `extension`
sub-modules. In this example, new factory methods can be added in the
future, through functional options, without breaking existing consumers
or providers.

Open interfaces are meant to enable extension points, enabling
capability detection on a method-by-method basis. Open interfaces are
implemented by extension components to provide capabilities through
type assertions. While it is not safe to add a new method to an open
interface, as it will break existing implementations, it is safe to
add new and alternative extension points that serve equivalent
functions. Consumers . Examples include
`extensionmiddleware.HTTPClient` and `extensionmiddleware.GRPCServer`,
our middleware interfaces.  In this example, new protocols can have
middleware support added in the future, and we can adjust to changes
in the existing HTTP and gRPC middleware.

## Key Concepts

### Two Categories of Extension Interfaces

Extension interfaces fall into two categories based on how they are
discovered and used.

**Sealed interfaces** are provided by a package and consumed by users.
They use an unexported `private()` method to prevent external
implementations, guaranteeing all implementations come through
package-provided constructors. Following the [guidance in "working
with
interfaces"](https://go.dev/blog/module-compatibility#working-with-interfaces),
this practice enables safely evolving interfaces because users are
forced to use constructor functions. Examples include
`receiver.Factory` and `component.Host`.

**Open interfaces** are implemented by extensions and discovered
through type assertions at runtime. Consumers check if an extension
implements a capability using Go's type assertion syntax. Multiple
independent implementations exist, and extensions "opt in" to
providing a feature. Examples include `extensionmiddleware.GRPCClient`
and `extensionauth.Client`.

The key distinction: sealed interfaces prevent external
implementation; open interfaces invite it.

### When to Seal an Interface

Interfaces MUST be sealed when the package provides the canonical
implementation and needs guaranteed control over all instances. This
is appropriate when interface evolution requires adding methods that
all implementations must have, and when the Functional Option pattern
will be used to extend capabilities over time.

### Evolving Sealed Interfaces

Sealed interfaces evolve by adding new methods with corresponding
function types and options. Because all implementations use
package-provided constructors, new methods can be added safely.

The `receiver.Factory` interface demonstrates this pattern:

```go
// Factory is sealed—external implementations prevented
type Factory interface {
    // Embed a sealed interface from another module
    component.Factory
    
    // Traces-specific methods
    CreateTraces(...) (Traces, error)
    TracesStability() component.StabilityLevel

    // ... additional methods for other signals

    unexportedFactoryFunc() // sealing method
}

// <Method>Func for CreateTraces
type CreateTracesFunc func(context.Context, Settings, component.Config, consumer.Traces) (Traces, error)

// Functional option for traces capability
func WithTraces(createTraces CreateTracesFunc, sl component.StabilityLevel) FactoryOption { ... }

// Constructor uses functional options
func NewFactory(cfgType component.Type, createDefaultConfig component.CreateDefaultConfigFunc, options ...FactoryOption) Factory
```

This pattern supports adding a new kind of signal to the interface,
safely, for example by adding `CreateProfiles()`, `CreateProfilesFunc`, and `WithProfiles`.

If the functional option pattern is already in use, new interface
methods require only new options. If there is not an existing
functional option pattern, it is still safe to extend the interface by
creating a new constructor.

Major interfaces SHOULD use the functional option pattern even when
there are initially no options.

### Evolving Open Interfaces

Open interfaces evolve by creating new companion interfaces for new
capabilities. Open interfaces MUST remain unchanged until a major
version bump, to preserve compatibility.

Imagine a new RPC framework is introduced, with a new kind of client
configuration that middleware extensions can implement. We cannot
extend the existing interfaces, but we can create new ones,

```
// Imagine a new RPC framework called "Super".
type SuperClient interface {
    GetSuperClientOptions(context.Context) ([]super.ClientOption, error)
}
```

We can also accommodate new interface types corresponding with
existing frameworks. For example, if gRPC decides to add a new sort of
configuration type, we can add an optional new interface,

```
// Imagine a V2 gRPC option type.
type GRPCClientV2 interface {
    GetGRPCClientOptionsV2(context.Context) ([]grpc.ClientOptionV2, error)
}
```

The new middleware extension can be added without changing the existing
implementations, because only a user of the new framework needs to know
about the new kind of middleware.

This works because components generally know which specific extension
interface they want. When there is more than one viable extension
interface, callers can choose the one they prefer depending on the use.

### Test Helpers with NewNop and NewErr

Every public and extension interface package SHOULD provide test
helpers in a `<package>test` subpackage using a dedicated Go module
(`go.mod`). These helpers generally can be composed of a
`<Method>Func` for every method in the extension.

For example, here is a type for use in testing that implements every
public interface method, making it easy for tests to override only one
method.

```go
type baseExtension struct {
    component.StartFunc
    component.ShutdownFunc
    extensionmiddleware.GetHTTPRoundTripperFunc
    extensionmiddleware.GetGRPCClientOptionsFunc
    extensionmiddleware.GetGRPCClientOptionsContextFunc

    // ... new middleware interfaces can be added using the corresponding
    // <Method>Funcs.
}
```

Two constructors are typically provided for standard testing:

`NewNop()` returns an extension where all methods have no-op behavior.
Because nil function types return zero values, `&baseExtension{}` is a
valid do-nothing implementation.

`NewErr(err error)` returns an extension where all methods return the
specified error, enabling testing of error handling paths.

## Examples

The `extensionmiddleware` package demonstrates this pattern with open
interfaces for capability detection:

- **Interface definitions**:
  [extension/extensionmiddleware/client.go](../../extension/extensionmiddleware/client.go)
  defines `GRPCClient`, `GRPCClientContext`, and their corresponding
  function types
- **Consumer uses both interfaces**:
  [config/configmiddleware/configmiddleware.go](../../config/configmiddleware/configmiddleware.go)
  shows cascading type assertions to detect capabilities
- **Test helpers**:
  [extension/extensionmiddleware/extensionmiddlewaretest/err.go](../../extension/extensionmiddleware/extensionmiddlewaretest/err.go)
  demonstrates `baseExtension` struct embedding all function types
  with `NewNop()` and `NewErr()`

## Summary

| Aspect | Sealed Interface | Open Interface |
|--------|-----------------|----------------|
| Purpose | Package-provided implementation | Capability detection |
| `private()` method | Yes | No |
| Constructor | Required (`NewX`) | Not used |
| External implementation | Prevented | Encouraged |
| Evolution | Add methods + options | Add companion interfaces |
| Example | `receiver.Factory` | `extensionmiddleware.GRPCClient` |

## References

- [Go Blog: Working with Interfaces](https://go.dev/blog/module-compatibility#working-with-interfaces)
- [http.HandlerFunc](https://pkg.go.dev/net/http#HandlerFunc)—prior art
- [Functional Options Pattern](https://commandcenter.blogspot.com/2014/01/self-referential-functions-and-design.html)
- [Function Types in Go](https://kinbiko.com/posts/2021-01-10-function-types-in-go/)
