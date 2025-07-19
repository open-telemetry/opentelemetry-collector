// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package json // import "go.opentelemetry.io/collector/pdata/internal/json"

import (
	"io"

	"github.com/gogo/protobuf/jsonpb"
	"github.com/gogo/protobuf/proto"
)

var marshaler = &jsonpb.Marshaler{
	// https://github.com/open-telemetry/opentelemetry-specification/pull/2758
	EnumsAsInts: true,
	// https://github.com/open-telemetry/opentelemetry-specification/pull/2829
	OrigName: false,
}

func Marshal(out io.Writer, pb proto.Message) error {
	return marshaler.Marshal(out, pb)
}

var unmarshaler = &jsonpb.Unmarshaler{
	AllowUnknownFields: true,
}

func Unmarshal(in io.Reader, pb proto.Message) error {
	return unmarshaler.Unmarshal(in, pb)
}
