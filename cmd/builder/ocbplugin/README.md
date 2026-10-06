# OCB Plugins

This package provides an API to create an OCB Plugin. It features the interface that a plugin is expected to implement and a function that can be used in a plugin's `main.go` to facilitate the plugin running as a process.

If you have a type that implements the `OCBPlugin` interface, you can call the `RunPlugin` function from this package with that type:

```go
package main

import "go.opentelemetry.io/collector/cmd/builder/ocbplugin"

type myType struct{}

var _ ocbplugin.OCBPlugin = (*myType)(nil)

func main() {
  ocbplugin.RunPlugin(&myType{})
}
```

<!-- This will link to the documentation on OCB Hooks when that functionality is merged. -->
This makes the plugin usable via OCB Hooks by giving it the proper CLI interface to recognize config files passed in by a calling OCB process.

## Testing

You can ensure your plugin implementation is valid by using the `ocbplugintest` package:

```go
package main

import (
  "go.opentelemetry.io/collector/cmd/builder/ocbplugin"
  "go.opentelemetry.io/collector/cmd/builder/ocbplugin/ocbplugintest"
)

type myType struct{}

var _ ocbplugin.OCBPlugin = (*myType)(nil)

func TestPluginValid(t *testing.T) {
  if !ocbplugintest.IsValidOCBPlugin(&myType{}) {
    t.Fatal("the myType plugin isn't valid")
  }
}
```