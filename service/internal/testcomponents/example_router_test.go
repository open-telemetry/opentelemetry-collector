// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package testcomponents

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/connector"
	"go.opentelemetry.io/collector/connector/connectortest"
	"go.opentelemetry.io/collector/connector/xconnector"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/consumer/consumertest"
	"go.opentelemetry.io/collector/consumer/xconsumer"
	"go.opentelemetry.io/collector/pdata/testdata"
	"go.opentelemetry.io/collector/pipeline"
)

func TestExampleRouter(t *testing.T) {
	conn := &ExampleRouter{}
	host := componenttest.NewNopHost()
	assert.False(t, conn.Started())
	require.NoError(t, conn.Start(context.Background(), host))
	assert.True(t, conn.Started())

	assert.False(t, conn.Stopped())
	require.NoError(t, conn.Shutdown(context.Background()))
	assert.True(t, conn.Stopped())
}

func TestTracesRouter(t *testing.T) {
	leftID := pipeline.NewIDWithName(pipeline.SignalTraces, "sink_left")
	rightID := pipeline.NewIDWithName(pipeline.SignalTraces, "sink_right")

	sinkLeft := new(consumertest.TracesSink)
	sinkRight := new(consumertest.TracesSink)

	// The service will build a router to give to every connector.
	// Many connectors will just call router.ConsumeTraces,
	// but some implementation will call RouteTraces instead.
	router := connector.NewTracesRouter(
		map[pipeline.ID]consumer.Traces{
			leftID:  sinkLeft,
			rightID: sinkRight,
		})

	cfg := ExampleRouterConfig{Traces: &LeftRightConfig{Left: leftID, Right: rightID}}
	tr, err := ExampleRouterFactory.CreateTracesToTraces(
		context.Background(), connectortest.NewNopSettings(ExampleRouterFactory.Type()), cfg, router)
	require.NoError(t, err)
	assert.False(t, tr.Capabilities().MutatesData)

	td := testdata.GenerateTraces(1)

	require.NoError(t, tr.ConsumeTraces(context.Background(), td))
	assert.Len(t, sinkRight.AllTraces(), 1)
	assert.Empty(t, sinkLeft.AllTraces())

	require.NoError(t, tr.ConsumeTraces(context.Background(), td))
	assert.Len(t, sinkRight.AllTraces(), 1)
	assert.Len(t, sinkLeft.AllTraces(), 1)

	assert.NoError(t, tr.ConsumeTraces(context.Background(), td))
	assert.NoError(t, tr.ConsumeTraces(context.Background(), td))
	assert.NoError(t, tr.ConsumeTraces(context.Background(), td))
	assert.Len(t, sinkRight.AllTraces(), 3)
	assert.Len(t, sinkLeft.AllTraces(), 2)
}

func TestMetricsRouter(t *testing.T) {
	leftID := pipeline.NewIDWithName(pipeline.SignalTraces, "sink_left")
	rightID := pipeline.NewIDWithName(pipeline.SignalTraces, "sink_right")

	sinkLeft := new(consumertest.MetricsSink)
	sinkRight := new(consumertest.MetricsSink)

	// The service will build a router to give to every connector.
	// Many connectors will just call router.ConsumeMetrics,
	// but some implementation will call RouteMetrics instead.
	router := connector.NewMetricsRouter(
		map[pipeline.ID]consumer.Metrics{
			leftID:  sinkLeft,
			rightID: sinkRight,
		})

	cfg := ExampleRouterConfig{Metrics: &LeftRightConfig{Left: leftID, Right: rightID}}
	mr, err := ExampleRouterFactory.CreateMetricsToMetrics(
		context.Background(), connectortest.NewNopSettings(ExampleRouterFactory.Type()), cfg, router)
	require.NoError(t, err)
	assert.False(t, mr.Capabilities().MutatesData)

	md := testdata.GenerateMetrics(1)

	require.NoError(t, mr.ConsumeMetrics(context.Background(), md))
	assert.Len(t, sinkRight.AllMetrics(), 1)
	assert.Empty(t, sinkLeft.AllMetrics())

	require.NoError(t, mr.ConsumeMetrics(context.Background(), md))
	assert.Len(t, sinkRight.AllMetrics(), 1)
	assert.Len(t, sinkLeft.AllMetrics(), 1)

	assert.NoError(t, mr.ConsumeMetrics(context.Background(), md))
	assert.NoError(t, mr.ConsumeMetrics(context.Background(), md))
	assert.NoError(t, mr.ConsumeMetrics(context.Background(), md))
	assert.Len(t, sinkRight.AllMetrics(), 3)
	assert.Len(t, sinkLeft.AllMetrics(), 2)
}

func TestLogsRouter(t *testing.T) {
	leftID := pipeline.NewIDWithName(pipeline.SignalTraces, "sink_left")
	rightID := pipeline.NewIDWithName(pipeline.SignalTraces, "sink_right")

	sinkLeft := new(consumertest.LogsSink)
	sinkRight := new(consumertest.LogsSink)

	// The service will build a router to give to every connector.
	// Many connectors will just call router.ConsumeLogs,
	// but some implementation will call RouteLogs instead.
	router := connector.NewLogsRouter(
		map[pipeline.ID]consumer.Logs{
			leftID:  sinkLeft,
			rightID: sinkRight,
		})

	cfg := ExampleRouterConfig{Logs: &LeftRightConfig{Left: leftID, Right: rightID}}
	lr, err := ExampleRouterFactory.CreateLogsToLogs(
		context.Background(), connectortest.NewNopSettings(ExampleRouterFactory.Type()), cfg, router)
	require.NoError(t, err)
	assert.False(t, lr.Capabilities().MutatesData)

	ld := testdata.GenerateLogs(1)

	require.NoError(t, lr.ConsumeLogs(context.Background(), ld))
	assert.Len(t, sinkRight.AllLogs(), 1)
	assert.Empty(t, sinkLeft.AllLogs())

	require.NoError(t, lr.ConsumeLogs(context.Background(), ld))
	assert.Len(t, sinkRight.AllLogs(), 1)
	assert.Len(t, sinkLeft.AllLogs(), 1)

	assert.NoError(t, lr.ConsumeLogs(context.Background(), ld))
	assert.NoError(t, lr.ConsumeLogs(context.Background(), ld))
	assert.NoError(t, lr.ConsumeLogs(context.Background(), ld))
	assert.Len(t, sinkRight.AllLogs(), 3)
	assert.Len(t, sinkLeft.AllLogs(), 2)
}

func TestProfilesRouter(t *testing.T) {
	leftID := pipeline.NewIDWithName(pipeline.SignalTraces, "sink_left")
	rightID := pipeline.NewIDWithName(pipeline.SignalTraces, "sink_right")

	sinkLeft := new(consumertest.ProfilesSink)
	sinkRight := new(consumertest.ProfilesSink)

	// The service will build a router to give to every connector.
	// Many connectors will just call router.ConsumeProfiles,
	// but some implementation will call RouteProfiles instead.
	router := xconnector.NewProfilesRouter(
		map[pipeline.ID]xconsumer.Profiles{
			leftID:  sinkLeft,
			rightID: sinkRight,
		})

	cfg := ExampleRouterConfig{Profiles: &LeftRightConfig{Left: leftID, Right: rightID}}
	tr, err := ExampleRouterFactory.CreateProfilesToProfiles(
		context.Background(), connectortest.NewNopSettings(ExampleRouterFactory.Type()), cfg, router)
	require.NoError(t, err)
	assert.False(t, tr.Capabilities().MutatesData)

	td := testdata.GenerateProfiles(1)

	require.NoError(t, tr.ConsumeProfiles(context.Background(), td))
	assert.Len(t, sinkRight.AllProfiles(), 1)
	assert.Empty(t, sinkLeft.AllProfiles())

	require.NoError(t, tr.ConsumeProfiles(context.Background(), td))
	assert.Len(t, sinkRight.AllProfiles(), 1)
	assert.Len(t, sinkLeft.AllProfiles(), 1)

	assert.NoError(t, tr.ConsumeProfiles(context.Background(), td))
	assert.NoError(t, tr.ConsumeProfiles(context.Background(), td))
	assert.NoError(t, tr.ConsumeProfiles(context.Background(), td))
	assert.Len(t, sinkRight.AllProfiles(), 3)
	assert.Len(t, sinkLeft.AllProfiles(), 2)
}
