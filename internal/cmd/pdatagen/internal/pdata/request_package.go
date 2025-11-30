// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package pdata // import "go.opentelemetry.io/collector/internal/cmd/pdatagen/internal/pdata"

import (
	"path/filepath"

	"go.opentelemetry.io/collector/internal/cmd/pdatagen/internal/proto"
)

var prequest = &Package{
	info: &PackageInfo{
		name: "request",
		path: filepath.Join("xpdata", "request", "internal"),
		imports: []string{
			`"encoding/binary"`,
			`"fmt"`,
			`"iter"`,
			`"math"`,
			`"sort"`,
			``,
			`"go.opentelemetry.io/collector/pdata/internal"`,
			`"go.opentelemetry.io/collector/pdata/internal/json"`,
			`"go.opentelemetry.io/collector/pdata/internal/proto"`,
			`"go.opentelemetry.io/collector/pdata/pcommon"`,
		},
		testImports: []string{
			`"testing"`,
			``,
			`"github.com/stretchr/testify/assert"`,
			`"google.golang.org/protobuf/types/known/emptypb"`,
			``,
			`"go.opentelemetry.io/collector/pdata/internal"`,
			`"go.opentelemetry.io/collector/pdata/internal/json"`,
			`"go.opentelemetry.io/collector/pdata/pcommon"`,
		},
	},
	structs: []baseStruct{
		spanContext,
		ipAddr,
		tcpAddr,
		udpAddr,
		unixAddr,
		requestContext,
		tracesRequest,
		metricsRequest,
		logsRequest,
		profilesRequest,
	},
}

var spanContext = &messageStruct{
	structName:    "SpanContext",
	packageName:   "request",
	protoName:     "SpanContext",
	upstreamProto: "emptypb.Empty",
	fields: []Field{
		&TypedField{
			fieldName:  "TraceID",
			protoID:    1,
			returnType: traceIDType,
		},
		&TypedField{
			fieldName:  "SpanID",
			protoID:    2,
			returnType: spanIDType,
		},
		&PrimitiveField{
			fieldName: "TraceFlags",
			protoID:   3,
			protoType: proto.TypeFixed32,
		},
		&MessageField{
			fieldName:     "TraceState",
			protoID:       4,
			returnMessage: traceState,
		},
		&PrimitiveField{
			fieldName: "Remote",
			protoID:   5,
			protoType: proto.TypeBool,
		},
	},
	hasOnlyInternal: true,
}

var ipAddr = &messageStruct{
	structName:    "IPAddr",
	packageName:   "request",
	protoName:     "IPAddr",
	upstreamProto: "emptypb.Empty",
	fields: []Field{
		&PrimitiveField{
			fieldName: "IP",
			protoID:   1,
			protoType: proto.TypeBytes,
		},
		&PrimitiveField{
			fieldName: "Zone",
			protoID:   2,
			protoType: proto.TypeString,
		},
	},
	hasOnlyInternal: true,
}

var tcpAddr = &messageStruct{
	structName:    "TCPAddr",
	packageName:   "request",
	protoName:     "TCPAddr",
	upstreamProto: "emptypb.Empty",
	fields: []Field{
		&PrimitiveField{
			fieldName: "IP",
			protoID:   1,
			protoType: proto.TypeBytes,
		},
		&PrimitiveField{
			fieldName: "Port",
			protoID:   2,
			protoType: proto.TypeInt64,
		},
		&PrimitiveField{
			fieldName: "Zone",
			protoID:   3,
			protoType: proto.TypeString,
		},
	},
	hasOnlyInternal: true,
}

var udpAddr = &messageStruct{
	structName:    "UDPAddr",
	packageName:   "request",
	protoName:     "UDPAddr",
	upstreamProto: "emptypb.Empty",
	fields: []Field{
		&PrimitiveField{
			fieldName: "IP",
			protoID:   1,
			protoType: proto.TypeBytes,
		},
		&PrimitiveField{
			fieldName: "Port",
			protoID:   2,
			protoType: proto.TypeInt64,
		},
		&PrimitiveField{
			fieldName: "Zone",
			protoID:   3,
			protoType: proto.TypeString,
		},
	},
	hasOnlyInternal: true,
}

var unixAddr = &messageStruct{
	structName:    "UnixAddr",
	packageName:   "request",
	protoName:     "UnixAddr",
	upstreamProto: "emptypb.Empty",
	fields: []Field{
		&PrimitiveField{
			fieldName: "Name",
			protoID:   1,
			protoType: proto.TypeString,
		},
		&PrimitiveField{
			fieldName: "Net",
			protoID:   2,
			protoType: proto.TypeString,
		},
	},
	hasOnlyInternal: true,
}

var requestContext = &messageStruct{
	structName:    "RequestContext",
	packageName:   "request",
	protoName:     "RequestContext",
	upstreamProto: "emptypb.Empty",
	fields: []Field{
		&MessageField{
			fieldName:     "SpanContext",
			protoID:       1,
			nullable:      true,
			returnMessage: spanContext,
		},
		&SliceField{
			fieldName:   "ClientMetadata",
			protoID:     2,
			protoType:   proto.TypeMessage,
			returnSlice: mapStruct,
		},
		&OneOfField{
			typeName:                   "ClientAddressType",
			originFieldName:            "ClientAddress",
			testValueIdx:               1, //
			omitOriginFieldNameInNames: true,
			values: []oneOfValue{
				&OneOfMessageValue{
					fieldName:     "IP",
					protoID:       3,
					returnMessage: ipAddr,
				},
				&OneOfMessageValue{
					fieldName:     "TCP",
					protoID:       4,
					returnMessage: tcpAddr,
				},
				&OneOfMessageValue{
					fieldName:     "UDP",
					protoID:       5,
					returnMessage: udpAddr,
				},
				&OneOfMessageValue{
					fieldName:     "Unix",
					protoID:       6,
					returnMessage: unixAddr,
				},
			},
		},
	},
	hasOnlyInternal: true,
}

var tracesRequest = &messageStruct{
	structName:    "TracesRequest",
	packageName:   "request",
	protoName:     "TracesRequest",
	upstreamProto: "emptypb.Empty",
	fields: []Field{
		&MessageField{
			fieldName:     "RequestContext",
			protoID:       2,
			nullable:      true,
			returnMessage: requestContext,
		},
		&MessageField{
			fieldName:     "TracesData",
			protoID:       3,
			returnMessage: tracesData,
		},
		&PrimitiveField{
			fieldName: "FormatVersion",
			protoID:   1,
			protoType: proto.TypeFixed32,
		},
	},
	hasOnlyInternal: true,
}

var metricsRequest = &messageStruct{
	structName:    "MetricsRequest",
	packageName:   "request",
	protoName:     "MetricsRequest",
	upstreamProto: "emptypb.Empty",
	fields: []Field{
		&MessageField{
			fieldName:     "RequestContext",
			protoID:       2,
			nullable:      true,
			returnMessage: requestContext,
		},
		&MessageField{
			fieldName:     "MetricsData",
			protoID:       3,
			returnMessage: metricsData,
		},
		&PrimitiveField{
			fieldName: "FormatVersion",
			protoID:   1,
			protoType: proto.TypeFixed32,
		},
	},
	hasOnlyInternal: true,
}

var logsRequest = &messageStruct{
	structName:    "LogsRequest",
	packageName:   "request",
	protoName:     "LogsRequest",
	upstreamProto: "emptypb.Empty",
	fields: []Field{
		&MessageField{
			fieldName:     "RequestContext",
			protoID:       2,
			nullable:      true,
			returnMessage: requestContext,
		},
		&MessageField{
			fieldName:     "LogsData",
			protoID:       3,
			returnMessage: logsData,
		},
		&PrimitiveField{
			fieldName: "FormatVersion",
			protoID:   1,
			protoType: proto.TypeFixed32,
		},
	},
	hasOnlyInternal: true,
}

var profilesRequest = &messageStruct{
	structName:    "ProfilesRequest",
	packageName:   "request",
	protoName:     "ProfilesRequest",
	upstreamProto: "emptypb.Empty",
	fields: []Field{
		&MessageField{
			fieldName:     "RequestContext",
			protoID:       2,
			nullable:      true,
			returnMessage: requestContext,
		},
		&MessageField{
			fieldName:     "ProfilesData",
			protoID:       3,
			returnMessage: profilesData,
		},
		&PrimitiveField{
			fieldName: "FormatVersion",
			protoID:   1,
			protoType: proto.TypeFixed32,
		},
	},
	hasOnlyInternal: true,
}
