// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package profiles // import "go.opentelemetry.io/collector/receiver/otlpreceiver/internal/profiles"

import (
	"context"

	"go.opentelemetry.io/collector/consumer/xconsumer"
	"go.opentelemetry.io/collector/extension/extensionlimiter"
	"go.opentelemetry.io/collector/pdata/pprofile"
	"go.opentelemetry.io/collector/pdata/pprofile/pprofileotlp"
	"go.opentelemetry.io/collector/receiver/otlpreceiver/internal/errors"
)

// Receiver is the type used to handle spans from OpenTelemetry exporters.
type Receiver struct {
	pprofileotlp.UnimplementedGRPCServer
	nextConsumer xconsumer.Profiles
	itemsLimiter extensionlimiter.ResourceLimiter
	sizeLimiter  extensionlimiter.ResourceLimiter
}

// New creates a new Receiver reference.
func New(nextConsumer xconsumer.Profiles, limiter extensionlimiter.Provider) *Receiver {
	var itemsLimiter, sizeLimiter extensionlimiter.ResourceLimiter
	if limiter != nil {
		itemsLimiter = limiter.ResourceLimiter(extensionlimiter.WeightKeyRequestItems)
		sizeLimiter = limiter.ResourceLimiter(extensionlimiter.WeightKeyResidentSize)
	}
	return &Receiver{
		nextConsumer: nextConsumer,
		itemsLimiter: itemsLimiter,
		sizeLimiter:  sizeLimiter,
	}
}

// Export implements the service Export profiles func.
func (r *Receiver) Export(ctx context.Context, req pprofileotlp.ExportRequest) (pprofileotlp.ExportResponse, error) {
	td := req.Profiles()
	// We need to ensure that it propagates the receiver name as a tag
	numProfiles := td.SampleCount()
	if numProfiles == 0 {
		return pprofileotlp.NewExportResponse(), nil
	}

	// Apply the items limiter if available
	if r.itemsLimiter != nil {
		release, err := r.itemsLimiter.Acquire(ctx, uint64(numProfiles))
		if err != nil {
			return pprofileotlp.NewExportResponse(), errors.GetStatusFromError(err)
		}
		defer release()
	}

	// Apply the memory size limiter if available
	if r.sizeLimiter != nil {
		// Get the marshaled size of the request as a proxy for memory size
		var sizer pprofile.ProtoMarshaler
		size := sizer.ProfilesSize(td)
		release, err := r.sizeLimiter.Acquire(ctx, uint64(size))
		if err != nil {
			return pprofileotlp.NewExportResponse(), errors.GetStatusFromError(err)
		}
		defer release()
	}

	err := r.nextConsumer.ConsumeProfiles(ctx, td)
	// Use appropriate status codes for permanent/non-permanent errors
	// If we return the error straightaway, then the grpc implementation will set status code to Unknown
	// Refer: https://github.com/grpc/grpc-go/blob/v1.59.0/server.go#L1345
	// So, convert the error to appropriate grpc status and return the error
	// NonPermanent errors will be converted to codes.Unavailable (equivalent to HTTP 503)
	// Permanent errors will be converted to codes.InvalidArgument (equivalent to HTTP 400)
	if err != nil {
		return pprofileotlp.NewExportResponse(), errors.GetStatusFromError(err)
	}

	return pprofileotlp.NewExportResponse(), nil
}
