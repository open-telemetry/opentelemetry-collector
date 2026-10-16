// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package profiles // import "go.opentelemetry.io/collector/receiver/otlpreceiver/internal/profiles"

import (
	"context"
	stderrors "errors"

	"go.opentelemetry.io/collector/consumer/consumererror"
	"go.opentelemetry.io/collector/consumer/xconsumer"
	"go.opentelemetry.io/collector/pdata/pprofile/pprofileotlp"
	"go.opentelemetry.io/collector/receiver/otlpreceiver/internal/errors"
	"go.opentelemetry.io/collector/receiver/otlpreceiver/internal/metadata"
	"go.opentelemetry.io/collector/receiver/receiverhelper"
)

const (
	dataFormatProtobuf = "protobuf"
	invalidUTF8Message = "request contains invalid UTF-8"
)

// errInvalidUTF8 is reported to obsreport so that data rejected for invalid UTF-8
// is accounted for in the receiver metrics and spans.
var errInvalidUTF8 = consumererror.NewPermanent(stderrors.New(invalidUTF8Message))

// Receiver is the type used to handle spans from OpenTelemetry exporters.
type Receiver struct {
	pprofileotlp.UnimplementedGRPCServer
	nextConsumer xconsumer.Profiles
	obsreport    *receiverhelper.ObsReport
}

// New creates a new Receiver reference.
func New(nextConsumer xconsumer.Profiles, obsreport *receiverhelper.ObsReport) *Receiver {
	return &Receiver{
		nextConsumer: nextConsumer,
		obsreport:    obsreport,
	}
}

// Export implements the service Export profiles func.
func (r *Receiver) Export(ctx context.Context, req pprofileotlp.ExportRequest) (pprofileotlp.ExportResponse, error) {
	td := req.Profiles()
	rejectedProfiles := 0
	rejectedSamples := 0
	if metadata.ReceiverOtlpRejectInvalidUTF8FeatureGate.IsEnabled() {
		// The partial-success response counts profiles, but the receiver telemetry
		// counts samples, so track both.
		samplesBefore := td.SampleCount()
		rejectedProfiles = req.RejectInvalidUTF8()
		rejectedSamples = samplesBefore - td.SampleCount()
	}
	resp := pprofileotlp.NewExportResponse()
	if rejectedProfiles > 0 {
		resp.PartialSuccess().SetRejectedProfiles(int64(rejectedProfiles))
		resp.PartialSuccess().SetErrorMessage(invalidUTF8Message)
		r.recordRejected(ctx, rejectedSamples)
	}
	// We need to ensure that it propagates the receiver name as a tag
	numSamples := td.SampleCount()
	if numSamples == 0 {
		return resp, nil
	}

	ctx = r.obsreport.StartProfilesOp(ctx)
	err := r.nextConsumer.ConsumeProfiles(ctx, td)
	r.obsreport.EndProfilesOp(ctx, dataFormatProtobuf, numSamples, err)

	// Use appropriate status codes for permanent/non-permanent errors
	// If we return the error straightaway, then the grpc implementation will set status code to Unknown
	// Refer: https://github.com/grpc/grpc-go/blob/v1.59.0/server.go#L1345
	// So, convert the error to appropriate grpc status and return the error
	// NonPermanent errors will be converted to codes.Unavailable (equivalent to HTTP 503)
	// Permanent errors will be converted to codes.InvalidArgument (equivalent to HTTP 400)
	if err != nil {
		return pprofileotlp.NewExportResponse(), errors.GetStatusFromError(err)
	}

	return resp, nil
}

// recordRejected accounts for samples rejected by the receiver itself so they show
// up in the receiver telemetry (e.g. otelcol_receiver_refused_profile_samples) and
// not only in the partial-success response.
func (r *Receiver) recordRejected(ctx context.Context, rejected int) {
	ctx = r.obsreport.StartProfilesOp(ctx)
	r.obsreport.EndProfilesOp(ctx, dataFormatProtobuf, rejected, errInvalidUTF8)
}
