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
	"go.opentelemetry.io/collector/exporter/xexporter"
	"go.opentelemetry.io/collector/pdata/pprofile"
	"go.opentelemetry.io/collector/pipeline/xpipeline"
)

var (
	profilesMarshaler   = &pprofile.ProtoMarshaler{}
	profilesUnmarshaler = &pprofile.ProtoUnmarshaler{}
)

type profilesRequest struct {
	pd               pprofile.Profiles
	cachedItemsCount int
}

func newProfilesRequest(pd pprofile.Profiles) exporterhelper.Request {
	return &profilesRequest{
		pd:               pd,
		cachedItemsCount: pd.SampleCount(),
	}
}

type profilesEncoding struct{}

func (profilesEncoding) Unmarshal(bytes []byte) (exporterhelper.Request, error) {
	profiles, err := profilesUnmarshaler.UnmarshalProfiles(bytes)
	if err != nil {
		return nil, err
	}
	return newProfilesRequest(profiles), nil
}

func (profilesEncoding) Marshal(req exporterhelper.Request) ([]byte, error) {
	return profilesMarshaler.MarshalProfiles(req.(*profilesRequest).pd)
}

func (req *profilesRequest) OnError(err error) exporterhelper.Request {
	var profileError xconsumererror.Profiles
	if errors.As(err, &profileError) {
		return newProfilesRequest(profileError.Data())
	}
	return req
}

func (req *profilesRequest) ItemsCount() int {
	return req.cachedItemsCount
}

func (req *profilesRequest) setCachedItemsCount(count int) {
	req.cachedItemsCount = count
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
	opts := []exporterhelper.Option{internal.WithEncoding(profilesEncoding{})}
	return NewProfilesRequestExporter(ctx, set, requestFromProfiles(), requestConsumeFromProfiles(pusher), append(opts, options...)...)
}

// Deprecated: [v0.122.0] use exporterhelper.RequestConverterFunc[pprofile.Profiles].
type RequestFromProfilesFunc = exporterhelper.RequestConverterFunc[pprofile.Profiles]

// requestConsumeFromProfiles returns a RequestConsumeFunc that consumes pprofile.Profiles.
func requestConsumeFromProfiles(pusher xconsumer.ConsumeProfilesFunc) exporterhelper.RequestConsumeFunc {
	return func(ctx context.Context, request exporterhelper.Request) error {
		return pusher.ConsumeProfiles(ctx, request.(*profilesRequest).pd)
	}
}

// requestFromProfiles returns a RequestFromProfilesFunc that converts pprofile.Profiles into a Request.
func requestFromProfiles() exporterhelper.RequestConverterFunc[pprofile.Profiles] {
	return func(_ context.Context, profiles pprofile.Profiles) (exporterhelper.Request, error) {
		return newProfilesRequest(profiles), nil
	}
}

// NewProfilesRequestExporter creates a new profiles exporter based on a custom ProfilesConverter and Sender.
// Experimental: This API is at the early stage of development and may change without backward compatibility
// until https://github.com/open-telemetry/opentelemetry-collector/issues/8122 is resolved.
func NewProfilesRequestExporter(
	_ context.Context,
	set exporter.Settings,
	converter exporterhelper.RequestConverterFunc[pprofile.Profiles],
	pusher exporterhelper.RequestConsumeFunc,
	options ...exporterhelper.Option,
) (xexporter.Profiles, error) {
	if set.Logger == nil {
		return nil, errNilLogger
	}

	if converter == nil {
		return nil, errNilProfilesConverter
	}

	if pusher == nil {
		return nil, errNilConsumeRequest
	}

	be, err := internal.NewBaseExporter(set, xpipeline.SignalProfiles, pusher, options...)
	if err != nil {
		return nil, err
	}

	tc, err := xconsumer.NewProfiles(newConsumeProfiles(converter, be, set.Logger), be.ConsumerOptions...)
	if err != nil {
		return nil, err
	}

	return &profileExporter{BaseExporter: be, Profiles: tc}, nil
}

func newConsumeProfiles(converter exporterhelper.RequestConverterFunc[pprofile.Profiles], be *internal.BaseExporter, logger *zap.Logger) xconsumer.ConsumeProfilesFunc {
	return func(ctx context.Context, pd pprofile.Profiles) error {
		req, err := converter(ctx, pd)
		if err != nil {
			logger.Error("Failed to convert profiles. Dropping data.",
				zap.Int("dropped_samples", pd.SampleCount()),
				zap.Error(err))
			return consumererror.NewPermanent(err)
		}
		return be.Send(ctx, req)
	}
}
