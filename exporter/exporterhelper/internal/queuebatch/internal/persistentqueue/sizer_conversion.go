// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package persistentqueue // import "go.opentelemetry.io/collector/exporter/exporterhelper/internal/queuebatch/internal/persistentqueue"

import (
	"fmt"

	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/request"
)

// SizerTypeToProto converts SizerType to int32 for protobuf serialization
func SizerTypeToProto(sizerType request.SizerType) (SizerType, error) {
	switch sizerType {
	case request.SizerTypeRequests:
		return SizerType_REQUESTS, nil
	case request.SizerTypeItems:
		return SizerType_ITEMS, nil
	case request.SizerTypeBytes:
		return SizerType_BYTES, nil
	default:
		return -1, fmt.Errorf("invalid sizer type: %v", sizerType)
	}
}

// SizerTypeFromProto creates SizerType from int32 representation
func SizerTypeFromProto(value SizerType) (request.SizerType, error) {
	switch value {
	case SizerType_REQUESTS:
		return request.SizerTypeRequests, nil
	case SizerType_ITEMS:
		return request.SizerTypeItems, nil
	case SizerType_BYTES:
		return request.SizerTypeBytes, nil
	default:
		return request.SizerType{}, fmt.Errorf("invalid sizer type value: %d", value)
	}
}
