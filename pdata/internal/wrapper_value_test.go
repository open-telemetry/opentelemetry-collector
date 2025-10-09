// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package internal

import (
	"testing"

	"github.com/stretchr/testify/require"
	gootlpcommon "go.opentelemetry.io/proto/slim/otlp/common/v1"
	goproto "google.golang.org/protobuf/proto"

	otlpcommon "go.opentelemetry.io/collector/pdata/internal/data/protogen/common/v1"
)

func TestAnyValueBytes(t *testing.T) {
	av := &gootlpcommon.AnyValue{Value: &gootlpcommon.AnyValue_BytesValue{BytesValue: nil}}
	buf, err := goproto.Marshal(av)
	require.NoError(t, err)

	pav := &otlpcommon.AnyValue{Value: &otlpcommon.AnyValue_BytesValue{BytesValue: nil}}
	pbuf := make([]byte, SizeProtoOrigAnyValue(pav))
	n := MarshalProtoOrigAnyValue(pav, pbuf)
	pbuf = pbuf[:n]
	require.Equal(t, buf, pbuf)
}
