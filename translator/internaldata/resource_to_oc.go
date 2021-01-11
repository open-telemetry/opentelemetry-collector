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
	"strconv"
	"time"

	occommon "github.com/census-instrumentation/opencensus-proto/gen-go/agent/common/v1"
	ocresource "github.com/census-instrumentation/opencensus-proto/gen-go/resource/v1"
	"go.opencensus.io/resource/resourcekeys"
	"google.golang.org/protobuf/types/known/timestamppb"

	"go.opentelemetry.io/collector/consumer/pdata"
	"go.opentelemetry.io/collector/translator/conventions"
	tracetranslator "go.opentelemetry.io/collector/translator/trace"
)

type ocInferredResourceType struct {
	// label presence to check against
	labelKeyPresent string
	// inferred resource type
	resourceType string
}

// mapping of label presence to inferred OC resource type
// NOTE: defined in the priority order (first match wins)
var labelPresenceToResourceType = []ocInferredResourceType{
	{
		// See https://github.com/open-telemetry/opentelemetry-specification/blob/master/specification/resource/semantic_conventions/container.md
		labelKeyPresent: conventions.AttributeContainerName,
		resourceType:    resourcekeys.ContainerType,
	},
	{
		// See https://github.com/open-telemetry/opentelemetry-specification/blob/master/specification/resource/semantic_conventions/k8s.md#pod
		labelKeyPresent: conventions.AttributeK8sPod,
		// NOTE: OpenCensus is using "k8s" rather than "k8s.pod" for Pod
		resourceType: resourcekeys.K8SType,
	},
	{
		// See https://github.com/open-telemetry/opentelemetry-specification/blob/master/specification/resource/semantic_conventions/host.md
		labelKeyPresent: conventions.AttributeHostName,
		resourceType:    resourcekeys.HostType,
	},
	{
		// See https://github.com/open-telemetry/opentelemetry-specification/blob/master/specification/resource/semantic_conventions/cloud.md
		labelKeyPresent: conventions.AttributeCloudProvider,
		resourceType:    resourcekeys.CloudType,
	},
}

var langToOCLangCodeMap = getSDKLangToOCLangCodeMap()

func getSDKLangToOCLangCodeMap() map[string]int32 {
	mappings := make(map[string]int32)
	mappings[conventions.AttributeSDKLangValueCPP] = 1
	mappings[conventions.AttributeSDKLangValueDotNET] = 2
	mappings[conventions.AttributeSDKLangValueErlang] = 3
	mappings[conventions.AttributeSDKLangValueGo] = 4
	mappings[conventions.AttributeSDKLangValueJava] = 5
	mappings[conventions.AttributeSDKLangValueNodeJS] = 6
	mappings[conventions.AttributeSDKLangValuePHP] = 7
	mappings[conventions.AttributeSDKLangValuePython] = 8
	mappings[conventions.AttributeSDKLangValueRuby] = 9
	mappings[conventions.AttributeSDKLangValueWebJS] = 10
	return mappings
}

func internalResourceToOC(resource pdata.Resource) (*occommon.Node, *ocresource.Resource) {
	attrs := resource.Attributes()
	if attrs.Len() == 0 {
		return nil, nil
	}

	ocNode := &occommon.Node{}
	ocResource := &ocresource.Resource{}
	labels := make(map[string]string, attrs.Len())
	attrs.ForEach(func(k string, v pdata.AttributeValue) {
		val := tracetranslator.AttributeValueToString(v, false)

		switch k {
		case conventions.OCAttributeResourceType:
			ocResource.Type = val
		case conventions.AttributeServiceName:
			getServiceInfo(ocNode).Name = val
		case conventions.OCAttributeProcessStartTime:
			t, err := time.Parse(time.RFC3339Nano, val)
			if err != nil {
				return
			}
			ts := timestamppb.New(t)
			getProcessIdentifier(ocNode).StartTimestamp = ts
		case conventions.AttributeHostName:
			getProcessIdentifier(ocNode).HostName = val
		case conventions.OCAttributeProcessID:
			pid, err := strconv.Atoi(val)
			if err != nil {
				pid = defaultProcessID
			}
			getProcessIdentifier(ocNode).Pid = uint32(pid)
		case conventions.AttributeTelemetrySDKVersion:
			getLibraryInfo(ocNode).CoreLibraryVersion = val
		case conventions.OCAttributeExporterVersion:
			getLibraryInfo(ocNode).ExporterVersion = val
		case conventions.AttributeTelemetrySDKLanguage:
			if code, ok := langToOCLangCodeMap[val]; ok {
				getLibraryInfo(ocNode).Language = occommon.LibraryInfo_Language(code)
			}
		default:
			// Not a special attribute, put it into resource labels
			labels[k] = val
		}
	})
	ocResource.Labels = labels

	// If resource type is missing, try to infer it
	// based on the presence of resource labels (semantic conventions)
	if ocResource.Type == "" {
		if resType, ok := inferResourceType(ocResource.Labels); ok {
			ocResource.Type = resType
		}
	}

	return ocNode, ocResource
}

func getProcessIdentifier(ocNode *occommon.Node) *occommon.ProcessIdentifier {
	if ocNode.Identifier == nil {
		ocNode.Identifier = &occommon.ProcessIdentifier{}
	}
	return ocNode.Identifier
}

func getLibraryInfo(ocNode *occommon.Node) *occommon.LibraryInfo {
	if ocNode.LibraryInfo == nil {
		ocNode.LibraryInfo = &occommon.LibraryInfo{}
	}
	return ocNode.LibraryInfo
}

func getServiceInfo(ocNode *occommon.Node) *occommon.ServiceInfo {
	if ocNode.ServiceInfo == nil {
		ocNode.ServiceInfo = &occommon.ServiceInfo{}
	}
	return ocNode.ServiceInfo
}

func inferResourceType(labels map[string]string) (string, bool) {
	if labels == nil {
		return "", false
	}

	for _, mapping := range labelPresenceToResourceType {
		if _, ok := labels[mapping.labelKeyPresent]; ok {
			return mapping.resourceType, true
		}
	}

	return "", false
}
