// Copyright 2019 OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package config

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/open-telemetry/opentelemetry-collector/component"
	"github.com/open-telemetry/opentelemetry-collector/consumer/consumerdata"
)

func TestExampleExporterConsumer(t *testing.T) {
	exp := &ExampleExporterConsumer{}
	host := component.NewMockHost()
	assert.Equal(t, false, exp.ExporterStarted)
	err := exp.Start(host)
	assert.NoError(t, err)
	assert.Equal(t, true, exp.ExporterStarted)

	assert.Equal(t, 0, len(exp.Traces))
	err = exp.ConsumeTraceData(context.Background(), consumerdata.TraceData{})
	assert.NoError(t, err)
	assert.Equal(t, 1, len(exp.Traces))

	assert.Equal(t, 0, len(exp.Metrics))
	err = exp.ConsumeMetricsData(context.Background(), consumerdata.MetricsData{})
	assert.NoError(t, err)
	assert.Equal(t, 1, len(exp.Metrics))

	assert.Equal(t, false, exp.ExporterShutdown)
	err = exp.Shutdown()
	assert.NoError(t, err)
	assert.Equal(t, true, exp.ExporterShutdown)
}

func TestExampleReceiverProducer(t *testing.T) {
	rcv := &ExampleReceiverProducer{}
	host := component.NewMockHost()
	assert.Equal(t, false, rcv.Started)
	err := rcv.Start(host)
	assert.NoError(t, err)
	assert.Equal(t, true, rcv.Started)

	err = rcv.Shutdown()
	assert.NoError(t, err)
	assert.Equal(t, true, rcv.Started)
}
