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

package fileexporter

import (
	"context"
	"io"
	"os"
	"sync"

	"github.com/gogo/protobuf/jsonpb"
	"github.com/gogo/protobuf/proto"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer/pdata"
	"go.opentelemetry.io/collector/exporter/exporterhelper"
	"go.opentelemetry.io/collector/internal"
)

// Marshaler configuration used for marhsaling Protobuf to JSON. Use default config.
var marshaler = &jsonpb.Marshaler{}

// fileExporter is the implementation of file exporter that writes telemetry data to a file
// in Protobuf-JSON format.
type fileExporter struct {
	file  io.WriteCloser
	mutex sync.Mutex
}

func newTracesExporter(cfg *Config, params component.ExporterCreateParams) (component.TracesExporter, error) {
	fe, err := createExporter(cfg)
	if err != nil {
		return nil, err
	}
	return exporterhelper.NewTraceExporter(cfg, params.Logger, fe.pushTraces, exporterhelper.WithShutdown(fe.shutdown))
}

func (e *fileExporter) pushTraces(_ context.Context, td pdata.Traces) error {
	return exportMessageAsLine(e, internal.TracesToOtlp(td.InternalRep()))
}

func (e *fileExporter) pushMetrics(_ context.Context, md pdata.Metrics) error {
	return exportMessageAsLine(e, internal.MetricsToOtlp(md.InternalRep()))
}

func (e *fileExporter) pushLogs(_ context.Context, ld pdata.Logs) error {
	return exportMessageAsLine(e, internal.LogsToOtlp(ld.InternalRep()))
}

func exportMessageAsLine(e *fileExporter, message proto.Message) error {
	// Ensure only one write operation happens at a time.
	e.mutex.Lock()
	defer e.mutex.Unlock()
	if err := marshaler.Marshal(e.file, message); err != nil {
		return err
	}
	if _, err := io.WriteString(e.file, "\n"); err != nil {
		return err
	}
	return nil
}

// Shutdown stops the exporter and is invoked during shutdown.
func (e *fileExporter) shutdown(context.Context) error {
	return e.file.Close()
}

func createExporter(cfg *Config) (*fileExporter, error) {
	// There must be one exporter for metrics, traces, and logs. We maintain a
	// map of exporters per config.

	// Check to see if there is already a exporter for this config.
	exporter, ok := exporters[cfg]

	if !ok {
		file, err := os.OpenFile(cfg.Path, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
		if err != nil {
			return nil, err
		}
		exporter = &fileExporter{file: file}

		// Remember the receiver in the map
		exporters[cfg] = exporter
	}
	return exporter, nil
}

// This is the map of already created File exporters for particular configurations.
// We maintain this map because the Factory is asked traces, metrics and logs receivers separately
// when it gets CreateTracesReceiver() and CreateMetricsReceiver() but they must not
// create separate objects, they must use one Receiver object per configuration.
var exporters = map[*Config]*fileExporter{}
