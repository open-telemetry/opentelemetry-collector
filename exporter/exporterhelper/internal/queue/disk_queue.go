// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package queue // import "go.opentelemetry.io/collector/exporter/exporterhelper/internal/queue"

import (
	"context"
	"errors"

	"go.uber.org/zap"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/queue/diskaccess"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/request"
	"go.opentelemetry.io/collector/pipeline"
)

var (
	_ Queue[request.Request]         = (*diskQueue[request.Request])(nil)
	_ readableQueue[request.Request] = (*diskQueue[request.Request])(nil)
)

var (
	errNoDiskAccessClient           = errors.New("no disk access client extension found")
	errDiskAccessWrongExtensionType = errors.New("requested extension is not a disk access extension")
)

type diskQueue[T request.Request] struct {
	logger           *zap.Logger
	encoding         Encoding[T]
	capacity         int64
	sizerType        request.SizerType
	activeSizer      request.Sizer
	refCounter       ReferenceCounter[T]
	storageID        component.ID
	signalType       pipeline.Signal
	id               component.ID
	diskAccessClient diskaccess.Client
}

func newDiskQueue[T request.Request](set Settings[T]) readableQueue[T] {
	d := diskQueue[T]{
		id:          set.ID,
		storageID:   *set.StorageID,
		logger:      set.Telemetry.Logger,
		encoding:    set.Encoding,
		capacity:    set.Capacity,
		sizerType:   set.SizerType,
		activeSizer: request.NewSizer(set.SizerType),
		refCounter:  set.ReferenceCounter,
		signalType:  set.Signal,
	}

	return &d
}

func toDiskAccessClient(ctx context.Context, storageID component.ID, host component.Host, ownerID component.ID, signal pipeline.Signal) (diskaccess.Client, error) {
	ext, found := host.GetExtensions()[storageID]
	if !found {
		return nil, errNoDiskAccessClient
	}

	storageExt, ok := ext.(diskaccess.Extension)
	if !ok {
		return nil, errDiskAccessWrongExtensionType
	}

	return storageExt.GetClient(ctx, component.KindExporter, ownerID, signal.String())
}

func (d *diskQueue[T]) Start(ctx context.Context, host component.Host) error {
	c, err := toDiskAccessClient(ctx, d.storageID, host, d.id, d.signalType)
	if err != nil {
		return err
	}
	d.diskAccessClient = c
	return nil
}

func (d *diskQueue[T]) Shutdown(ctx context.Context) error {
	if d.diskAccessClient == nil {
		return nil
	}
	return d.diskAccessClient.Shutdown(ctx)
}

func (d *diskQueue[T]) Offer(ctx context.Context, item T) error {
	size := d.activeSizer.Sizeof(item)
	// Ignore empty requests, see https://github.com/open-telemetry/opentelemetry-proto/blob/main/docs/specification.md#empty-telemetry-envelopes
	if size == 0 {
		return nil
	}

	if size <= 0 {
		return errInvalidSize
	}

	// If element larger than the capacity, will never been able to add it.
	if size > d.capacity {
		return errSizeTooLarge
	}

	if d.capacity < (d.diskAccessClient.Size() + size) {
		return ErrQueueIsFull
	}
	if d.refCounter != nil {
		d.refCounter.Ref(item)
	}
	b, err := d.encoding.Marshal(ctx, item)
	if err != nil {
		// Unref in case of an error since there will not be any async worker to pick it up.
		if d.refCounter != nil {
			d.refCounter.Unref(item)
		}
		return err
	}

	if err := d.diskAccessClient.Write(diskaccess.WriteOp{
		Payload: b,
		Size:    size,
	}); err != nil {
		// Unref in case of an error since there will not be any async worker to pick it up.
		if d.refCounter != nil {
			d.refCounter.Unref(item)
		}
		return err
	}
	return nil
}

func (d *diskQueue[T]) Size() int64 {
	return d.diskAccessClient.Size()
}

func (d *diskQueue[T]) Capacity() int64 {
	return d.capacity
}

type onDoneFunc func(err error)

func (o onDoneFunc) OnDone(err error) {
	o(err)
}

func (d *diskQueue[T]) Read(_ context.Context) (context.Context, T, Done, bool) {
	peekChan := d.diskAccessClient.Peek()
	if peekChan == nil {
		// The client has already shut down; there will never be another item.
		var el T
		return context.Background(), el, nil, false
	}
	msg, ok := <-peekChan
	if !ok {
		// The client shut down while we were waiting for an item.
		var el T
		return context.Background(), el, nil, false
	}
	restoredCtx, el, err := d.encoding.Unmarshal(msg.Payload)
	if err != nil {
		d.logger.Debug("Failed to unmarshall item", zap.Error(err))
		// We need to make sure that currently dispatched items list is cleaned
		msg.ConsumeCallback(err)
		return restoredCtx, el, nil, false
	}
	var fn onDoneFunc = func(err error) {
		msg.ConsumeCallback(err)
		if d.refCounter != nil {
			d.refCounter.Unref(el)
		}
	}
	return restoredCtx, el, fn, true
}
