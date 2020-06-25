// Copyright The OpenTelemetry Authors
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
	"fmt"
	"strconv"
	"strings"
	"time"

	occommon "github.com/census-instrumentation/opencensus-proto/gen-go/agent/common/v1"
	ocresource "github.com/census-instrumentation/opencensus-proto/gen-go/resource/v1"
	"github.com/golang/protobuf/ptypes"

	"go.opentelemetry.io/collector/consumer/pdata"
	"go.opentelemetry.io/collector/translator/conventions"
)

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
			ts, err := ptypes.TimestampProto(t)
			if err != nil {
				return
			}
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

	default:
		return fmt.Sprintf("<Unknown OpenTelemetry attribute value type %q>", attr.Type())
	}

	// TODO: Add support for ARRAY type.
}
