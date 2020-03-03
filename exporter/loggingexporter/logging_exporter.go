// Copyright 2019, OpenTelemetry Authors
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

package loggingexporter

import (
	"context"
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/open-telemetry/opentelemetry-collector/config/configmodels"
	"github.com/open-telemetry/opentelemetry-collector/consumer/consumerdata"
	"github.com/open-telemetry/opentelemetry-collector/exporter"
	"github.com/open-telemetry/opentelemetry-collector/exporter/exporterhelper"

	"go.uber.org/zap"
)

type traceDataBuffer struct {
	str strings.Builder
}

func (b *traceDataBuffer) logEntry(format string, a ...interface{}) {
	b.str.WriteString(fmt.Sprintf(format+"\n", a...))
}

func (b *traceDataBuffer) logAttr(label string, value string) {
	b.logEntry("    %-12s: %s", label, value)
}

func (b *traceDataBuffer) logMap(label string, data *map[string]string) {
	if data == nil {
		return
	}

	b.logEntry(label + ":")
	for label, value := range *data {
		b.logEntry("     -> %s: %s", label, value)
	}
}

type loggingExporter struct {
	level  string
	logger *zap.Logger
}

func (s *loggingExporter) pushTraceData(
	ctx context.Context,
	td consumerdata.TraceData,
) (int, error) {
	debug := s.level == "debug"

	buf := traceDataBuffer{}

	buf.logEntry("TraceData with %d spans", len(td.Spans))

	if td.Resource != nil {
		buf.logEntry("Resource %s with $d labels", td.Resource.Type, len(td.Resource.Labels))
		if debug {
			buf.logMap("Resource labels", &td.Resource.Labels)
		}
	}

	if debug {
		if td.Node != nil {
			buf.logEntry("Node service name: %s", td.Node.ServiceInfo.Name)
			id := td.Node.Identifier
			if id != nil {
				buf.logEntry("HostName: %s", id.HostName)
				buf.logEntry("PID %s", id.Pid)
			}
			buf.logMap("Node attributes", &td.Node.Attributes)
			li := td.Node.LibraryInfo
			if li != nil {
				buf.logEntry("Library language: %s", li.Language.String())
				buf.logEntry("Core library version: %s", li.CoreLibraryVersion)
				buf.logEntry("Exporter version: %s", li.ExporterVersion)
			}
		}
	}

	s.logger.Info(buf.str.String())

	if debug {
		for i, span := range td.Spans {
			buf = traceDataBuffer{}
			buf.logEntry("Span #%d", i)
			if span == nil {
				buf.logEntry("* Empty span")
				continue
			}

			buf.logAttr("Trace ID", hex.EncodeToString(span.TraceId))
			buf.logAttr("ID", hex.EncodeToString(span.SpanId))
			buf.logAttr("Parent ID", hex.EncodeToString(span.ParentSpanId))
			buf.logAttr("Name", span.Name.Value)
			buf.logAttr("Kind", span.Kind.String())
			buf.logAttr("Start time", span.StartTime.String())
			buf.logAttr("End time", span.EndTime.String())
			if span.Status != nil {
				buf.logAttr("Status code", string(span.Status.Code))
				buf.logAttr("Status message", span.Status.Message)
			}

			if span.Attributes != nil {
				buf.logAttr("Span attributes", "")
				for attr, value := range span.Attributes.AttributeMap {
					v := ""
					ts := value.GetStringValue()

					if ts != nil {
						v = ts.Value
					} else {
						// For other types, just use the proto compact form rather than digging into series of checks
						v = value.String()
					}

					buf.logEntry("         -> %s: %s", attr, v)
				}
			}

			s.logger.Debug(buf.str.String())
		}
	}

	return 0, nil
}

// NewTraceExporter creates an exporter.TraceExporter that just drops the
// received data and logs debugging messages.
func NewTraceExporter(config configmodels.Exporter, level string, logger *zap.Logger) (exporter.TraceExporter, error) {
	s := &loggingExporter{
		level:  level,
		logger: logger,
	}

	return exporterhelper.NewTraceExporter(
		config,
		s.pushTraceData,
		exporterhelper.WithShutdown(logger.Sync),
	)
}

// NewMetricsExporter creates an exporter.MetricsExporter that just drops the
// received data and logs debugging messages.
func NewMetricsExporter(config configmodels.Exporter, logger *zap.Logger) (exporter.MetricsExporter, error) {
	typeLog := zap.String("type", config.Type())
	nameLog := zap.String("name", config.Name())
	return exporterhelper.NewMetricsExporter(
		config,
		func(ctx context.Context, md consumerdata.MetricsData) (int, error) {
			logger.Info("MetricsExporter", typeLog, nameLog, zap.Int("#metrics", len(md.Metrics)))
			// TODO: Add ability to record the received data
			return 0, nil
		},
		exporterhelper.WithShutdown(logger.Sync),
	)
}
