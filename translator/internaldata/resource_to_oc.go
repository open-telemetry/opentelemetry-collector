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
	"fmt"
	"strconv"
	"strings"
	"time"

	occommon "github.com/census-instrumentation/opencensus-proto/gen-go/agent/common/v1"
	ocresource "github.com/census-instrumentation/opencensus-proto/gen-go/resource/v1"
	"go.opencensus.io/resource/resourcekeys"
	"google.golang.org/protobuf/types/known/timestamppb"

	"go.opentelemetry.io/collector/consumer/pdata"
	"go.opentelemetry.io/collector/translator/conventions"
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

func internalResourceToOC(resource pdata.Resource) (*occommon.Node, *ocresource.Resource) {
	if resource.IsNil() {
		return nil, nil
	}
	attrs := resource.Attributes()

	ocNode := occommon.Node{}
	ocResource := ocresource.Resource{}

	if attrs.Len() == 0 {
		return &ocNode, &ocResource
	}

	labels := make(map[string]string, attrs.Len())
	attrs.ForEach(func(k string, v pdata.AttributeValue) {
		val := attributeValueToString(v, false)

		switch k {
		case conventions.OCAttributeResourceType:
			ocResource.Type = val
		case conventions.AttributeServiceName:
			if ocNode.ServiceInfo == nil {
				ocNode.ServiceInfo = &occommon.ServiceInfo{}
			}
			ocNode.ServiceInfo.Name = val
		case conventions.OCAttributeProcessStartTime:
			t, err := time.Parse(time.RFC3339Nano, val)
			if err != nil {
				return
			}
			ts := timestamppb.New(t)
			if ocNode.Identifier == nil {
				ocNode.Identifier = &occommon.ProcessIdentifier{}
			}
			ocNode.Identifier.StartTimestamp = ts
		case conventions.AttributeHostHostname:
			if ocNode.Identifier == nil {
				ocNode.Identifier = &occommon.ProcessIdentifier{}
			}
			ocNode.Identifier.HostName = val
		case conventions.OCAttributeProcessID:
			pid, err := strconv.Atoi(val)
			if err != nil {
				pid = defaultProcessID
			}
			if ocNode.Identifier == nil {
				ocNode.Identifier = &occommon.ProcessIdentifier{}
			}
			ocNode.Identifier.Pid = uint32(pid)
		case conventions.AttributeTelemetrySDKVersion:
			if ocNode.LibraryInfo == nil {
				ocNode.LibraryInfo = &occommon.LibraryInfo{}
			}
			ocNode.LibraryInfo.CoreLibraryVersion = val
		case conventions.OCAttributeExporterVersion:
			if ocNode.LibraryInfo == nil {
				ocNode.LibraryInfo = &occommon.LibraryInfo{}
			}
			ocNode.LibraryInfo.ExporterVersion = val
		case conventions.AttributeTelemetrySDKLanguage:
			if code, ok := occommon.LibraryInfo_Language_value[val]; ok {
				if ocNode.LibraryInfo == nil {
					ocNode.LibraryInfo = &occommon.LibraryInfo{}
				}
				ocNode.LibraryInfo.Language = occommon.LibraryInfo_Language(code)
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

	return &ocNode, &ocResource
}

func attributeValueToString(attr pdata.AttributeValue, jsonLike bool) string {
	switch attr.Type() {
	case pdata.AttributeValueNULL:
		if jsonLike {
			return "null"
		}
		return ""
	case pdata.AttributeValueSTRING:
		if jsonLike {
			return fmt.Sprintf("%q", attr.StringVal())
		}
		return attr.StringVal()

	case pdata.AttributeValueBOOL:
		return strconv.FormatBool(attr.BoolVal())

	case pdata.AttributeValueDOUBLE:
		return strconv.FormatFloat(attr.DoubleVal(), 'f', -1, 64)

	case pdata.AttributeValueINT:
		return strconv.FormatInt(attr.IntVal(), 10)

	case pdata.AttributeValueMAP:
		// OpenCensus attributes cannot represent maps natively. Convert the
		// map to a JSON-like string.
		var sb strings.Builder
		sb.WriteString("{")
		m := attr.MapVal()
		first := true
		m.ForEach(func(k string, v pdata.AttributeValue) {
			if !first {
				sb.WriteString(",")
			}
			first = false
			sb.WriteString(fmt.Sprintf("%q:%s", k, attributeValueToString(v, true)))
		})
		sb.WriteString("}")
		return sb.String()

	case pdata.AttributeValueARRAY:
		// OpenCensus attributes cannot represent arrays natively. Convert the
		// array to a JSON-like string.
		var sb strings.Builder
		sb.WriteString("[")
		m := attr.ArrayVal()
		first := true
		len := m.Len()
		for i := 0; i < len; i++ {
			v := m.At(i)
			if !first {
				sb.WriteString(",")
			}
			first = false
			sb.WriteString(attributeValueToString(v, true))
		}
		sb.WriteString("]")
		return sb.String()

	default:
		return fmt.Sprintf("<Unknown OpenTelemetry attribute value type %q>", attr.Type())
	}
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
