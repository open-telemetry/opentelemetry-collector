// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package proto // import "go.opentelemetry.io/collector/internal/cmd/pdatagen/internal/proto"

// WireType represents the proto wire type.
type WireType uint32

const (
	WireTypeVarint     WireType = 0
	WireTypeI64        WireType = 1
	WireTypeLen        WireType = 2
	WireTypeStartGroup WireType = 3
	WireTypeEndGroup   WireType = 4
	WireTypeI32        WireType = 5
)
