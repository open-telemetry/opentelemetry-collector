// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package internal

import (
	"encoding/binary"
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/pdata/internal/proto"
)

// For each repeated field in the OTLP proto that should be packed, check that we can unmarshal
// payloads where it is encoded as unpacked.
// The Protobuf spec recommends this for backwards compatibility purposes.
// We do not test profiles payloads since their proto definition is currently in development.

func appendTag(buf []byte, fieldNo byte, wireType proto.WireType) []byte {
	return append(buf, (fieldNo<<3)|byte(wireType))
}

func appendVarint(buf []byte, v uint64) []byte {
	n := proto.Sov(v)
	for range n {
		buf = append(buf, 0)
	}
	_ = proto.EncodeVarint(buf, len(buf), v)
	return buf
}

func TestUnmarshalUnpackedHistogramDataPoint(t *testing.T) {
	var pb []byte
	pb = appendTag(pb, 6, proto.WireTypeI64) // bucket_counts
	pb = binary.LittleEndian.AppendUint64(pb, 42)
	pb = appendTag(pb, 7, proto.WireTypeI64) // explicit_bounds
	pb = binary.LittleEndian.AppendUint64(pb, math.Float64bits(42.0))

	var hdp HistogramDataPoint
	err := hdp.UnmarshalProto(pb)
	require.NoError(t, err)
	assert.Equal(t, HistogramDataPoint{
		BucketCounts:   []uint64{42},
		ExplicitBounds: []float64{42.0},
	}, hdp)
}

func TestUnmarshalUnpackedExponentialHistogramDataPoint_Buckets(t *testing.T) {
	var pb []byte
	pb = appendTag(pb, 2, proto.WireTypeVarint) // bucket_counts
	pb = appendVarint(pb, 42)

	var ehdpb ExponentialHistogramDataPointBuckets
	err := ehdpb.UnmarshalProto(pb)
	require.NoError(t, err)
	assert.Equal(t, ExponentialHistogramDataPointBuckets{
		BucketCounts: []uint64{42},
	}, ehdpb)
}
