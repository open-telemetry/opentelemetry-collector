// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package internal // import "go.opentelemetry.io/collector/internal/cmd/pdatagen/internal"

type Field interface {
	GenerateAccessors(ms *messageStruct) string

	GenerateAccessorsTest(ms *messageStruct) string

	GenerateSetWithTestValue(ms *messageStruct) string

	GenerateCopyOrig(ms *messageStruct) string

	GenerateMarshalJSON(ms *messageStruct) string

	GenerateUnmarshalJSON(ms *messageStruct) string

	GenerateSizeProto(ms *messageStruct) string

	GenerateMarshalProto(ms *messageStruct) string

	GenerateTestValue(ms *messageStruct) string
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
