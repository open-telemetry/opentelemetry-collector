# Storage

**Status: under development**

A limit extension supports flexible methods for admission and rate
control.  Other components, typically receivers, can request a limit
client from the limit extension and use it to admit requests.

This interface is byte-weight oriented, enabling limiting for
individual payloads within a stream-oriented RPC.  For request-level
limiting, consider using the Auth extension instead.

The `limit.Extension` interface extends `component.Extension` by
adding the following method:

```
GetClient(context.Context, component.Kind, component.ID, string) (Client, error)
```

After the context argument are two component-level identifiers and a
string, which is the name of the extension.  Typically, components
will support a list of named limiters to apply in sequence, e.g.,

```
receivers:
  someprotocol:
    limiters:
	- rate
	- memory
```

The `limit.Client` interface contains the following method:
```
Acquire(ctx context.Context, weight uint64) (ReleaseFunc, error)
```

The weight parameter specifies how large of an request to consider in
the limiter, which should estimate the amount of real memory used by
to represent the data after it is uncompressed.

The result is either a non-nil release function and nil error, or a
nil release function and non-nil error.  The limiter may block or fail
fast, at its discretion.  When a non-nil release function is returned,
the component is responsible for calling the release function after
the memory is no longer used.

Note: It is the responsibility of each component to `Close` a storage
client that it has requested.
