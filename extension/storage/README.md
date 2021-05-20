# Storage

A storage extension persists state beyond the collector process. Other components can request a storage client from the storage extension and use it to manage state. 

The `storage.Extension` interface extends `component.Extension` by adding the following method:
```
GetClient(component.Kind, component.Kind, config.ComponentID) (Client, error)
```

The `storage.Client` interface contains the following methods:
```
Get(string) ([]byte, error)
Set(string, []byte) error
Delete(string) error
```
Note: All methods should return error only if a problem occurred. (For example, if a file is no longer accessible, or if a remote service is unavailable.)
