// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package exporterhelper // import "go.opentelemetry.io/collector/exporter/exporterhelper"

import (
	"context"
	"errors"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/consumer/consumererror"
	"go.opentelemetry.io/collector/exporter"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal"
	"go.opentelemetry.io/collector/pdata/pprofile"
)

var profilesMarshaler = &pprofile.ProtoMarshaler{}
var profilesUnmarshaler = &pprofile.ProtoUnmarshaler{}

type profilesRequest struct {
	baseRequest
	ld     pprofile.Profiles
	pusher consumer.ConsumeProfilesFunc
}

func newProfilesRequest(ctx context.Context, ld pprofile.Profiles, pusher consumer.ConsumeProfilesFunc) internal.Request {
	return &profilesRequest{
		baseRequest: baseRequest{ctx: ctx},
		ld:          ld,
		pusher:      pusher,
	}
}

func newProfilesRequestUnmarshalerFunc(pusher consumer.ConsumeProfilesFunc) internal.RequestUnmarshaler {
	return func(bytes []byte) (internal.Request, error) {
		profiles, err := profilesUnmarshaler.UnmarshalProfiles(bytes)
		if err != nil {
			return nil, err
		}
		return newProfilesRequest(context.Background(), profiles, pusher), nil
	}
}

func (req *profilesRequest) OnError(err error) internal.Request {
	var profileError consumererror.Profiles
	if errors.As(err, &profileError) {
		return newProfilesRequest(req.ctx, profileError.Data(), req.pusher)
	}
	return req
}

func (req *profilesRequest) Export(ctx context.Context) error {
	return req.pusher(ctx, req.ld)
}

func (req *profilesRequest) Marshal() ([]byte, error) {
	return profilesMarshaler.MarshalProfiles(req.ld)
}

func (req *profilesRequest) Count() int {
	return req.ld.ProfileRecordCount()
}

type profilesExporter struct {
	*baseExporter
	consumer.Profiles
}

// NewProfilesExporter creates an exporter.Profiles that records observability metrics and wraps every request with a Span.
func NewProfilesExporter(
	_ context.Context,
	set exporter.CreateSettings,
	cfg component.Config,
	pusher consumer.ConsumeProfilesFunc,
	options ...Option,
) (exporter.Profiles, error) {
	if cfg == nil {
		return nil, errNilConfig
	}

	if set.Logger == nil {
		return nil, errNilLogger
	}

	if pusher == nil {
		return nil, errNilPushProfilesData
	}

	bs := fromOptions(options...)
	be, err := newBaseExporter(set, bs, component.DataTypeProfiles, newProfilesRequestUnmarshalerFunc(pusher))
	if err != nil {
		return nil, err
	}
	be.wrapConsumerSender(func(nextSender requestSender) requestSender {
		return &profilesExporterWithObservability{
			obsrep:     be.obsrep,
			nextSender: nextSender,
		}
	})

	lc, err := consumer.NewProfiles(func(ctx context.Context, ld pprofile.Profiles) error {
		req := newProfilesRequest(ctx, ld, pusher)
		serr := be.sender.send(req)
		if errors.Is(serr, errSendingQueueIsFull) {
			be.obsrep.recordProfilesEnqueueFailure(req.Context(), int64(req.Count()))
		}
		return serr
	}, bs.consumerOptions...)

	return &profilesExporter{
		baseExporter: be,
		Profiles:     lc,
	}, err
}

type profilesExporterWithObservability struct {
	obsrep     *obsExporter
	nextSender requestSender
}

func (lewo *profilesExporterWithObservability) send(req internal.Request) error {
	req.SetContext(lewo.obsrep.StartProfilesOp(req.Context()))
	err := lewo.nextSender.send(req)
	lewo.obsrep.EndProfilesOp(req.Context(), req.Count(), err)
	return err
}
