## Plugins

Plugins are a mechanism by which symbols can be dynamically
loaded and linked into a binary.

A plugin system democratizes and scales development of observability signal
exporting, expanding on OpenCensus' goal. By democratization, think
of the ability to separately develop your own exporter, compile the
source separately and only provide only the application binary compatible
object code file without having to make a pull request against this
repository. It also ensures smaller binary sizes but also the ability
to dynamically enable and disable various exporters.

The ability to dynamically enable an exporter just by linking it ensures
that reactions to critical events are possible by linking specialized
exporters at runtime.

## Limitations
* Go's plugin system building only works on Linux and macOS.
* The plugins should be compiled with the EXACT same version of Go as
was used to compile the running binary `ocagent`
* The plugins need to have been built on Go1.10+ because Go didn't support
plugins for darwin/amd64 in Go1.9
* The shared libraries should be written in Go or at least their symbols
should match exactly

## Writing a TraceExporter plugin

To write a TraceExporter plugin, we'll have to conform to the interface
exporter.TraceExporterPlugin whose signature is
```go
type TraceExporterPlugin interface {
	TraceExportersFromConfig(blob []byte, configType ConfigType) ([]TraceExporter, error)
}
```

* The first step is to ensure that the package used is `main` but no main function
```go
package main
```

Now with that we'll to create a type that exports to multiple files in parallel

### Sample implementation
```go
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"go.opencensus.io/trace"
	yaml "gopkg.in/yaml.v2"

	commonpb "github.com/census-instrumentation/opencensus-proto/gen-go/agent/common/v1"
	"github.com/census-instrumentation/opencensus-service/exporter"
)

type customInParallelFilesTraceExporter struct {
	mu    sync.Mutex
	files []*os.File
}

type config struct {
	Filepaths []string `yaml:"file_paths"`
}

var _ exporter.TraceExporterPlugin = (*customInParallelFilesTraceExporter)(nil)

func (cwp *customInParallelFilesTraceExporter) TraceExportersFromConfig(blob []byte, configType exporter.ConfigType) (exporters []exporter.TraceExporter, err error) {
	if configType != exporter.ConfigTypeYAML {
		return nil, fmt.Errorf("Only parsing configs of YAML type, got %v", configType)
	}

	// Now parse the YAML
	cfg := new(config)
	if err := yaml.Unmarshal(blob, cfg); err != nil {
		return nil, err
	}

	if len(cfg.Filepaths) == 0 {
		return nil, fmt.Errorf("No parsed filepaths")
	}

	for _, relPath := range cfg.Filepaths {
		abspath, err := filepath.Abs(relPath)
		if err != nil {
			// Abort mission
			cwp.closeAllFiles()
			return nil, err
		}
		f, err := os.Open(abspath)
		if err != nil {
			// Abort mission
			cwp.closeAllFiles()
			return nil, err
		}
		exporters = append(exporters, makeJSONEncoder(f))
		cwp.files = append(cwp.files, f)
	}
	return exporters, nil
}

type fileExporter struct {
	jstream *json.Encoder
}

func makeJSONEncoder(w io.Writer) *fileExporter { return &fileExporter{jstream: json.NewEncoder(w)} }

type outfileData struct {
	At       time.Time         `json:"at"`
	Node     *commonpb.Node    `json:"node"`
	SpanData []*trace.SpanData `json:"span_data"`
}

func (fe *fileExporter) ExportSpanData(node *commonpb.Node, spandata ...*trace.SpanData) error {
	return fe.jstream.Encode(&outfileData{At: time.Now().UTC(), Node: node, SpanData: spandata})
}

func (cwp *customInParallelFilesTraceExporter) Stop(ctx context.Context) error {
	// Before closing all the files, invoke Sync on each of them.
	cwp.mu.Lock()
	defer cwp.mu.Unlock()

	var errs []string
	for _, file := range cwp.files {
		if file == nil {
			continue
		}
		_ = file.Sync()
		if err := file.Close(); err != nil {
			errs = append(errs, fmt.Sprintf("%s Close error: %v", file.Name(), err))
		}
	}

	if len(errs) == 0 {
		return nil
	}

	return fmt.Errorf(strings.Join(errs, "\n"))
}

func (cwp *customInParallelFilesTraceExporter) closeAllFiles() {
	for _, file := range cwp.files {
		if file != nil {
			_ = file.Close()
		}
	}
}
```

## Compiling the plugin to make it a shared object
Once your package implementation is ready, you can create shared objects by setting the buildmode
for `go build` to `plugin`. For example, with our implementation in file `files_exporter.go`

```go
go build -buildmode=plugin files_exporter
```
which will produce `files_exporter.so` and it is that this `.so` file that will be passed
to ocagent at startup time.

