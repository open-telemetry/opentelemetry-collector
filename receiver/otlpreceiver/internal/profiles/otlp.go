// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package profiles // import "go.opentelemetry.io/collector/receiver/otlpreceiver/internal/profiles"

import (
	"context"

	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/obsreport"
	"go.opentelemetry.io/collector/pdata/pprofile/pprofileotlp"
)

const dataFormatProtobuf = "protobuf"

// Receiver is the type used to handle profiles from OpenTelemetry exporters.
type Receiver struct {
	pprofileotlp.UnimplementedGRPCServer
	nextConsumer consumer.Profiles
	obsrecv      *obsreport.Receiver
}

// New creates a new Receiver reference.
func New(nextConsumer consumer.Profiles, obsrecv *obsreport.Receiver) *Receiver {
	return &Receiver{
		nextConsumer: nextConsumer,
		obsrecv:      obsrecv,
	}
}

// Export implements the service Export profiles func.
func (r *Receiver) Export(ctx context.Context, req pprofileotlp.ExportRequest) (pprofileotlp.ExportResponse, error) {
	ld := req.Profiles()
	numProfiles := ld.ProfileCount()
	if numProfiles == 0 {
		return pprofileotlp.NewExportResponse(), nil
	}

	ctx = r.obsrecv.StartProfilesOp(ctx)
	err := r.nextConsumer.ConsumeProfiles(ctx, ld)
	r.obsrecv.EndProfilesOp(ctx, dataFormatProtobuf, numProfiles, err)

	return pprofileotlp.NewExportResponse(), err
}
