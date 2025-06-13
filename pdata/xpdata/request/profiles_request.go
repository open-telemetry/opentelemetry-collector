// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package request // import "go.opentelemetry.io/collector/pdata/xpdata/request"

import (
	"context"
	"fmt"

	"google.golang.org/protobuf/proto"

	"go.opentelemetry.io/collector/pdata/internal"
	"go.opentelemetry.io/collector/pdata/pprofile"
	reqint "go.opentelemetry.io/collector/pdata/xpdata/request/internal"
)

// MarshalProfiles marshals pprofile.Profiles along with the context into a byte slice.
func MarshalProfiles(ctx context.Context, ld pprofile.Profiles) ([]byte, error) {
	otlpProfiles := internal.ProfilesToProto(internal.Profiles(ld))
	rc := encodeContext(ctx)
	pr := reqint.ProfilesRequest{
		FormatVersion:  requestFormatVersion,
		ProfilesData:   &otlpProfiles,
		RequestContext: &rc,
	}
	return proto.Marshal(&pr)
}

// UnmarshalProfiles unmarshals a byte slice into pprofile.Profiles and the context.
func UnmarshalProfiles(buf []byte) (context.Context, pprofile.Profiles, error) {
	if !isRequestPayloadV1(buf) {
		return context.Background(), pprofile.Profiles{}, ErrInvalidFormat
	}
	pr := reqint.ProfilesRequest{}
	if err := proto.Unmarshal(buf, &pr); err != nil {
		return context.Background(), pprofile.Profiles{}, fmt.Errorf("failed to unmarshal profiles request: %w", err)
	}
	return decodeContext(pr.RequestContext), pprofile.Profiles(internal.ProfilesFromProto(*pr.ProfilesData)), nil
}
