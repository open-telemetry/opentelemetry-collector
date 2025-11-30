// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package request // import "go.opentelemetry.io/collector/pdata/xpdata/request"

import (
	"context"
	"fmt"

	"go.opentelemetry.io/collector/pdata/internal"
	"go.opentelemetry.io/collector/pdata/pprofile"
)

// MarshalProfiles marshals pprofile.Profiles along with the context into a byte slice.
func MarshalProfiles(ctx context.Context, ld pprofile.Profiles) ([]byte, error) {
	pr := internal.ProfilesRequest{
		FormatVersion:  requestFormatVersion,
		ProfilesData:   internal.ProfilesToProto(internal.ProfilesWrapper(ld)),
		RequestContext: encodeContext(ctx),
	}
	buf := make([]byte, pr.SizeProto())
	pr.MarshalProto(buf)
	return buf, nil
}

// UnmarshalProfiles unmarshals a byte slice into pprofile.Profiles and the context.
func UnmarshalProfiles(buf []byte) (context.Context, pprofile.Profiles, error) {
	ctx := context.Background()
	if !isRequestPayloadV1(buf) {
		return ctx, pprofile.Profiles{}, ErrInvalidFormat
	}
	pr := internal.ProfilesRequest{}
	if err := pr.UnmarshalProto(buf); err != nil {
		return ctx, pprofile.Profiles{}, fmt.Errorf("failed to unmarshal profiles request: %w", err)
	}
	return decodeContext(ctx, pr.RequestContext), pprofile.Profiles(internal.ProfilesFromProto(pr.ProfilesData)), nil
}
