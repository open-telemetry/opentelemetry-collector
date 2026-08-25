# Storage

**Status: under development; This is currently just the interface**

A storage extension persists state beyond the collector process. Other components can request a storage client from the storage extension and use it to manage state.

The `storage.Extension` interface extends `component.Extension` by adding the following method:
```
GetClient(context.Context, component.Kind, component.ID, string) (Client, error)
```

The `storage.Client` interface contains the following methods:
```
Get(context.Context, string) ([]byte, error)
Set(context.Context, string, []byte) error
Delete(context.Context, string) error
Close(context.Context) error
```

It is possible to execute several operations in a single transaction via `Batch`. The method takes a collection of
`Operation` arguments (each of which contains `Key`, `Value` and `Type` properties):
```
Batch(context.Context, ...Operation) error
```

The elements itself can be created using:

```
SetOperation(string, []byte) Operation
GetOperation(string) Operation
DeleteOperation(string) Operation
```

Get operation results are stored in-place into the given Operation and can be retrieved using its `Value` property.

Note: All methods should return error only if a problem occurred. (For example, if a file is no longer accessible, or if a remote service is unavailable.)

Note: It is the responsibility of each component to `Close` a storage client that it has requested.

## Walking storage entries

A `storage.Client` may optionally implement the `storage.Walker` interface to support iterating over all key/value entries:
```
Walk(context.Context, WalkFunc) error
```

The `WalkFunc` callback is called for every key/value pair. Key order is not guaranteed.

The callback returns a slice of `Operation` and an error:
```
WalkFunc = func(key string, value []byte) ([]*Operation, error)
```

Operations returned by `WalkFunc` are collected during the walk and applied in order when the walk completes successfully or when `SkipAll` is returned. Returning a `nil` slice is valid and contributes no operations.

All operation types (`Get`, `Set`, `Delete`) are valid in the returned slice. `Get` operations work as usual — the result is stored in the `Operation` instance.

If `WalkFunc` returns the `SkipAll` error, the walk stops early and all collected operations are still applied. If `WalkFunc` returns any other non-nil error, or if the walk encounters an internal error, the walk stops and no collected operations are applied.
