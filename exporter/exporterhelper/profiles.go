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
	"go.opentelemetry.io/collector/exporter/exporterqueue"
	"go.opentelemetry.io/collector/exporter/internal/queue"
	"go.opentelemetry.io/collector/pdata/pprofile"
)

var profilesMarshaler = &pprofile.ProtoMarshaler{}
var profilesUnmarshaler = &pprofile.ProtoUnmarshaler{}

type profilesRequest struct {
	pd     pprofile.Profiles
	pusher consumer.ConsumeProfilesFunc
}

func newProfilesRequest(pd pprofile.Profiles, pusher consumer.ConsumeProfilesFunc) Request {
	return &profilesRequest{
		pd:     pd,
		pusher: pusher,
	}
}

func newProfilesRequestUnmarshalerFunc(pusher consumer.ConsumeProfilesFunc) exporterqueue.Unmarshaler[Request] {
	return func(bytes []byte) (Request, error) {
		profiles, err := profilesUnmarshaler.UnmarshalProfiles(bytes)
		if err != nil {
			return nil, err
		}
		return newProfilesRequest(profiles, pusher), nil
	}
}

func profilesRequestMarshaler(req Request) ([]byte, error) {
	return profilesMarshaler.MarshalProfiles(req.(*profilesRequest).pd)
}

func (req *profilesRequest) OnError(err error) Request {
	var profileError consumererror.Profiles
	if errors.As(err, &profileError) {
		return newProfilesRequest(profileError.Data(), req.pusher)
	}
	return req
}

func (req *profilesRequest) ItemsCount() int {
	return req.pd.ProfileCount()
}

func (req *profilesRequest) Export(ctx context.Context) error {
	return req.pusher(ctx, req.pd)
}

func (req *profilesRequest) Marshal() ([]byte, error) {
	return profilesMarshaler.MarshalProfiles(req.pd)
}

func (req *profilesRequest) Count() int {
	return req.pd.ProfileCount()
}

type profilesExporter struct {
	*baseExporter
	consumer.Profiles
}

// NewProfilesExporter creates an exporter.Profiles that records observability metrics and wraps every request with a Span.
func NewProfilesExporter(
	ctx context.Context,
	set exporter.CreateSettings,
	cfg component.Config,
	pusher consumer.ConsumeProfilesFunc,
	options ...Option,
) (exporter.Profiles, error) {
	if cfg == nil {
		return nil, errNilConfig
	}
	if pusher == nil {
		return nil, errNilPushProfilesData
	}
	profilesOpts := []Option{
		withMarshaler(profilesRequestMarshaler), withUnmarshaler(newProfilesRequestUnmarshalerFunc(pusher)),
		withBatchFuncs(mergeProfiles, mergeSplitProfiles),
	}
	return NewProfilesRequestExporter(ctx, set, requestFromProfiles(pusher), append(profilesOpts, options...)...)
}

// RequestFromProfilesFunc converts pprofile.Profiles data into a user-defined request.
// Experimental: This API is at the early stage of development and may change without backward compatibility
// until https://github.com/open-telemetry/opentelemetry-collector/issues/8122 is resolved.
type RequestFromProfilesFunc func(context.Context, pprofile.Profiles) (Request, error)

// requestFromProfiles returns a RequestFromProfilesFunc that converts pprofile.Profiles into a Request.
func requestFromProfiles(pusher consumer.ConsumeProfilesFunc) RequestFromProfilesFunc {
	return func(_ context.Context, pd pprofile.Profiles) (Request, error) {
		return newProfilesRequest(pd, pusher), nil
	}
}

// NewProfilesRequestExporter creates new profiles exporter based on custom ProfilesConverter and RequestSender.
// Experimental: This API is at the early stage of development and may change without backward compatibility
// until https://github.com/open-telemetry/opentelemetry-collector/issues/8122 is resolved.
func NewProfilesRequestExporter(
	_ context.Context,
	set exporter.CreateSettings,
	converter RequestFromProfilesFunc,
	options ...Option,
) (exporter.Profiles, error) {
	if set.Logger == nil {
		return nil, errNilLogger
	}

	if converter == nil {
		return nil, errNilProfilesConverter
	}

	be, err := newBaseExporter(set, component.DataTypeProfiles, newProfilesExporterWithObservability, options...)
	if err != nil {
		return nil, err
	}

	lc, err := consumer.NewProfiles(func(ctx context.Context, ld pprofile.Profiles) error {
		req, cErr := converter(ctx, ld)
		if cErr != nil {
			set.Logger.Error("Failed to convert profiles. Dropping data.",
				zap.Int("dropped_profiles", ld.ProfileCount()),
				zap.Error(err))
			return consumererror.NewPermanent(cErr)
		}
		sErr := be.send(ctx, req)
		if errors.Is(sErr, queue.ErrQueueIsFull) {
			be.obsrep.recordEnqueueFailure(ctx, component.DataTypeProfiles, int64(req.ItemsCount()))
		}
		return sErr
	}, be.consumerOptions...)

	return &profilesExporter{
		baseExporter: be,
		Profiles:     lc,
	}, err
}

type profilesExporterWithObservability struct {
	baseRequestSender
	obsrep *ObsReport
}

func newProfilesExporterWithObservability(obsrep *ObsReport) requestSender {
	return &profilesExporterWithObservability{obsrep: obsrep}
}

func (pewo *profilesExporterWithObservability) send(ctx context.Context, req Request) error {
	c := pewo.obsrep.StartProfilesOp(ctx)
	err := pewo.nextSender.send(c, req)
	pewo.obsrep.EndProfilesOp(c, req.ItemsCount(), err)
	return err
}
