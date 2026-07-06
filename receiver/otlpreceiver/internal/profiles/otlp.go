// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package profiles // import "go.opentelemetry.io/collector/receiver/otlpreceiver/internal/profiles"

import (
	"context"

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
	rejectedProfiles := 0
	if metadata.OtlpReceiverRejectInvalidUTF8FeatureGate.IsEnabled() {
		rejectedProfiles = req.RejectInvalidUTF8()
	}
	resp := pprofileotlp.NewExportResponse()
	if rejectedProfiles > 0 {
		resp.PartialSuccess().SetRejectedProfiles(int64(rejectedProfiles))
		resp.PartialSuccess().SetErrorMessage(invalidUTF8Message)
	}
	td := req.Profiles()
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
