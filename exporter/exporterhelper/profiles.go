// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package exporterhelper // import "go.opentelemetry.io/collector/exporter/exporterhelper"

import (
	"context"
	"errors"

	"go.uber.org/zap"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componentprofiles"
	"go.opentelemetry.io/collector/consumer/consumererror"
	"go.opentelemetry.io/collector/consumer/consumererror/consumererrorprofiles"
	"go.opentelemetry.io/collector/consumer/consumerprofiles"
	"go.opentelemetry.io/collector/exporter"
	"go.opentelemetry.io/collector/exporter/exporterprofiles"
	"go.opentelemetry.io/collector/exporter/exporterqueue"
	"go.opentelemetry.io/collector/pdata/pprofile"
)

var profilesMarshaler = &pprofile.ProtoMarshaler{}
var profilesUnmarshaler = &pprofile.ProtoUnmarshaler{}

type profilesRequest struct {
	pd     pprofile.Profiles
	pusher consumerprofiles.ConsumeProfilesFunc
}

func newProfilesRequest(pd pprofile.Profiles, pusher consumerprofiles.ConsumeProfilesFunc) Request {
	return &profilesRequest{
		pd:     pd,
		pusher: pusher,
	}
}

func newProfileRequestUnmarshalerFunc(pusher consumerprofiles.ConsumeProfilesFunc) exporterqueue.Unmarshaler[Request] {
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
	var profileError consumererrorprofiles.Profiles
	if errors.As(err, &profileError) {
		return newProfilesRequest(profileError.Data(), req.pusher)
	}
	return req
}

func (req *profilesRequest) Export(ctx context.Context) error {
	return req.pusher(ctx, req.pd)
}

func (req *profilesRequest) ItemsCount() int {
	return req.pd.SampleCount()
}

type profileExporter struct {
	*baseExporter
	consumerprofiles.Profiles
}

// NewProfilesExporter creates an exporterprofiles.Profiless that records observability metrics and wraps every request with a Span.
func NewProfilesExporter(
	ctx context.Context,
	set exporter.Settings,
	cfg component.Config,
	pusher consumerprofiles.ConsumeProfilesFunc,
	options ...Option,
) (exporterprofiles.Profiles, error) {
	if cfg == nil {
		return nil, errNilConfig
	}
	if pusher == nil {
		return nil, errNilPushProfileData
	}
	profilesOpts := []Option{
		withMarshaler(profilesRequestMarshaler), withUnmarshaler(newProfileRequestUnmarshalerFunc(pusher)),
		withBatchFuncs(mergeProfiles, mergeSplitProfiles),
	}
	return NewProfilesRequestExporter(ctx, set, requestFromProfiles(pusher), append(profilesOpts, options...)...)
}

// RequestFromProfilesFunc converts pprofile.Profiles into a user-defined Request.
// Experimental: This API is at the early stage of development and may change without backward compatibility
// until https://github.com/open-telemetry/opentelemetry-collector/issues/8122 is resolved.
type RequestFromProfilesFunc func(context.Context, pprofile.Profiles) (Request, error)

// requestFromProfiles returns a RequestFromProfilesFunc that converts pprofile.Profiles into a Request.
func requestFromProfiles(pusher consumerprofiles.ConsumeProfilesFunc) RequestFromProfilesFunc {
	return func(_ context.Context, profiles pprofile.Profiles) (Request, error) {
		return newProfilesRequest(profiles, pusher), nil
	}
}

// NewProfilesRequestExporter creates a new profiles exporter based on a custom ProfilesConverter and RequestSender.
// Experimental: This API is at the early stage of development and may change without backward compatibility
// until https://github.com/open-telemetry/opentelemetry-collector/issues/8122 is resolved.
func NewProfilesRequestExporter(
	_ context.Context,
	set exporter.Settings,
	converter RequestFromProfilesFunc,
	options ...Option,
) (exporterprofiles.Profiles, error) {
	if set.Logger == nil {
		return nil, errNilLogger
	}

	if converter == nil {
		return nil, errNilProfilesConverter
	}

	be, err := newBaseExporter(set, componentprofiles.DataTypeProfiles, newProfilesExporterWithObservability, options...)
	if err != nil {
		return nil, err
	}

	tc, err := consumerprofiles.NewProfiles(func(ctx context.Context, pd pprofile.Profiles) error {
		req, cErr := converter(ctx, pd)
		if cErr != nil {
			set.Logger.Error("Failed to convert profiles. Dropping data.",
				zap.Int("dropped_samples", pd.SampleCount()),
				zap.Error(err))
			return consumererror.NewPermanent(cErr)
		}
		return be.send(ctx, req)
	}, be.consumerOptions...)

	return &profileExporter{
		baseExporter: be,
		Profiles:     tc,
	}, err
}

type profilesExporterWithObservability struct {
	baseRequestSender
	obsrep *obsReport
}

func newProfilesExporterWithObservability(obsrep *obsReport) requestSender {
	return &profilesExporterWithObservability{obsrep: obsrep}
}

func (tewo *profilesExporterWithObservability) send(ctx context.Context, req Request) error {
	c := tewo.obsrep.startProfilesOp(ctx)
	numSamples := req.ItemsCount()
	// Forward the data to the next consumer (this pusher is the next).
	err := tewo.nextSender.send(c, req)
	tewo.obsrep.endProfilesOp(c, numSamples, err)
	return err
}
