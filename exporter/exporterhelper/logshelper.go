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

package exporterhelper

import (
	"context"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config/configmodels"
	"go.opentelemetry.io/collector/internal/data"
	"go.opentelemetry.io/collector/obsreport"
)

// PushLogsData is a helper function that is similar to ConsumeLogsData but also returns
// the number of dropped logs.
type PushLogsData func(ctx context.Context, md data.Logs) (droppedTimeSeries int, err error)

type logsExporter struct {
	baseExporter
	pushLogsData PushLogsData
}

func (me *logsExporter) ConsumeLogs(ctx context.Context, md data.Logs) error {
	exporterCtx := obsreport.ExporterContext(ctx, me.exporterFullName)
	_, err := me.pushLogsData(exporterCtx, md)
	return err
}

// NewLogsExporter creates an LogsExporter that can record logs and can wrap every request with a Span.
// TODO: Add support for retries.
func NewLogsExporter(config configmodels.Exporter, pushLogsData PushLogsData, options ...ExporterOption) (component.LogExporter, error) {
	if config == nil {
		return nil, errNilConfig
	}

	if pushLogsData == nil {
		return nil, errNilPushLogsData
	}

	pushLogsData = pushLogsWithObservability(pushLogsData, config.Name())

	return &logsExporter{
		baseExporter: newBaseExporter(config.Name(), options...),
		pushLogsData: pushLogsData,
	}, nil
}

func pushLogsWithObservability(next PushLogsData, exporterName string) PushLogsData {
	return func(ctx context.Context, ld data.Logs) (int, error) {
		ctx = obsreport.StartLogsExportOp(ctx, exporterName)
		numDroppedLogs, err := next(ctx, ld)

		numLogs := ld.LogRecordCount()

		obsreport.EndLogsExportOp(ctx, numLogs, numDroppedLogs, err)
		return numLogs, err
	}
}
