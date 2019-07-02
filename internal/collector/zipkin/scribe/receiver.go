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

// Package zipkinscribereceiver wraps the functionality to start the end-point that
// receives Zipkin Scribe spans.
package zipkinscribereceiver

import (
	"fmt"

	"github.com/spf13/viper"
	"go.uber.org/zap"

	"github.com/open-telemetry/opentelemetry-service/cmd/occollector/app/builder"
	"github.com/open-telemetry/opentelemetry-service/consumer"
	"github.com/open-telemetry/opentelemetry-service/receiver"
	"github.com/open-telemetry/opentelemetry-service/receiver/zipkinreceiver/zipkinscribereceiver"
)

// Start starts the Zipkin Scribe receiver endpoint.
func Start(logger *zap.Logger, v *viper.Viper, traceConsumer consumer.TraceConsumer, host receiver.Host) (receiver.TraceReceiver, error) {
	rOpts, err := builder.NewDefaultZipkinScribeReceiverCfg().InitFromViper(v)
	if err != nil {
		return nil, err
	}

	sr, err := zipkinscribereceiver.NewReceiver(rOpts.Address, rOpts.Port, rOpts.Category, traceConsumer)
	if err != nil {
		return nil, fmt.Errorf("Failed to create the Zipkin Scribe receiver: %v", err)
	}

	if err := sr.StartTraceReception(host); err != nil {
		return nil, fmt.Errorf("Cannot start Zipkin Scribe receiver %+v: %v", rOpts, err)
	}

	logger.Info("Zipkin Scribe receiver is running.", zap.Uint16("port", rOpts.Port), zap.String("category", rOpts.Category))

	return sr, nil
}
