// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package persistentqueue // import "go.opentelemetry.io/collector/exporter/exporterhelper/internal/queuebatch/internal/persistentqueue"

import (
	"fmt"

	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/request"
)

// Numeric constants for protobuf serialization
const (
	SizerTypeBytes    int32 = iota // 0
	SizerTypeItems                 // 1
	SizerTypeRequests              // 2
)

// SizerTypeToProto converts SizerType to int32 for protobuf serialization
func SizerTypeToProto(sizerType request.SizerType) (int32, error) {
	switch sizerType {
	case request.SizerTypeBytes:
		return SizerTypeBytes, nil
	case request.SizerTypeItems:
		return SizerTypeItems, nil
	case request.SizerTypeRequests:
		return SizerTypeRequests, nil
	default:
		return -1, fmt.Errorf("invalid sizer type: %v", sizerType)
	}
}

// SizerTypeFromProto creates SizerType from int32 representation
func SizerTypeFromProto(value int32) (request.SizerType, error) {
	switch value {
	case SizerTypeBytes:
		return request.SizerTypeBytes, nil
	case SizerTypeItems:
		return request.SizerTypeItems, nil
	case SizerTypeRequests:
		return request.SizerTypeRequests, nil
	default:
		return request.SizerType{}, fmt.Errorf("invalid sizer type value: %d", value)
	}
}
