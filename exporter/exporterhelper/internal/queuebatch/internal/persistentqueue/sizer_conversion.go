// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package persistentqueue // import "go.opentelemetry.io/collector/exporter/exporterhelper/internal/queuebatch/internal/persistentqueue"

import (
	"fmt"

	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/request"
)

// Numeric constants for protobuf serialization
const (
	SizerTypeBytesValue    int32 = iota // 0
	SizerTypeItemsValue                 // 1
	SizerTypeRequestsValue              // 2
)

// SizerTypeToInt32 converts SizerType to int32 for protobuf serialization
func SizerTypeToInt32(sizerType request.SizerType) (int32, error) {
	switch sizerType {
	case request.SizerTypeBytes:
		return SizerTypeBytesValue, nil
	case request.SizerTypeItems:
		return SizerTypeItemsValue, nil
	case request.SizerTypeRequests:
		return SizerTypeRequestsValue, nil
	default:
		return -1, fmt.Errorf("invalid sizer type: %v", sizerType)
	}
}

// SizerTypeFromInt32 creates SizerType from int32 representation
func SizerTypeFromInt32(value int32) (request.SizerType, error) {
	switch value {
	case SizerTypeBytesValue:
		return request.SizerTypeBytes, nil
	case SizerTypeItemsValue:
		return request.SizerTypeItems, nil
	case SizerTypeRequestsValue:
		return request.SizerTypeRequests, nil
	default:
		return request.SizerType{}, fmt.Errorf("invalid sizer type value: %d", value)
	}
}
