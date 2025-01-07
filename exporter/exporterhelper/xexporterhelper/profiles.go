// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package xexporterhelper // import "go.opentelemetry.io/collector/exporter/exporterhelper/xexporterhelper"

import (
	"context"
	"errors"

	"go.uber.org/zap"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer/consumererror"
	"go.opentelemetry.io/collector/consumer/consumererror/xconsumererror"
	"go.opentelemetry.io/collector/consumer/xconsumer"
	"go.opentelemetry.io/collector/exporter"
	"go.opentelemetry.io/collector/exporter/exporterhelper"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal"
	"go.opentelemetry.io/collector/exporter/exporterqueue"
	"go.opentelemetry.io/collector/exporter/xexporter"
	"go.opentelemetry.io/collector/pdata/pprofile"
	"go.opentelemetry.io/collector/pipeline/xpipeline"
)

var (
	profilesMarshaler   = &pprofile.ProtoMarshaler{}
	profilesUnmarshaler = &pprofile.ProtoUnmarshaler{}
)

type profilesRequest struct {
	pd     pprofile.Profiles
	pusher xconsumer.ConsumeProfilesFunc
}

func newProfilesRequest(pd pprofile.Profiles, pusher xconsumer.ConsumeProfilesFunc) exporterhelper.Request {
	return &profilesRequest{
		pd:     pd,
		pusher: pusher,
	}
}

func newProfileRequestUnmarshalerFunc(pusher xconsumer.ConsumeProfilesFunc) exporterqueue.Unmarshaler[exporterhelper.Request] {
	return func(bytes []byte) (exporterhelper.Request, error) {
		profiles, err := profilesUnmarshaler.UnmarshalProfiles(bytes)
		if err != nil {
			return nil, err
		}
		return newProfilesRequest(profiles, pusher), nil
	}
}

func profilesRequestMarshaler(req exporterhelper.Request) ([]byte, error) {
	return profilesMarshaler.MarshalProfiles(req.(*profilesRequest).pd)
}

func (req *profilesRequest) OnError(err error) exporterhelper.Request {
	var profileError xconsumererror.Profiles
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
	*internal.BaseExporter
	xconsumer.Profiles
}

// NewProfilesExporter creates an xexporter.Profiles that records observability metrics and wraps every request with a Span.
func NewProfilesExporter(
	ctx context.Context,
	set exporter.Settings,
	cfg component.Config,
	pusher xconsumer.ConsumeProfilesFunc,
	options ...exporterhelper.Option,
) (xexporter.Profiles, error) {
	if cfg == nil {
		return nil, errNilConfig
	}
	if pusher == nil {
		return nil, errNilPushProfileData
	}
	profilesOpts := []exporterhelper.Option{
		internal.WithMarshaler(profilesRequestMarshaler), internal.WithUnmarshaler(newProfileRequestUnmarshalerFunc(pusher)),
	}
	return NewProfilesRequestExporter(ctx, set, requestFromProfiles(pusher), append(profilesOpts, options...)...)
}

// RequestFromProfilesFunc converts pprofile.Profiles into a user-defined Request.
// Experimental: This API is at the early stage of development and may change without backward compatibility
// until https://github.com/open-telemetry/opentelemetry-collector/issues/8122 is resolved.
type RequestFromProfilesFunc func(context.Context, pprofile.Profiles) (exporterhelper.Request, error)

// requestFromProfiles returns a RequestFromProfilesFunc that converts pprofile.Profiles into a Request.
func requestFromProfiles(pusher xconsumer.ConsumeProfilesFunc) RequestFromProfilesFunc {
	return func(_ context.Context, profiles pprofile.Profiles) (exporterhelper.Request, error) {
		return newProfilesRequest(profiles, pusher), nil
	}
}

// NewProfilesRequestExporter creates a new profiles exporter based on a custom ProfilesConverter and Sender.
// Experimental: This API is at the early stage of development and may change without backward compatibility
// until https://github.com/open-telemetry/opentelemetry-collector/issues/8122 is resolved.
func NewProfilesRequestExporter(
	_ context.Context,
	set exporter.Settings,
	converter RequestFromProfilesFunc,
	options ...exporterhelper.Option,
) (xexporter.Profiles, error) {
	if set.Logger == nil {
		return nil, errNilLogger
	}

	if converter == nil {
		return nil, errNilProfilesConverter
	}

	be, err := internal.NewBaseExporter(set, xpipeline.SignalProfiles, newProfilesExporterWithObservability, options...)
	if err != nil {
		return nil, err
	}

	tc, err := xconsumer.NewProfiles(func(ctx context.Context, pd pprofile.Profiles) error {
		req, cErr := converter(ctx, pd)
		if cErr != nil {
			set.Logger.Error("Failed to convert profiles. Dropping data.",
				zap.Int("dropped_samples", pd.SampleCount()),
				zap.Error(err))
			return consumererror.NewPermanent(cErr)
		}
		return be.Send(ctx, req)
	}, be.ConsumerOptions...)

	return &profileExporter{
		BaseExporter: be,
		Profiles:     tc,
	}, err
}

type profilesExporterWithObservability struct {
	internal.BaseSender[exporterhelper.Request]
	obsrep *internal.ObsReport
}

func newProfilesExporterWithObservability(obsrep *internal.ObsReport) internal.Sender[exporterhelper.Request] {
	return &profilesExporterWithObservability{obsrep: obsrep}
}

func (tewo *profilesExporterWithObservability) Send(ctx context.Context, req exporterhelper.Request) error {
	c := tewo.obsrep.StartProfilesOp(ctx)
	numSamples := req.ItemsCount()
	// Forward the data to the next consumer (this pusher is the next).
	err := tewo.NextSender.Send(c, req)
	tewo.obsrep.EndProfilesOp(c, numSamples, err)
	return err
}
