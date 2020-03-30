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

package common

import (
	"strconv"

	occommon "github.com/census-instrumentation/opencensus-proto/gen-go/agent/common/v1"
	ocresource "github.com/census-instrumentation/opencensus-proto/gen-go/resource/v1"
	"github.com/golang/protobuf/ptypes"
	otlpcommon "github.com/open-telemetry/opentelemetry-proto/gen/go/common/v1"
	otlpresource "github.com/open-telemetry/opentelemetry-proto/gen/go/resource/v1"

	"github.com/open-telemetry/opentelemetry-collector/translator/conventions"
)

func OCNodeResourceToOtlp(node *occommon.Node, resource *ocresource.Resource) *otlpresource.Resource {
	otlpResource := &otlpresource.Resource{}

	// Move all special fields in Node and in Resource to a temporary attributes map.
	// After that we will build the attributes slice in OTLP format from this map.
	// This ensures there are no attributes with duplicate keys.

	// Number of special fields in the Node. See the code below that deals with special fields.
	const specialNodeAttrCount = 7

	// Number of special fields in the Resource.
	const specialResourceAttrCount = 1

	// Calculate maximum total number of attributes. It is OK if we are a bit higher than
	// the exact number since this is only needed for capacity reservation.
	totalAttrCount := 0
	if node != nil {
		totalAttrCount += len(node.Attributes) + specialNodeAttrCount
	}
	if resource != nil {
		totalAttrCount += len(resource.Labels) + specialResourceAttrCount
	}

	// Create a temporary map where we will place all attributes from the Node and Resource.
	attrs := make(map[string]string, totalAttrCount)

	if node != nil {
		// Copy all Attributes.
		for k, v := range node.Attributes {
			attrs[k] = v
		}

		// Add all special fields.
		if node.ServiceInfo != nil {
			if node.ServiceInfo.Name != "" {
				attrs[conventions.AttributeServiceName] = node.ServiceInfo.Name
			}
		}
		if node.Identifier != nil {
			if node.Identifier.StartTimestamp != nil {
				attrs[conventions.OCAttributeProcessStartTime] = ptypes.TimestampString(node.Identifier.StartTimestamp)
			}
			if node.Identifier.HostName != "" {
				attrs[conventions.AttributeHostHostname] = node.Identifier.HostName
			}
			if node.Identifier.Pid != 0 {
				attrs[conventions.OCAttributeProcessID] = strconv.Itoa(int(node.Identifier.Pid))
			}
		}
		if node.LibraryInfo != nil {
			if node.LibraryInfo.CoreLibraryVersion != "" {
				attrs[conventions.AttributeLibraryVersion] = node.LibraryInfo.CoreLibraryVersion
			}
			if node.LibraryInfo.ExporterVersion != "" {
				attrs[conventions.OCAttributeExporterVersion] = node.LibraryInfo.ExporterVersion
			}
			if node.LibraryInfo.Language != occommon.LibraryInfo_LANGUAGE_UNSPECIFIED {
				attrs[conventions.AttributeLibraryLanguage] = node.LibraryInfo.Language.String()
			}
		}
	}

	if resource != nil {
		// Copy resource Labels.
		for k, v := range resource.Labels {
			attrs[k] = v
		}
		// Add special fields.
		if resource.Type != "" {
			attrs[conventions.OCAttributeResourceType] = resource.Type
		}
	}

	// Convert everything from the temporary matp to OTLP format.
	otlpResource.Attributes = attrMapToOtlp(attrs)

	return otlpResource
}

func attrMapToOtlp(ocAttrs map[string]string) []*otlpcommon.AttributeKeyValue {
	if len(ocAttrs) == 0 {
		return nil
	}

	otlpAttrs := make([]*otlpcommon.AttributeKeyValue, len(ocAttrs))
	i := 0
	for k, v := range ocAttrs {
		otlpAttrs[i] = otlpStringAttr(k, v)
		i++
	}
	return otlpAttrs
}

func otlpStringAttr(key string, val string) *otlpcommon.AttributeKeyValue {
	return &otlpcommon.AttributeKeyValue{
		Key:         key,
		Type:        otlpcommon.AttributeKeyValue_STRING,
		StringValue: val,
	}
}
