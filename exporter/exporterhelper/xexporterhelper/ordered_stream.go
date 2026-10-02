// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package xexporterhelper

import (
	"context"
	"errors"

	"go.opentelemetry.io/collector/exporter"
	"go.opentelemetry.io/collector/exporter/exporterhelper"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal"
	"go.opentelemetry.io/collector/pdata/plog"
)

// Position describes whether a request continues an open ordered stream or
// ends it at a boundary that permits the next stream to select a new writer.
type Position uint8

const (
	PositionContinue Position = iota
	PositionEnd
)

// Descriptor is one queue-owned item produced from an incoming pdata request.
// The same partition key must be used for every item in one logical stream.
type Descriptor[T any] struct {
	Request      T
	PartitionKey string
	Position     Position
}

// Dispatch is the metadata associated with one ordered queue item while it is
// being written and awaiting its final outcome.
type Dispatch[T any] struct {
	Descriptor[T]
	GroupItems int
	// StreamContext is canceled when the attempt ends, including terminal failure.
	StreamContext context.Context
	StreamAttempt uint64
	Recovery      bool
}

// RequestsConverterFunc expands one pdata input into the ordered queue items
// that represent it. The helper admits the returned descriptors atomically.
type RequestsConverterFunc[T any] func(context.Context, T) ([]Descriptor[T], error)

// AsyncRequestConsumeFunc starts one ordered write phase. It returns once the
// request body is no longer needed; Completion reports the later durability
// outcome.
type AsyncRequestConsumeFunc[T any] func(context.Context, Dispatch[T], Completion) error

// Completion separates permission to write the next item from final queue
// retirement. Release must follow the item's complete write phase. Succeed or
// Fail records its eventual acknowledgement outcome.
type Completion interface {
	Release()
	Succeed()
	Fail(error)
}

// OrderedStreamSettings bounds the in-memory coordinator state used by an
// ordered stream exporter. These limits are independent of the helper queue's
// admission and persistence capacity.
type OrderedStreamSettings struct {
	// MaxConcurrentWrites bounds concurrently dispatched partitions. The helper
	// reads its persistent queue with one worker to preserve queue insertion
	// order, then uses this limit for independent partition writes.
	MaxConcurrentWrites  int
	MaxStaged            int
	MaxActivePartitions  int
	MaxReleasedRequests  int
	MaxReleasedBytes     int
	MaxRecoveryTailBytes int
	MaxGroupRequests     int
	MaxGroupItems        int
	MaxGroupBytes        int
	MaxPartitionKeyBytes int
}

// NewLogsRequests creates a logs exporter that atomically admits each
// converter result and schedules child requests in per-partition order.
func NewLogsRequests(
	ctx context.Context,
	set exporter.Settings,
	converter RequestsConverterFunc[plog.Logs],
	pusher AsyncRequestConsumeFunc[plog.Logs],
	limits OrderedStreamSettings,
	options ...exporterhelper.Option,
) (exporter.Logs, error) {
	var internalConverter internal.OrderedLogsConverterFunc
	if converter != nil {
		internalConverter = func(ctx context.Context, ld plog.Logs) ([]internal.OrderedLogsDescriptor, error) {
			descriptors, err := converter(ctx, ld)
			if err != nil {
				return nil, err
			}
			result := make([]internal.OrderedLogsDescriptor, 0, len(descriptors))
			for _, descriptor := range descriptors {
				position := internal.OrderedPositionContinue
				if descriptor.Position == PositionEnd {
					position = internal.OrderedPositionEnd
				} else if descriptor.Position != PositionContinue {
					return nil, errors.New("ordered stream descriptor has an invalid position")
				}
				result = append(result, internal.OrderedLogsDescriptor{
					Request: descriptor.Request, PartitionKey: descriptor.PartitionKey, Position: position,
				})
			}
			return result, nil
		}
	}
	var internalPusher internal.OrderedLogsConsumeFunc
	if pusher != nil {
		internalPusher = func(ctx context.Context, dispatch internal.OrderedLogsDispatch, completion internal.OrderedLogsCompletion) error {
			position := PositionContinue
			if dispatch.Position == internal.OrderedPositionEnd {
				position = PositionEnd
			}
			return pusher(ctx, Dispatch[plog.Logs]{
				Descriptor: Descriptor[plog.Logs]{Request: dispatch.Request, PartitionKey: dispatch.PartitionKey, Position: position},
				GroupItems: dispatch.GroupItems, StreamContext: dispatch.StreamContext, StreamAttempt: dispatch.StreamAttempt, Recovery: dispatch.Recovery,
			}, completion)
		}
	}
	return internal.NewLogsRequests(ctx, set, internalConverter, internalPusher, internal.OrderedLogsSettings(limits), options...)
}

// Validate rejects unbounded or internally inconsistent stream limits.
func (s OrderedStreamSettings) Validate() error {
	return internal.OrderedLogsSettings(s).Validate()
}
