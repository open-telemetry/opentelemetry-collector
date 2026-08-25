// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package internal

import (
	"testing"

	"github.com/stretchr/testify/require"
	gootlpcommon "go.opentelemetry.io/proto/slim/otlp/common/v1"
	goproto "google.golang.org/protobuf/proto"
)

func TestAnyValueBytes(t *testing.T) {
	av := &gootlpcommon.AnyValue{Value: &gootlpcommon.AnyValue_BytesValue{BytesValue: nil}}
	buf, err := goproto.Marshal(av)
	require.NoError(t, err)

	pav := &AnyValue{Value: &AnyValue_BytesValue{BytesValue: nil}}
	pbuf := make([]byte, pav.SizeProto())
	n := pav.MarshalProto(pbuf)
	pbuf = pbuf[:n]
	require.Equal(t, buf, pbuf)
}
