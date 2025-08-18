// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package pdata // import "go.opentelemetry.io/collector/internal/cmd/pdatagen/internal/pdata"

import (
	"go.opentelemetry.io/collector/internal/cmd/pdatagen/internal/proto"
)

type Field interface {
	GenerateAccessors(ms *messageStruct) string

	GenerateAccessorsTest(ms *messageStruct) string

	GenerateSetWithTestValue(ms *messageStruct) string

	GenerateTestValue(ms *messageStruct) string

	GenerateCopyOrig(ms *messageStruct) string

	GenerateUnmarshalJSON(ms *messageStruct) string

	toProtoField(ms *messageStruct) *proto.Field
}

func origAccessor(hasWrapper bool) string {
	if hasWrapper {
		return "getOrig()"
	}
	return "orig"
}

func stateAccessor(hasWrapper bool) string {
	if hasWrapper {
		return "getState()"
	}
	return "state"
}
