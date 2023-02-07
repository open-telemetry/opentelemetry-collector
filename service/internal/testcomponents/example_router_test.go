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

package testcomponents

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/connector/connectortest"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/consumer/consumertest"
	"go.opentelemetry.io/collector/internal/testdata"
	"go.opentelemetry.io/collector/service/internal/fanoutconsumer"
)

func TestExampleRouter(t *testing.T) {
	conn := &ExampleRouter{}
	host := componenttest.NewNopHost()
	assert.False(t, conn.Started())
	assert.NoError(t, conn.Start(context.Background(), host))
	assert.True(t, conn.Started())

	assert.False(t, conn.Stopped())
	assert.NoError(t, conn.Shutdown(context.Background()))
	assert.True(t, conn.Stopped())
}

func TestTracesRouter(t *testing.T) {
	leftID := component.NewIDWithName("sink", "left")
	rightID := component.NewIDWithName("sink", "right")

	sinkLeft := new(consumertest.TracesSink)
	sinkRight := new(consumertest.TracesSink)

	// The service will build a router to give to every connector.
	// Many connectors will just call router.ConsumeTraces,
	// but some implementation will call RouteTraces instead.
	router := fanoutconsumer.NewTracesRouter(
		map[component.ID]consumer.Traces{
			leftID:  sinkLeft,
			rightID: sinkRight,
		})

	cfg := ExampleRouterConfig{Traces: &LeftRightConfig{Left: leftID, Right: rightID}}
	tr, err := ExampleRouterFactory.CreateTracesToTraces(
		context.Background(), connectortest.NewNopCreateSettings(), cfg, router)
	assert.NoError(t, err)
	assert.False(t, tr.Capabilities().MutatesData)

	td := testdata.GenerateTraces(1)

	assert.NoError(t, tr.ConsumeTraces(context.Background(), td))
	assert.Len(t, sinkRight.AllTraces(), 1)
	assert.Len(t, sinkLeft.AllTraces(), 0)

	assert.NoError(t, tr.ConsumeTraces(context.Background(), td))
	assert.Len(t, sinkRight.AllTraces(), 1)
	assert.Len(t, sinkLeft.AllTraces(), 1)

	assert.NoError(t, tr.ConsumeTraces(context.Background(), td))
	assert.NoError(t, tr.ConsumeTraces(context.Background(), td))
	assert.NoError(t, tr.ConsumeTraces(context.Background(), td))
	assert.Len(t, sinkRight.AllTraces(), 3)
	assert.Len(t, sinkLeft.AllTraces(), 2)
}

func TestMetricsRouter(t *testing.T) {
	leftID := component.NewIDWithName("sink", "left")
	rightID := component.NewIDWithName("sink", "right")

	sinkLeft := new(consumertest.MetricsSink)
	sinkRight := new(consumertest.MetricsSink)

	// The service will build a router to give to every connector.
	// Many connectors will just call router.ConsumeMetrics,
	// but some implementation will call RouteMetrics instead.
	router := fanoutconsumer.NewMetricsRouter(
		map[component.ID]consumer.Metrics{
			leftID:  sinkLeft,
			rightID: sinkRight,
		})

	cfg := ExampleRouterConfig{Metrics: &LeftRightConfig{Left: leftID, Right: rightID}}
	mr, err := ExampleRouterFactory.CreateMetricsToMetrics(
		context.Background(), connectortest.NewNopCreateSettings(), cfg, router)
	assert.NoError(t, err)
	assert.False(t, mr.Capabilities().MutatesData)

	md := testdata.GenerateMetrics(1)

	assert.NoError(t, mr.ConsumeMetrics(context.Background(), md))
	assert.Len(t, sinkRight.AllMetrics(), 1)
	assert.Len(t, sinkLeft.AllMetrics(), 0)

	assert.NoError(t, mr.ConsumeMetrics(context.Background(), md))
	assert.Len(t, sinkRight.AllMetrics(), 1)
	assert.Len(t, sinkLeft.AllMetrics(), 1)

	assert.NoError(t, mr.ConsumeMetrics(context.Background(), md))
	assert.NoError(t, mr.ConsumeMetrics(context.Background(), md))
	assert.NoError(t, mr.ConsumeMetrics(context.Background(), md))
	assert.Len(t, sinkRight.AllMetrics(), 3)
	assert.Len(t, sinkLeft.AllMetrics(), 2)
}

func TestLogsRouter(t *testing.T) {
	leftID := component.NewIDWithName("sink", "left")
	rightID := component.NewIDWithName("sink", "right")

	sinkLeft := new(consumertest.LogsSink)
	sinkRight := new(consumertest.LogsSink)

	// The service will build a router to give to every connector.
	// Many connectors will just call router.ConsumeLogs,
	// but some implementation will call RouteLogs instead.
	router := fanoutconsumer.NewLogsRouter(
		map[component.ID]consumer.Logs{
			leftID:  sinkLeft,
			rightID: sinkRight,
		})

	cfg := ExampleRouterConfig{Logs: &LeftRightConfig{Left: leftID, Right: rightID}}
	lr, err := ExampleRouterFactory.CreateLogsToLogs(
		context.Background(), connectortest.NewNopCreateSettings(), cfg, router)
	assert.NoError(t, err)
	assert.False(t, lr.Capabilities().MutatesData)

	ld := testdata.GenerateLogs(1)

	assert.NoError(t, lr.ConsumeLogs(context.Background(), ld))
	assert.Len(t, sinkRight.AllLogs(), 1)
	assert.Len(t, sinkLeft.AllLogs(), 0)

	assert.NoError(t, lr.ConsumeLogs(context.Background(), ld))
	assert.Len(t, sinkRight.AllLogs(), 1)
	assert.Len(t, sinkLeft.AllLogs(), 1)

	assert.NoError(t, lr.ConsumeLogs(context.Background(), ld))
	assert.NoError(t, lr.ConsumeLogs(context.Background(), ld))
	assert.NoError(t, lr.ConsumeLogs(context.Background(), ld))
	assert.Len(t, sinkRight.AllLogs(), 3)
	assert.Len(t, sinkLeft.AllLogs(), 2)
}
