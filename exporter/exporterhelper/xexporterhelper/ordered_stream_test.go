// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package xexporterhelper

import (
	"context"
	"fmt"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/config/configoptional"
	"go.opentelemetry.io/collector/exporter/exporterhelper"
	"go.opentelemetry.io/collector/exporter/exportertest"
	"go.opentelemetry.io/collector/pdata/plog"
)

type orderedLogsTestResult struct {
	dispatch   Dispatch[plog.Logs]
	completion Completion
}

func orderedStreamTestLogs() plog.Logs {
	return orderedStreamTestLogsWithBody("")
}

func orderedStreamTestLogsWithBody(body string) plog.Logs {
	ld := plog.NewLogs()
	ld.ResourceLogs().AppendEmpty().ScopeLogs().AppendEmpty().LogRecords().AppendEmpty().Body().SetStr(body)
	return ld
}

func orderedStreamTestBody(ld plog.Logs) string {
	return ld.ResourceLogs().At(0).ScopeLogs().At(0).LogRecords().At(0).Body().Str()
}

func orderedStreamTestSettings() OrderedStreamSettings {
	return OrderedStreamSettings{
		MaxStaged: 16, MaxActivePartitions: 4, MaxReleasedRequests: 16,
		MaxReleasedBytes: 1 << 20, MaxRecoveryTailBytes: 1 << 20,
		MaxGroupRequests: 8, MaxGroupItems: 16, MaxGroupBytes: 1 << 20,
		MaxPartitionKeyBytes: 128,
	}
}

func TestNewLogsRequestsReleasesSuccessorBeforeFirstCompletion(t *testing.T) {
	for _, consumers := range []int{1, 32} {
		t.Run(fmt.Sprintf("num_consumers_%d", consumers), func(t *testing.T) {
			dispatched := make(chan orderedLogsTestResult, 2)
			queueCfg := exporterhelper.NewDefaultQueueConfig()
			queueCfg.NumConsumers = consumers
			queueCfg.Batch = configoptional.Optional[exporterhelper.BatchConfig]{}
			exp, err := NewLogsRequests(
				context.Background(), exportertest.NewNopSettings(exportertest.NopType),
				func(_ context.Context, ld plog.Logs) ([]Descriptor[plog.Logs], error) {
					return []Descriptor[plog.Logs]{{Request: ld, PartitionKey: "one-input-channel", Position: PositionContinue}}, nil
				},
				func(_ context.Context, dispatch Dispatch[plog.Logs], completion Completion) error {
					dispatched <- orderedLogsTestResult{dispatch: dispatch, completion: completion}
					completion.Release()
					return nil
				}, orderedStreamTestSettings(),
				WithQueueBatch(configoptional.Some(queueCfg), NewLogsQueueBatchSettings()),
			)
			require.NoError(t, err)
			require.NoError(t, exp.Start(context.Background(), componenttest.NewNopHost()))
			t.Cleanup(func() { require.NoError(t, exp.Shutdown(context.Background())) })

			require.NoError(t, exp.ConsumeLogs(context.Background(), orderedStreamTestLogs()))
			first := receiveOrderedLogsResult(t, dispatched)
			require.NoError(t, exp.ConsumeLogs(context.Background(), orderedStreamTestLogs()))
			second := receiveOrderedLogsResult(t, dispatched)
			require.Equal(t, first.dispatch.PartitionKey, second.dispatch.PartitionKey)
			second.completion.Succeed()
			first.completion.Succeed()
		})
	}
}

func receiveOrderedLogsResult(t *testing.T, dispatched <-chan orderedLogsTestResult) orderedLogsTestResult {
	t.Helper()
	select {
	case result := <-dispatched:
		return result
	case <-time.After(time.Second):
		t.Fatal("ordered stream successor was not dispatched")
		return orderedLogsTestResult{}
	}
}

func TestNewLogsRequestsRequestSizerChargesEveryDescriptor(t *testing.T) {
	for _, capacity := range []int64{1, 4} {
		t.Run(strconv.FormatInt(capacity, 10), func(t *testing.T) {
			q := exporterhelper.NewDefaultQueueConfig()
			q.QueueSize = capacity
			q.Batch = configoptional.Optional[exporterhelper.BatchConfig]{}
			written := make(chan orderedLogsTestResult, 4)
			exp, err := NewLogsRequests(context.Background(), exportertest.NewNopSettings(exportertest.NopType),
				func(context.Context, plog.Logs) ([]Descriptor[plog.Logs], error) {
					var descriptors []Descriptor[plog.Logs]
					for i := range 4 {
						descriptors = append(descriptors, Descriptor[plog.Logs]{Request: orderedStreamTestLogsWithBody(strconv.Itoa(i)), PartitionKey: "p"})
					}
					descriptors[3].Position = PositionEnd
					return descriptors, nil
				}, func(_ context.Context, d Dispatch[plog.Logs], done Completion) error {
					written <- orderedLogsTestResult{d, done}
					done.Release()
					return nil
				},
				orderedStreamTestSettings(), WithQueueBatch(configoptional.Some(q), NewLogsQueueBatchSettings()))
			require.NoError(t, err)
			require.NoError(t, exp.Start(context.Background(), componenttest.NewNopHost()))
			t.Cleanup(func() { require.NoError(t, exp.Shutdown(context.Background())) })
			err = exp.ConsumeLogs(context.Background(), orderedStreamTestLogs())
			if capacity == 1 {
				require.Error(t, err)
				require.Empty(t, written)
				return
			}
			require.NoError(t, err)
			var pending []orderedLogsTestResult
			for i := range 4 {
				d := receiveOrderedLogsResult(t, written)
				require.Equal(t, strconv.Itoa(i), orderedStreamTestBody(d.dispatch.Request))
				pending = append(pending, d)
			}
			for _, d := range pending {
				d.completion.Succeed()
			}
		})
	}
}

func TestNewLogsRequestsRejectsPersistentQueue(t *testing.T) {
	q := exporterhelper.NewDefaultQueueConfig()
	storageID := component.NewID(component.MustNewType("storage"))
	q.StorageID = &storageID
	q.Batch = configoptional.Optional[exporterhelper.BatchConfig]{}
	exp, err := NewLogsRequests(context.Background(), exportertest.NewNopSettings(exportertest.NopType),
		func(_ context.Context, ld plog.Logs) ([]Descriptor[plog.Logs], error) {
			return []Descriptor[plog.Logs]{{Request: ld, PartitionKey: "p"}}, nil
		}, func(context.Context, Dispatch[plog.Logs], Completion) error { return nil },
		orderedStreamTestSettings(), WithQueueBatch(configoptional.Some(q), NewLogsQueueBatchSettings()))
	require.ErrorContains(t, err, "ordered stream persistent queues are not supported")
	require.Nil(t, exp)
}
