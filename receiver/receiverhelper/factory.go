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

package receiverhelper // import "go.opentelemetry.io/collector/receiver/receiverhelper"

import (
	"go.opentelemetry.io/collector/component"
)

// Deprecated: [v0.46.0] use component.ReceiverFactoryOption.
type FactoryOption = component.ReceiverFactoryOption

// Deprecated: [v0.46.0] use component.WithTracesReceiver.
var WithTraces = component.WithTracesReceiver

// Deprecated: [v0.46.0] use component.WithMetricsReceiver.
var WithMetrics = component.WithMetricsReceiver

// Deprecated: [v0.46.0] use component.WithLogsReceiver.
var WithLogs = component.WithLogsReceiver

// Deprecated: [v0.46.0] use component.ReceiverCreateDefaultConfigFunc.
type CreateDefaultConfig = component.ReceiverCreateDefaultConfigFunc

// Deprecated: [v0.46.0] use component.CreateTracesReceiverFunc.
type CreateTracesReceiver = component.CreateTracesReceiverFunc

// Deprecated: [v0.46.0] use component.CreateMetricsReceiverFunc.
type CreateMetricsReceiver = component.CreateMetricsReceiverFunc

// Deprecated: [v0.46.0] use component.CreateLogsReceiverFunc.
type CreateLogsReceiver = component.CreateLogsReceiverFunc

// Deprecated: [v0.46.0] use component.NewReceiverFactory.
var NewFactory = component.NewReceiverFactory
