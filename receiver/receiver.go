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

package receiver // import "go.opentelemetry.io/collector/receiver"

import (
	"go.opentelemetry.io/collector/component"
)

// A TracesReceiver receives traces.
// Its purpose is to translate data from any format to the collector's internal trace format.
// TracesReceiver feeds a consumer.Traces with data.
//
// For example it could be Zipkin data source which translates Zipkin spans into ptrace.Traces.
type Traces = component.TracesReceiver //nolint:staticcheck

// A MetricsReceiver receives metrics.
// Its purpose is to translate data from any format to the collector's internal metrics format.
// MetricsReceiver feeds a consumer.Metrics with data.
//
// For example it could be Prometheus data source which translates Prometheus metrics into pmetric.Metrics.
type Metrics = component.MetricsReceiver //nolint:staticcheck

// A LogsReceiver receives logs.
// Its purpose is to translate data from any format to the collector's internal logs data format.
// LogsReceiver feeds a consumer.Logs with data.
//
// For example a LogsReceiver can read syslogs and convert them into plog.Logs.
type Logs = component.LogsReceiver //nolint:staticcheck

// CreateSettings configures Receiver creators.
type CreateSettings = component.ReceiverCreateSettings //nolint:staticcheck

// Factory is factory interface for receivers.
//
// This interface cannot be directly implemented. Implementations must
// use the NewReceiverFactory to implement it.
type Factory = component.ReceiverFactory //nolint:staticcheck

// FactoryOption apply changes to ReceiverOptions.
type FactoryOption = component.ReceiverFactoryOption //nolint:staticcheck

// CreateTracesFunc is the equivalent of ReceiverFactory.CreateTracesReceiver().
type CreateTracesFunc = component.CreateTracesReceiverFunc //nolint:staticcheck

// CreateMetricsFunc is the equivalent of ReceiverFactory.CreateMetricsReceiver().
type CreateMetricsFunc = component.CreateMetricsReceiverFunc //nolint:staticcheck

// CreateLogsReceiverFunc is the equivalent of ReceiverFactory.CreateLogsReceiver().
type CreateLogsFunc = component.CreateLogsReceiverFunc //nolint:staticcheck

// WithTraces overrides the default "error not supported" implementation for CreateTracesReceiver and the default "undefined" stability level.
var WithTraces = component.WithTracesReceiver //nolint:staticcheck

// WithMetrics overrides the default "error not supported" implementation for CreateMetricsReceiver and the default "undefined" stability level.
var WithMetrics = component.WithMetricsReceiver //nolint:staticcheck

// WithLogs overrides the default "error not supported" implementation for CreateLogsReceiver and the default "undefined" stability level.
var WithLogs = component.WithLogsReceiver //nolint:staticcheck

// NewFactory returns a ReceiverFactory.
var NewFactory = component.NewReceiverFactory //nolint:staticcheck

// MakeFactoryMap takes a list of receiver factories and returns a map
// with factory type as keys. It returns a non-nil error when more than one factories
// have the same type.
var MakeFactoryMap = component.MakeReceiverFactoryMap //nolint:staticcheck
