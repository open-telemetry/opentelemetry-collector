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
}

func origAccessor(packageName string) string {
	if usedByOtherDataTypes(packageName) {
		return "getOrig()"
	}
	return "orig"
}

func stateAccessor(packageName string) string {
	if usedByOtherDataTypes(packageName) {
		return "getState()"
	}
	return "state"
}
