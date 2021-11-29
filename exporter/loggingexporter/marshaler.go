// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//       http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package loggingexporter // import "go.opentelemetry.io/collector/exporter/loggingexporter"

import (
	"fmt"

	"go.opentelemetry.io/collector/exporter/loggingexporter/internal/jsonstream"
	"go.opentelemetry.io/collector/internal/otlptext"
	"go.opentelemetry.io/collector/model/otlp"
	"go.opentelemetry.io/collector/model/pdata"
)

// marshaler embeds marshalers for logs, metrics and traces.
type marshaler struct {
	pdata.LogsMarshaler
	pdata.MetricsMarshaler
	pdata.TracesMarshaler
}

// newMarshaler returns a marshaler that uses the given serialization format.
// Currently the format could be either "text", "json" or "jsonstream". An error
// is returned if the format is unknown.
func newMarshaler(format string) (*marshaler, error) {
	switch format {
	case "text":
		return newTextMarshaler(), nil
	case "json":
		return newJSONMarshaler(), nil
	case "jsonstream":
		return newJSONStreamMarshaler(), nil
	default:
		return nil, fmt.Errorf("invalid format %q", format)
	}
}

// newTextMarshaler returns a marshaler for the text serialization format.
func newTextMarshaler() *marshaler {
	return &marshaler{
		LogsMarshaler:    otlptext.NewTextLogsMarshaler(),
		MetricsMarshaler: otlptext.NewTextMetricsMarshaler(),
		TracesMarshaler:  otlptext.NewTextTracesMarshaler(),
	}
}

// newJSONMarshaler returns a marshaler for the JSON serialization format.
func newJSONMarshaler() *marshaler {
	return &marshaler{
		LogsMarshaler:    otlp.NewJSONLogsMarshaler(),
		MetricsMarshaler: otlp.NewJSONMetricsMarshaler(),
		TracesMarshaler:  otlp.NewJSONTracesMarshaler(),
	}
}

// newJSONStreamMarshaler returns a marshaler for the JSON stream serialization
// format.
func newJSONStreamMarshaler() *marshaler {
	return &marshaler{
		LogsMarshaler:    jsonstream.NewJSONLogsMarshaler(),
		MetricsMarshaler: jsonstream.NewJSONMetricsMarshaler(),
		TracesMarshaler:  jsonstream.NewJSONTracesMarshaler(),
	}
}
