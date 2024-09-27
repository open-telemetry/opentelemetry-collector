// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package exporterhelper // import "go.opentelemetry.io/collector/exporter/exporterhelper"

import (
	"context"
	"errors"

	"go.uber.org/zap"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/consumer/consumererror"
	"go.opentelemetry.io/collector/exporter"
	"go.opentelemetry.io/collector/exporter/exporterbatcher"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal"
	"go.opentelemetry.io/collector/exporter/exporterqueue"
	"go.opentelemetry.io/collector/exporter/internal/queue"
	"go.opentelemetry.io/collector/pdata/pentity"
	"go.opentelemetry.io/collector/pipeline"
)

var entitiesMarshaler = &pentity.ProtoMarshaler{}
var entitiesUnmarshaler = &pentity.ProtoUnmarshaler{}

type entitiesRequest struct {
	ld     pentity.Entities
	pusher consumer.ConsumeEntitiesFunc
}

func newEntitiesRequest(ld pentity.Entities, pusher consumer.ConsumeEntitiesFunc) Request {
	return &entitiesRequest{
		ld:     ld,
		pusher: pusher,
	}
}

// Merge merges the provided entities request into the current request and returns the merged request.
func (req *entitiesRequest) Merge(context.Context, Request) (Request, error) {
	// TODO: Implement this method
	return req, nil
}

// MergeSplit splits and/or merges the provided entities request and the current request into one or more requests
// conforming with the MaxSizeConfig.
func (req *entitiesRequest) MergeSplit(context.Context, exporterbatcher.MaxSizeConfig, Request) ([]Request, error) {
	// TODO: Implement this method
	return nil, nil
}

func newEntitiesRequestUnmarshalerFunc(pusher consumer.ConsumeEntitiesFunc) exporterqueue.Unmarshaler[Request] {
	return func(bytes []byte) (Request, error) {
		entities, err := entitiesUnmarshaler.UnmarshalEntities(bytes)
		if err != nil {
			return nil, err
		}
		return newEntitiesRequest(entities, pusher), nil
	}
}

func entitiesRequestMarshaler(req Request) ([]byte, error) {
	return entitiesMarshaler.MarshalEntities(req.(*entitiesRequest).ld)
}

func (req *entitiesRequest) OnError(err error) Request {
	var eError consumererror.Entities
	if errors.As(err, &eError) {
		return newEntitiesRequest(eError.Data(), req.pusher)
	}
	return req
}

func (req *entitiesRequest) Export(ctx context.Context) error {
	return req.pusher(ctx, req.ld)
}

func (req *entitiesRequest) ItemsCount() int {
	return req.ld.EntityCount()
}

type entitiesExporter struct {
	*internal.BaseExporter
	consumer.Entities
}

// NewEntities creates an exporter.Entities that records observability metrics and wraps every request with a Span.
func NewEntities(
	ctx context.Context,
	set exporter.Settings,
	cfg component.Config,
	pusher consumer.ConsumeEntitiesFunc,
	options ...Option,
) (exporter.Entities, error) {
	if cfg == nil {
		return nil, errNilConfig
	}
	if pusher == nil {
		return nil, errNilPushEntitiesData
	}
	entitiesOpts := []Option{
		internal.WithMarshaler(entitiesRequestMarshaler), internal.WithUnmarshaler(newEntitiesRequestUnmarshalerFunc(pusher)),
	}
	return NewEntitiesRequest(ctx, set, requestFromEntities(pusher), append(entitiesOpts, options...)...)
}

// RequestFromEntitiesFunc converts pentity.Entities data into a user-defined request.
// Experimental: This API is at the early stage of development and may change without backward compatibility
// until https://github.com/open-telemetry/opentelemetry-collector/issues/8122 is resolved.
type RequestFromEntitiesFunc func(context.Context, pentity.Entities) (Request, error)

// requestFromEntities returns a RequestFromEntitiesFunc that converts pentity.Entities into a Request.
func requestFromEntities(pusher consumer.ConsumeEntitiesFunc) RequestFromEntitiesFunc {
	return func(_ context.Context, ld pentity.Entities) (Request, error) {
		return newEntitiesRequest(ld, pusher), nil
	}
}

// NewEntitiesRequest creates new entities exporter based on custom EntitiesConverter and RequestSender.
// Experimental: This API is at the early stage of development and may change without backward compatibility
// until https://github.com/open-telemetry/opentelemetry-collector/issues/8122 is resolved.
func NewEntitiesRequest(
	_ context.Context,
	set exporter.Settings,
	converter RequestFromEntitiesFunc,
	options ...Option,
) (exporter.Entities, error) {
	if set.Logger == nil {
		return nil, errNilLogger
	}

	if converter == nil {
		return nil, errNilEntitiesConverter
	}

	be, err := internal.NewBaseExporter(set, pipeline.SignalEntities, newEntitiesWithObservability, options...)
	if err != nil {
		return nil, err
	}

	lc, err := consumer.NewEntities(func(ctx context.Context, ld pentity.Entities) error {
		req, cErr := converter(ctx, ld)
		if cErr != nil {
			set.Logger.Error("Failed to convert entities. Dropping data.",
				zap.Int("dropped_log_records", ld.EntityCount()),
				zap.Error(err))
			return consumererror.NewPermanent(cErr)
		}
		sErr := be.Send(ctx, req)
		if errors.Is(sErr, queue.ErrQueueIsFull) {
			be.Obsrep.RecordEnqueueFailure(ctx, pipeline.SignalEntities, int64(req.ItemsCount()))
		}
		return sErr
	}, be.ConsumerOptions...)

	return &entitiesExporter{
		BaseExporter: be,
		Entities:     lc,
	}, err
}

type entitiesExporterWithObservability struct {
	internal.BaseRequestSender
	obsrep *internal.ObsReport
}

func newEntitiesWithObservability(obsrep *internal.ObsReport) internal.RequestSender {
	return &entitiesExporterWithObservability{obsrep: obsrep}
}

func (lewo *entitiesExporterWithObservability) Send(ctx context.Context, req Request) error {
	c := lewo.obsrep.StartEntitiesOp(ctx)
	numLogRecords := req.ItemsCount()
	err := lewo.NextSender.Send(c, req)
	lewo.obsrep.EndEntitiesOp(c, numLogRecords, err)
	return err
}
