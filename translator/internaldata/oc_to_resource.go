// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//       http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package internaldata

import (
	"time"

	occommon "github.com/census-instrumentation/opencensus-proto/gen-go/agent/common/v1"
	ocresource "github.com/census-instrumentation/opencensus-proto/gen-go/resource/v1"

	"go.opentelemetry.io/collector/consumer/pdata"
	"go.opentelemetry.io/collector/translator/conventions"
)

var ocLangCodeToLangMap = getOCLangCodeToLangMap()

func getOCLangCodeToLangMap() map[occommon.LibraryInfo_Language]string {
	mappings := make(map[occommon.LibraryInfo_Language]string)
	mappings[1] = conventions.AttributeSDKLangValueCPP
	mappings[2] = conventions.AttributeSDKLangValueDotNET
	mappings[3] = conventions.AttributeSDKLangValueErlang
	mappings[4] = conventions.AttributeSDKLangValueGo
	mappings[5] = conventions.AttributeSDKLangValueJava
	mappings[6] = conventions.AttributeSDKLangValueNodeJS
	mappings[7] = conventions.AttributeSDKLangValuePHP
	mappings[8] = conventions.AttributeSDKLangValuePython
	mappings[9] = conventions.AttributeSDKLangValueRuby
	mappings[10] = conventions.AttributeSDKLangValueWebJS
	return mappings
}

func ocNodeResourceToInternal(ocNode *occommon.Node, ocResource *ocresource.Resource, dest pdata.Resource) {
	if ocNode == nil && ocResource == nil {
		return
	}

	// Number of special fields in OC that will be translated to Attributes
	const serviceInfoAttrCount = 1     // Number of Node.ServiceInfo fields.
	const nodeIdentifierAttrCount = 3  // Number of Node.Identifier fields.
	const libraryInfoAttrCount = 3     // Number of Node.LibraryInfo fields.
	const specialResourceAttrCount = 1 // Number of Resource fields.

	// Calculate maximum total number of attributes for capacity reservation.
	maxTotalAttrCount := 0
	if ocNode != nil {
		maxTotalAttrCount += len(ocNode.Attributes)
		if ocNode.ServiceInfo != nil {
			maxTotalAttrCount += serviceInfoAttrCount
		}
		if ocNode.Identifier != nil {
			maxTotalAttrCount += nodeIdentifierAttrCount
		}
		if ocNode.LibraryInfo != nil {
			maxTotalAttrCount += libraryInfoAttrCount
		}
	}
	if ocResource != nil {
		maxTotalAttrCount += len(ocResource.Labels)
		if ocResource.Type != "" {
			maxTotalAttrCount += specialResourceAttrCount
		}
	}

	// There are no attributes to be set.
	if maxTotalAttrCount == 0 {
		return
	}

	attrs := dest.Attributes()
	attrs.InitEmptyWithCapacity(maxTotalAttrCount)

	if ocNode != nil {
		// Copy all Attributes.
		for k, v := range ocNode.Attributes {
			attrs.InsertString(k, v)
		}

		// Add all special fields.
		if ocNode.ServiceInfo != nil {
			if ocNode.ServiceInfo.Name != "" {
				attrs.UpsertString(conventions.AttributeServiceName, ocNode.ServiceInfo.Name)
			}
		}
		if ocNode.Identifier != nil {
			if ocNode.Identifier.StartTimestamp != nil {
				attrs.UpsertString(conventions.OCAttributeProcessStartTime, ocNode.Identifier.StartTimestamp.AsTime().Format(time.RFC3339Nano))
			}
			if ocNode.Identifier.HostName != "" {
				attrs.UpsertString(conventions.AttributeHostName, ocNode.Identifier.HostName)
			}
			if ocNode.Identifier.Pid != 0 {
				attrs.UpsertInt(conventions.OCAttributeProcessID, int64(ocNode.Identifier.Pid))
			}
		}
		if ocNode.LibraryInfo != nil {
			if ocNode.LibraryInfo.CoreLibraryVersion != "" {
				attrs.UpsertString(conventions.AttributeTelemetrySDKVersion, ocNode.LibraryInfo.CoreLibraryVersion)
			}
			if ocNode.LibraryInfo.ExporterVersion != "" {
				attrs.UpsertString(conventions.OCAttributeExporterVersion, ocNode.LibraryInfo.ExporterVersion)
			}
			if ocNode.LibraryInfo.Language != occommon.LibraryInfo_LANGUAGE_UNSPECIFIED {
				if str, ok := ocLangCodeToLangMap[ocNode.LibraryInfo.Language]; ok {
					attrs.UpsertString(conventions.AttributeTelemetrySDKLanguage, str)
				}
			}
		}
	}

	if ocResource != nil {
		// Copy resource Labels.
		for k, v := range ocResource.Labels {
			attrs.InsertString(k, v)
		}
		// Add special fields.
		if ocResource.Type != "" {
			attrs.UpsertString(conventions.OCAttributeResourceType, ocResource.Type)
		}
	}
}
