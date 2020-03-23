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
	"fmt"
	"strconv"
	"time"

	occommon "github.com/census-instrumentation/opencensus-proto/gen-go/agent/common/v1"
	ocresource "github.com/census-instrumentation/opencensus-proto/gen-go/resource/v1"
	"github.com/golang/protobuf/ptypes"
	"github.com/golang/protobuf/ptypes/timestamp"
	otlpcommon "github.com/open-telemetry/opentelemetry-proto/gen/go/common/v1"
	otlpresource "github.com/open-telemetry/opentelemetry-proto/gen/go/resource/v1"

	"github.com/open-telemetry/opentelemetry-collector/translator/conventions"
)

var (
	defaultTime      = time.Time{} // zero
	defaultTimestamp = timestamp.Timestamp{}
	defaultProcessID = 0
)

func ResourceToOC(resource *otlpresource.Resource) (*occommon.Node, *ocresource.Resource) {
	if resource == nil {
		return nil, nil
	}

	ocNode := occommon.Node{}
	ocResource := ocresource.Resource{}

	if len(resource.Attributes) == 0 {
		return &ocNode, &ocResource
	}

	labels := make(map[string]string, len(resource.Attributes))
	for _, attr := range resource.Attributes {
		key := attr.Key
		val := attributeValueToString(attr)

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
				t = defaultTime
			}
			ts, err := ptypes.TimestampProto(t)
			if err != nil {
				ts = &defaultTimestamp
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

func attributeValueToString(attr *otlpcommon.AttributeKeyValue) string {
	switch attr.Type {
	case otlpcommon.AttributeKeyValue_STRING:
		return attr.StringValue
	case otlpcommon.AttributeKeyValue_BOOL:
		return strconv.FormatBool(attr.BoolValue)
	case otlpcommon.AttributeKeyValue_DOUBLE:
		return strconv.FormatFloat(attr.DoubleValue, 'f', -1, 64)
	case otlpcommon.AttributeKeyValue_INT:
		return strconv.FormatInt(attr.IntValue, 10)
	default:
		return fmt.Sprintf("<Unknown OpenTelemetry attribute value type %q>", attr.Type)
	}
}
