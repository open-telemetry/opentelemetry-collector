// Copyright 2019 OpenTelemetry Authors
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
	"time"

	occommon "github.com/census-instrumentation/opencensus-proto/gen-go/agent/common/v1"
	ocresource "github.com/census-instrumentation/opencensus-proto/gen-go/resource/v1"
	"github.com/golang/protobuf/ptypes"

	"github.com/open-telemetry/opentelemetry-collector/internal/data"
	"github.com/open-telemetry/opentelemetry-collector/translator/conventions"
)

func internalResourceToOC(resource *data.Resource) (*occommon.Node, *ocresource.Resource) {
	attrs := resource.Attributes()

	ocNode := occommon.Node{}
	ocResource := ocresource.Resource{}

	if len(attrs) == 0 {
		return &ocNode, &ocResource
	}

	labels := make(map[string]string, len(attrs))
	for key, attributeValue := range attrs {
		val := attributeValueToString(attributeValue)

		switch key {
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
				continue
			}
			ts, err := ptypes.TimestampProto(t)
			if err != nil {
				continue
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
		case conventions.AttributeLibraryVersion:
			if ocNode.LibraryInfo == nil {
				ocNode.LibraryInfo = &occommon.LibraryInfo{}
			}
			ocNode.LibraryInfo.CoreLibraryVersion = val
		case conventions.OCAttributeExporterVersion:
			if ocNode.LibraryInfo == nil {
				ocNode.LibraryInfo = &occommon.LibraryInfo{}
			}
			ocNode.LibraryInfo.ExporterVersion = val
		case conventions.AttributeLibraryLanguage:
			if code, ok := occommon.LibraryInfo_Language_value[val]; ok {
				if ocNode.LibraryInfo == nil {
					ocNode.LibraryInfo = &occommon.LibraryInfo{}
				}
				ocNode.LibraryInfo.Language = occommon.LibraryInfo_Language(code)
			}
		default:
			// Not a special attribute, put it into resource labels
			labels[key] = val
		}
	}
	ocResource.Labels = labels

	return &ocNode, &ocResource
}

func attributeValueToString(attr data.AttributeValue) string {
	switch attr.Type() {
	case data.AttributeValueSTRING:
		return attr.StringVal()
	case data.AttributeValueBOOL:
		return strconv.FormatBool(attr.BoolVal())
	case data.AttributeValueDOUBLE:
		return strconv.FormatFloat(attr.DoubleVal(), 'f', -1, 64)
	case data.AttributeValueINT:
		return strconv.FormatInt(attr.IntVal(), 10)
	default:
		return fmt.Sprintf("<Unknown OpenTelemetry attribute value type %q>", attr.Type())
	}
}
