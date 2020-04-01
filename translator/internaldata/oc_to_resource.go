// Copyright 2020 OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package internaldata

import (
	occommon "github.com/census-instrumentation/opencensus-proto/gen-go/agent/common/v1"
	ocresource "github.com/census-instrumentation/opencensus-proto/gen-go/resource/v1"
	"github.com/golang/protobuf/ptypes"

	"github.com/open-telemetry/opentelemetry-collector/internal/data"
	"github.com/open-telemetry/opentelemetry-collector/translator/conventions"
)

func ocNodeResourceToInternal(ocNode *occommon.Node, ocResource *ocresource.Resource, dest data.Resource) {
	// Number of special fields in the Node. See the code below that deals with special fields.
	const specialNodeAttrCount = 7

	// Number of special fields in the Resource.
	const specialResourceAttrCount = 1

	// Calculate maximum total number of attributes. It is OK if we are a bit higher than
	// the exact number since this is only needed for capacity reservation.
	maxTotalAttrCount := 0
	if ocNode != nil {
		maxTotalAttrCount += len(ocNode.Attributes) + specialNodeAttrCount
	}
	if ocResource != nil {
		maxTotalAttrCount += len(ocResource.Labels) + specialResourceAttrCount
	}

	// Create a map where we will place all attributes from the Node and Resource.
	attrs := make(map[string]data.AttributeValue, maxTotalAttrCount)

	if ocNode != nil {
		// Copy all Attributes.
		for k, v := range ocNode.Attributes {
			attrs[k] = data.NewAttributeValueString(v)
		}

		// Add all special fields.
		if ocNode.ServiceInfo != nil {
			if ocNode.ServiceInfo.Name != "" {
				attrs[conventions.AttributeServiceName] = data.NewAttributeValueString(
					ocNode.ServiceInfo.Name)
			}
		}
		if ocNode.Identifier != nil {
			if ocNode.Identifier.StartTimestamp != nil {
				attrs[conventions.OCAttributeProcessStartTime] = data.NewAttributeValueString(
					ptypes.TimestampString(ocNode.Identifier.StartTimestamp))
			}
			if ocNode.Identifier.HostName != "" {
				attrs[conventions.AttributeHostHostname] = data.NewAttributeValueString(
					ocNode.Identifier.HostName)
			}
			if ocNode.Identifier.Pid != 0 {
				attrs[conventions.OCAttributeProcessID] = data.NewAttributeValueInt(int64(ocNode.Identifier.Pid))
			}
		}
		if ocNode.LibraryInfo != nil {
			if ocNode.LibraryInfo.CoreLibraryVersion != "" {
				attrs[conventions.AttributeLibraryVersion] = data.NewAttributeValueString(
					ocNode.LibraryInfo.CoreLibraryVersion)
			}
			if ocNode.LibraryInfo.ExporterVersion != "" {
				attrs[conventions.OCAttributeExporterVersion] = data.NewAttributeValueString(
					ocNode.LibraryInfo.ExporterVersion)
			}
			if ocNode.LibraryInfo.Language != occommon.LibraryInfo_LANGUAGE_UNSPECIFIED {
				attrs[conventions.AttributeLibraryLanguage] = data.NewAttributeValueString(
					ocNode.LibraryInfo.Language.String())
			}
		}
	}

	if ocResource != nil {
		// Copy resource Labels.
		for k, v := range ocResource.Labels {
			attrs[k] = data.NewAttributeValueString(v)
		}
		// Add special fields.
		if ocResource.Type != "" {
			attrs[conventions.OCAttributeResourceType] = data.NewAttributeValueString(ocResource.Type)
		}
	}

	if len(attrs) != 0 {
		dest.InitEmpty()
		// TODO: Re-evaluate if we want to construct a map first, or we can construct directly
		// a slice of AttributeKeyValue.
		dest.Attributes().InitFromMap(attrs)
	}
}
