// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package fileexporter

import (
	"context"
	"io"
	"sync"

	commonpb "github.com/census-instrumentation/opencensus-proto/gen-go/agent/common/v1"
	resourcepb "github.com/census-instrumentation/opencensus-proto/gen-go/resource/v1"
	"github.com/gogo/protobuf/jsonpb"
	"github.com/gogo/protobuf/proto"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer/pdata"
	"go.opentelemetry.io/collector/consumer/pdatautil"
	"go.opentelemetry.io/collector/translator/internaldata"
)

// Marshaler configuration used for marhsaling Protobuf to JSON. Use default config.
var marshaler = &jsonpb.Marshaler{}

// Helper struct to write JSON objects and arrays.
type jsonWriter struct {
	firstFieldDone     bool
	firstArrayItemDone bool
	writer             io.Writer
}

func (jw *jsonWriter) Reset() {
	jw.firstFieldDone = false
}

// Begin writing JSON. Call first.
func (jw *jsonWriter) Begin() error {
	_, err := io.WriteString(jw.writer, "{\n")
	return err
}

// End writing JSON. Call last.
func (jw *jsonWriter) End() error {
	_, err := io.WriteString(jw.writer, "\n}\n")
	return err
}

// MarshalObject marshals an object as a field of top-level object.
func (jw *jsonWriter) MarshalObject(fieldName string, pb proto.Message) error {
	if jw.firstFieldDone {
		io.WriteString(jw.writer, ",\n")
	} else {
		jw.firstFieldDone = true
	}
	_, err := io.WriteString(jw.writer, `  "`+fieldName+`": `)
	if err != nil {
		return err
	}

	err = marshaler.Marshal(jw.writer, pb)
	if err != nil {
		return err
	}
	return nil
}

// BeginMarshalArray prepares to marshal array items under a field of top-level object.
func (jw *jsonWriter) BeginMarshalArray(fieldName string) error {
	if jw.firstFieldDone {
		io.WriteString(jw.writer, ",\n")
	} else {
		jw.firstFieldDone = true
	}
	_, err := io.WriteString(jw.writer, `  "`+fieldName+"\": [")
	jw.firstArrayItemDone = false
	return err
}

// EndMarshalArray must be called after all array items are marshaled.
func (jw *jsonWriter) EndMarshalArray() error {
	var str string
	if jw.firstArrayItemDone {
		// Non-empty array. End on a new line.
		str = "\n  ]"
	} else {
		// Empty array. End on the same line.
		str = "]"
	}
	_, err := io.WriteString(jw.writer, str)
	return err
}

// MarshalArrayItem marshals single array item. Call repeatedly after BeginMarshalArray.
func (jw *jsonWriter) MarshalArrayItem(pb proto.Message) error {
	var str string
	if jw.firstArrayItemDone {
		str = ",\n    "
	} else {
		str = "\n    "
		jw.firstArrayItemDone = true
	}
	_, err := io.WriteString(jw.writer, str)
	if err != nil {
		return err
	}
	err = marshaler.Marshal(jw.writer, pb)
	if err != nil {
		return err
	}
	return nil
}

func exportResourceAndNode(writer *jsonWriter, node *commonpb.Node, resource *resourcepb.Resource) error {
	if resource != nil {
		err := writer.MarshalObject("resource", resource)
		if err != nil {
			return err
		}
	}
	if node != nil {
		return writer.MarshalObject("node", node)
	}
	return nil
}

// Exporter is the implementation of file exporter that writes telemetry data to a file
// in Protobuf-JSON format.
type Exporter struct {
	file  io.WriteCloser
	mutex sync.Mutex
}

func (e *Exporter) ConsumeTraces(_ context.Context, td pdata.Traces) error {
	octds := internaldata.TraceDataToOC(td)
	for _, octd := range octds {
		// Ensure only one write operation happens at a time.
		e.mutex.Lock()
		defer e.mutex.Unlock()

		// Prepare to write JSON object.
		jw := &jsonWriter{writer: e.file}
		if err := jw.Begin(); err != nil {
			return err
		}
		defer jw.End()

		if err := exportResourceAndNode(jw, octd.Node, octd.Resource); err != nil {
			return err
		}

		if err := jw.BeginMarshalArray("spans"); err != nil {
			return err
		}
		defer jw.EndMarshalArray()

		for _, span := range octd.Spans {
			if span != nil {
				if err := jw.MarshalArrayItem(span); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func (e *Exporter) ConsumeMetrics(_ context.Context, md pdata.Metrics) error {
	ocmds := pdatautil.MetricsToMetricsData(md)
	for _, ocmd := range ocmds {
		// Ensure only one write operation happens at a time.
		e.mutex.Lock()
		defer e.mutex.Unlock()

		// Prepare to write JSON object.
		jw := &jsonWriter{writer: e.file}
		if err := jw.Begin(); err != nil {
			return err
		}
		defer jw.End()

		if err := exportResourceAndNode(jw, ocmd.Node, ocmd.Resource); err != nil {
			return err
		}

		if err := jw.BeginMarshalArray("metrics"); err != nil {
			return err
		}
		defer jw.EndMarshalArray()

		for _, metric := range ocmd.Metrics {
			if metric != nil {
				if err := jw.MarshalArrayItem(metric); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func (e *Exporter) ConsumeLogs(_ context.Context, ld pdata.Logs) error {
	// Ensure only one write operation happens at a time.
	e.mutex.Lock()
	defer e.mutex.Unlock()

	// Prepare to write JSON object.
	jw := &jsonWriter{writer: e.file}

	logsProto := pdata.LogsToOtlp(ld)

	for _, rl := range logsProto {
		if err := jw.Begin(); err != nil {
			return err
		}
		err := jw.MarshalObject("resource", rl.Resource)
		if err != nil {
			return err
		}

		if err := jw.BeginMarshalArray("logs"); err != nil {
			return err
		}

		for _, ill := range rl.InstrumentationLibraryLogs {
			// TODO: output ill.InstrumentationLibrary
			for _, log := range ill.Logs {
				if log != nil {
					if err := jw.MarshalArrayItem(log); err != nil {
						return err
					}
				}
			}
		}
		jw.EndMarshalArray()
		jw.End()
		jw.Reset()
	}
	return nil
}

func (e *Exporter) Start(ctx context.Context, host component.Host) error {
	return nil
}

// Shutdown stops the exporter and is invoked during shutdown.
func (e *Exporter) Shutdown(context.Context) error {
	return e.file.Close()
}
