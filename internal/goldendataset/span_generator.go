// Copyright 2020, OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package goldendataset

import (
	"encoding/csv"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"time"

	otlpcommon "github.com/open-telemetry/opentelemetry-proto/gen/go/common/v1"
	otlptrace "github.com/open-telemetry/opentelemetry-proto/gen/go/trace/v1"

	"github.com/open-telemetry/opentelemetry-collector/translator/conventions"
)

const (
	ColumnParent     = 0
	ColumnTracestate = 1
	ColumnKind       = 2
	ColumnAttributes = 3
	ColumnEvents     = 4
	ColumnLinks      = 5
	ColumnStatus     = 6
)

const (
	SpanParentRoot  = "Root"
	SpanParentChild = "Child"
)

const (
	TraceStateEmpty = "Empty"
	TraceStateOne   = "One"
	TraceStateFour  = "Four"
)

const (
	SpanKindUnspecified = "Unspecified"
	SpanKindInternal    = "Internal"
	SpanKindServer      = "Server"
	SpanKindClient      = "Client"
	SpanKindProducer    = "Producer"
	SpanKindConsumer    = "Consumer"
)

const (
	SpanAttrNil               = "Nil"
	SpanAttrEmpty             = "Empty"
	SpanAttrDatabaseSQL       = "DatabaseSQL"
	SpanAttrDatabaseNoSQL     = "DatabaseNoSQL"
	SpanAttrFaaSDatasource    = "FaaSDatasource"
	SpanAttrFaaSHTTP          = "FaaSHTTP"
	SpanAttrFaaSPubSub        = "FaaSPubSub"
	SpanAttrFaaSTimer         = "FaaSTimer"
	SpanAttrFaaSOther         = "FaaSOther"
	SpanAttrHTTPClient        = "HTTPClient"
	SpanAttrHTTPServer        = "HTTPServer"
	SpanAttrMessagingProducer = "MessagingProducer"
	SpanAttrMessagingConsumer = "MessagingConsumer"
	SpanAttrGRPCClient        = "gRPCClient"
	SpanAttrGRPCServer        = "gRPCServer"
	SpanAttrInternal          = "Internal"
)

const (
	SpanChildCountNil   = "Nil"
	SpanChildCountEmpty = "Empty"
	SpanChildCountOne   = "One"
	SpanChildCountTwo   = "Two"
	SpanChildCountEight = "Eight"
)

const (
	SpanStatusNil                = "Nil"
	SpanStatusOk                 = "Ok"
	SpanStatusCancelled          = "Cancelled"
	SpanStatusUnknownError       = "UnknownError"
	SpanStatusInvalidArgument    = "InvalidArgument"
	SpanStatusDeadlineExceeded   = "DeadlineExceeded"
	SpanStatusNotFound           = "NotFound"
	SpanStatusAlreadyExists      = "AlreadyExists"
	SpanStatusPermissionDenied   = "PermissionDenied"
	SpanStatusResourceExhausted  = "ResourceExhausted"
	SpanStatusFailedPrecondition = "FailedPrecondition"
	SpanStatusAborted            = "Aborted"
	SpanStatusOutOfRange         = "OutOfRange"
	SpanStatusUnimplemented      = "Unimplemented"
	SpanStatusInternalError      = "InternalError"
	SpanStatusUnavailable        = "Unavailable"
	SpanStatusDataLoss           = "DataLoss"
	SpanStatusUnauthenticated    = "Unauthenticated"
)

var statusCodeMap = constructStatusCodeMap()
var statusMsgMap = constructStatusMessageMap()

func constructStatusCodeMap() map[string]otlptrace.Status_StatusCode {
	statusMap := make(map[string]otlptrace.Status_StatusCode)
	statusMap[SpanStatusOk] = otlptrace.Status_Ok
	statusMap[SpanStatusCancelled] = otlptrace.Status_Cancelled
	statusMap[SpanStatusUnknownError] = otlptrace.Status_UnknownError
	statusMap[SpanStatusInvalidArgument] = otlptrace.Status_InvalidArgument
	statusMap[SpanStatusDeadlineExceeded] = otlptrace.Status_DeadlineExceeded
	statusMap[SpanStatusNotFound] = otlptrace.Status_NotFound
	statusMap[SpanStatusAlreadyExists] = otlptrace.Status_AlreadyExists
	statusMap[SpanStatusPermissionDenied] = otlptrace.Status_PermissionDenied
	statusMap[SpanStatusResourceExhausted] = otlptrace.Status_ResourceExhausted
	statusMap[SpanStatusFailedPrecondition] = otlptrace.Status_FailedPrecondition
	statusMap[SpanStatusAborted] = otlptrace.Status_Aborted
	statusMap[SpanStatusOutOfRange] = otlptrace.Status_OutOfRange
	statusMap[SpanStatusUnimplemented] = otlptrace.Status_Unimplemented
	statusMap[SpanStatusInternalError] = otlptrace.Status_InternalError
	statusMap[SpanStatusUnavailable] = otlptrace.Status_Unavailable
	statusMap[SpanStatusDataLoss] = otlptrace.Status_DataLoss
	statusMap[SpanStatusUnauthenticated] = otlptrace.Status_Unauthenticated
	return statusMap
}

func constructStatusMessageMap() map[string]string {
	statusMap := make(map[string]string)
	statusMap[SpanStatusOk] = ""
	statusMap[SpanStatusCancelled] = "Cancellation received"
	statusMap[SpanStatusUnknownError] = ""
	statusMap[SpanStatusInvalidArgument] = "parameter is required"
	statusMap[SpanStatusDeadlineExceeded] = "timed out after 30002 ms"
	statusMap[SpanStatusNotFound] = "/dragons/RomanianLonghorn not found"
	statusMap[SpanStatusAlreadyExists] = "/dragons/Drogon already exists"
	statusMap[SpanStatusPermissionDenied] = "tlannister does not have write permission"
	statusMap[SpanStatusResourceExhausted] = "ResourceExhausted"
	statusMap[SpanStatusFailedPrecondition] = "33a64df551425fcc55e4d42a148795d9f25f89d4 has been edited"
	statusMap[SpanStatusAborted] = ""
	statusMap[SpanStatusOutOfRange] = "Range Not Satisfiable"
	statusMap[SpanStatusUnimplemented] = "Unimplemented"
	statusMap[SpanStatusInternalError] = "java.lang.NullPointerException"
	statusMap[SpanStatusUnavailable] = "RecommendationService is currently unavailable"
	statusMap[SpanStatusDataLoss] = ""
	statusMap[SpanStatusUnauthenticated] = "nstark is unknown user"
	return statusMap
}

func GenerateSpans(count int, startPos int, pictFile string) ([]*otlptrace.Span, int, error) {
	pairsData, err := loadPictOutputFile(pictFile)
	if err != nil {
		return nil, 0, err
	}
	pairsTotal := len(pairsData)
	spanList := make([]*otlptrace.Span, count)
	index := startPos + 1
	var inputs []string
	traceID := generateTraceID()
	parentID := generateSpanID()
	for i := 0; i < count; i++ {
		inputs = pairsData[index]
		if SpanParentRoot == inputs[ColumnParent] {
			traceID = generateTraceID()
			parentID = nil
		}
		spanName := generateSpanName(inputs[ColumnAttributes], i)
		spanList[i] = GenerateSpan(traceID, inputs[ColumnTracestate], parentID, spanName, inputs[ColumnKind],
			inputs[ColumnAttributes], inputs[ColumnEvents], inputs[ColumnLinks], inputs[ColumnStatus])
		parentID = spanList[i].SpanId
		index++
		if index >= pairsTotal {
			index = 1
		}
	}
	return spanList, index, nil
}

func loadPictOutputFile(fileName string) ([][]string, error) {
	file, err := os.Open(filepath.Clean(fileName))
	if err != nil {
		return nil, err
	}
	defer file.Close()

	reader := csv.NewReader(file)
	reader.Comma = '\t'

	return reader.ReadAll()
}

func generateSpanName(spanTypeID string, index int) string {
	if SpanAttrHTTPClient == spanTypeID {
		return fmt.Sprintf("/dragons/%d", index)
	} else if SpanAttrHTTPServer == spanTypeID {
		return "/dragons/{dragonId}"
	} else if SpanAttrGRPCClient == spanTypeID || SpanAttrGRPCServer == spanTypeID {
		return "com.example.PetFoodService/DispenseFeed"
	} else {
		return fmt.Sprintf("gotest%d", index)
	}
}

func GenerateSpan(traceID []byte, tracestate string, parentID []byte, spanName string, kind string, spanTypeID string,
	eventCnt string, linkCnt string, statusStr string) *otlptrace.Span {
	endTime := time.Now().Add(-50 * time.Microsecond)
	return &otlptrace.Span{
		TraceId:                traceID,
		SpanId:                 generateSpanID(),
		TraceState:             generateTraceState(tracestate),
		ParentSpanId:           parentID,
		Name:                   spanName,
		Kind:                   lookupSpanKind(kind),
		StartTimeUnixNano:      uint64(endTime.Add(-215 * time.Millisecond).UnixNano()),
		EndTimeUnixNano:        uint64(endTime.UnixNano()),
		Attributes:             generateSpanAttributes(spanTypeID),
		DroppedAttributesCount: 0,
		Events:                 generateSpanEvents(eventCnt),
		DroppedEventsCount:     0,
		Links:                  generateSpanLinks(linkCnt),
		DroppedLinksCount:      0,
		Status:                 generateStatus(statusStr),
	}
}

func generateTraceID() []byte {
	var r [16]byte
	_, err := rand.Read(r[:])
	if err != nil {
		panic(err)
	}
	return r[:]
}

func generateSpanID() []byte {
	var r [8]byte
	_, err := rand.Read(r[:])
	if err != nil {
		panic(err)
	}
	return r[:]
}

func generateTraceState(tracestate string) string {
	if TraceStateOne == tracestate {
		return "lasterror=f39cd56cc44274fd5abd07ef1164246d10ce2955"
	} else if TraceStateFour == tracestate {
		return "err@ck=80ee5638,rate@ck=1.62,rojo=00f067aa0ba902b7,congo=t61rcWkgMzE"
	} else {
		return ""
	}
}

func lookupSpanKind(kind string) otlptrace.Span_SpanKind {
	if SpanKindClient == kind {
		return otlptrace.Span_CLIENT
	} else if SpanKindServer == kind {
		return otlptrace.Span_SERVER
	} else if SpanKindProducer == kind {
		return otlptrace.Span_PRODUCER
	} else if SpanKindConsumer == kind {
		return otlptrace.Span_CONSUMER
	} else if SpanKindInternal == kind {
		return otlptrace.Span_INTERNAL
	} else {
		return otlptrace.Span_SPAN_KIND_UNSPECIFIED
	}
}

func generateSpanAttributes(spanTypeID string) []*otlpcommon.AttributeKeyValue {
	var attrs map[string]interface{}
	if SpanAttrNil == spanTypeID {
		attrs = nil
	} else if SpanAttrEmpty == spanTypeID {
		attrs = make(map[string]interface{})
	} else if SpanAttrDatabaseSQL == spanTypeID {
		attrs = generateDatabaseSQLAttributes()
	} else if SpanAttrDatabaseNoSQL == spanTypeID {
		attrs = generateDatabaseNoSQLAttributes()
	} else if SpanAttrFaaSDatasource == spanTypeID {
		attrs = generateFaaSDatasourceAttributes()
	} else if SpanAttrFaaSHTTP == spanTypeID {
		attrs = generateFaaSHTTPAttributes()
	} else if SpanAttrFaaSPubSub == spanTypeID {
		attrs = generateFaaSPubSubAttributes()
	} else if SpanAttrFaaSTimer == spanTypeID {
		attrs = generateFaaSTimerAttributes()
	} else if SpanAttrFaaSOther == spanTypeID {
		attrs = generateFaaSOtherAttributes()
	} else if SpanAttrHTTPClient == spanTypeID {
		attrs = generateHTTPClientAttributes()
	} else if SpanAttrHTTPServer == spanTypeID {
		attrs = generateHTTPServerAttributes()
	} else if SpanAttrMessagingProducer == spanTypeID {
		attrs = generateMessagingProducerAttributes()
	} else if SpanAttrMessagingConsumer == spanTypeID {
		attrs = generateMessagingConsumerAttributes()
	} else if SpanAttrGRPCClient == spanTypeID {
		attrs = generateGRPCClientAttributes()
	} else if SpanAttrGRPCServer == spanTypeID {
		attrs = generateGRPCServerAttributes()
	} else if SpanAttrInternal == spanTypeID {
		attrs = generateInternalAttributes()
	} else {
		panic("invalid spanTypeID")
	}
	return convertMapToAttributeKeyValues(attrs)
}

func generateStatus(statusStr string) *otlptrace.Status {
	if SpanStatusNil == statusStr {
		return nil
	}
	return &otlptrace.Status{
		Code:    statusCodeMap[statusStr],
		Message: statusMsgMap[statusStr],
	}
}

func generateDatabaseSQLAttributes() map[string]interface{} {
	attrMap := make(map[string]interface{})
	attrMap[conventions.AttributeDBType] = "sql"
	attrMap[conventions.AttributeDBInstance] = "inventory"
	attrMap[conventions.AttributeDBStatement] =
		"SELECT c.product_catg_id, c.catg_name, c.description, c.html_frag, c.image_url, p.name FROM product_catg c OUTER JOIN product p ON c.product_catg_id=p.product_catg_id WHERE c.product_catg_id = :catgId"
	attrMap[conventions.AttributeDBUser] = "invsvc"
	attrMap[conventions.AttributeDBURL] = "jdbc:postgresql://invdev.cdsr3wfqepqo.us-east-1.rds.amazonaws.com:5432/inventory"
	attrMap[conventions.AttributeNetPeerIP] = "172.30.2.7"
	attrMap[conventions.AttributeNetPeerPort] = int64(5432)
	attrMap[conventions.AttributeEnduserID] = "unittest"
	return attrMap
}

func generateDatabaseNoSQLAttributes() map[string]interface{} {
	attrMap := make(map[string]interface{})
	attrMap[conventions.AttributeDBType] = "cosmosdb"
	attrMap[conventions.AttributeDBInstance] = "graphdb"
	attrMap[conventions.AttributeDBStatement] = "g.V().hasLabel('postive').has('age', gt(65)).values('geocode')"
	attrMap[conventions.AttributeDBURL] = "wss://contacttrace.gremlin.cosmos.azure.com:443/"
	attrMap[conventions.AttributeNetPeerIP] = "10.118.17.63"
	attrMap[conventions.AttributeNetPeerPort] = int64(443)
	attrMap[conventions.AttributeEnduserID] = "unittest"
	return attrMap
}

func generateFaaSDatasourceAttributes() map[string]interface{} {
	attrMap := make(map[string]interface{})
	attrMap[conventions.AttributeFaaSTrigger] = conventions.FaaSTriggerDataSource
	attrMap[conventions.AttributeFaaSExecution] = "DB85AF51-5E13-473D-8454-1E2D59415EAB"
	attrMap[conventions.AttributeFaaSDocumentCollection] = "faa-flight-delay-information-incoming"
	attrMap[conventions.AttributeFaaSDocumentOperation] = "insert"
	attrMap[conventions.AttributeFaaSDocumentTime] = "2020-05-09T19:50:06Z"
	attrMap[conventions.AttributeFaaSDocumentName] = "delays-20200509-13.csv"
	attrMap[conventions.AttributeEnduserID] = "unittest"
	return attrMap
}

func generateFaaSHTTPAttributes() map[string]interface{} {
	attrMap := make(map[string]interface{})
	attrMap[conventions.AttributeFaaSTrigger] = conventions.FaaSTriggerHTTP
	attrMap[conventions.AttributeHTTPMethod] = "POST"
	attrMap[conventions.AttributeHTTPScheme] = "https"
	attrMap[conventions.AttributeHTTPHost] = "api.opentelemetry.io"
	attrMap[conventions.AttributeHTTPTarget] = "/blog/posts"
	attrMap[conventions.AttributeHTTPFlavor] = "2"
	attrMap[conventions.AttributeHTTPStatusCode] = int64(201)
	attrMap[conventions.AttributeHTTPUserAgent] =
		"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_4) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/13.1 Safari/605.1.15"
	attrMap[conventions.AttributeEnduserID] = "unittest"
	return attrMap
}

func generateFaaSPubSubAttributes() map[string]interface{} {
	attrMap := make(map[string]interface{})
	attrMap[conventions.AttributeFaaSTrigger] = conventions.FaaSTriggerPubSub
	attrMap[conventions.AttributeMessagingSystem] = "sqs"
	attrMap[conventions.AttributeMessagingDestination] = "video-views-au"
	attrMap[conventions.AttributeMessagingOperation] = "process"
	attrMap[conventions.AttributeEnduserID] = "unittest"
	return attrMap
}

func generateFaaSTimerAttributes() map[string]interface{} {
	attrMap := make(map[string]interface{})
	attrMap[conventions.AttributeFaaSTrigger] = conventions.FaaSTriggerTimer
	attrMap[conventions.AttributeFaaSExecution] = "73103A4C-E22F-4493-BDE8-EAE5CAB37B50"
	attrMap[conventions.AttributeFaaSTime] = "2020-05-09T20:00:08Z"
	attrMap[conventions.AttributeFaaSCron] = "0/15 * * * *"
	attrMap[conventions.AttributeEnduserID] = "unittest"
	return attrMap
}

func generateFaaSOtherAttributes() map[string]interface{} {
	attrMap := make(map[string]interface{})
	attrMap[conventions.AttributeFaaSTrigger] = conventions.FaaSTriggerOther
	attrMap["processed.count"] = int64(256)
	attrMap["processed.data"] = 14.46
	attrMap["processed.errors"] = false
	attrMap[conventions.AttributeEnduserID] = "unittest"
	return attrMap
}

func generateHTTPClientAttributes() map[string]interface{} {
	attrMap := make(map[string]interface{})
	attrMap[conventions.AttributeHTTPMethod] = "GET"
	attrMap[conventions.AttributeHTTPURL] = "https://opentelemetry.io/registry/"
	attrMap[conventions.AttributeHTTPStatusCode] = int64(200)
	attrMap[conventions.AttributeHTTPStatusText] = "More Than OK"
	attrMap[conventions.AttributeEnduserID] = "unittest"
	return attrMap
}

func generateHTTPServerAttributes() map[string]interface{} {
	attrMap := make(map[string]interface{})
	attrMap[conventions.AttributeHTTPMethod] = "POST"
	attrMap[conventions.AttributeHTTPScheme] = "https"
	attrMap[conventions.AttributeHTTPServerName] = "api22.opentelemetry.io"
	attrMap[conventions.AttributeNetHostPort] = int64(443)
	attrMap[conventions.AttributeHTTPTarget] = "/blog/posts"
	attrMap[conventions.AttributeHTTPFlavor] = "2"
	attrMap[conventions.AttributeHTTPStatusCode] = int64(201)
	attrMap[conventions.AttributeHTTPUserAgent] =
		"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_4) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/81.0.4044.138 Safari/537.36"
	attrMap[conventions.AttributeHTTPRoute] = "/blog/posts"
	attrMap[conventions.AttributeHTTPClientIP] = "2001:506:71f0:16e::1"
	attrMap[conventions.AttributeEnduserID] = "unittest"
	return attrMap
}

func generateMessagingProducerAttributes() map[string]interface{} {
	attrMap := make(map[string]interface{})
	attrMap[conventions.AttributeMessagingSystem] = "nats"
	attrMap[conventions.AttributeMessagingDestination] = "time.us.east.atlanta"
	attrMap[conventions.AttributeMessagingDestinationKind] = "topic"
	attrMap[conventions.AttributeMessagingMessageID] = "AA7C5438-D93A-43C8-9961-55613204648F"
	attrMap["messaging.sequence"] = int64(1)
	attrMap[conventions.AttributeNetPeerIP] = "10.10.212.33"
	attrMap[conventions.AttributeEnduserID] = "unittest"
	return attrMap
}

func generateMessagingConsumerAttributes() map[string]interface{} {
	attrMap := make(map[string]interface{})
	attrMap[conventions.AttributeMessagingSystem] = "kafka"
	attrMap[conventions.AttributeMessagingDestination] = "infrastructure-events-zone1"
	attrMap[conventions.AttributeMessagingOperation] = "receive"
	attrMap[conventions.AttributeNetPeerIP] = "2600:1700:1f00:11c0:4de0:c223:a800:4e87"
	attrMap[conventions.AttributeEnduserID] = "unittest"
	return attrMap
}

func generateGRPCClientAttributes() map[string]interface{} {
	attrMap := make(map[string]interface{})
	attrMap[conventions.AttributeRPCService] = "PullRequestsService"
	attrMap[conventions.AttributeNetPeerIP] = "2600:1700:1f00:11c0:4de0:c223:a800:4e87"
	attrMap[conventions.AttributeNetHostPort] = int64(8443)
	attrMap[conventions.AttributeEnduserID] = "unittest"
	return attrMap
}

func generateGRPCServerAttributes() map[string]interface{} {
	attrMap := make(map[string]interface{})
	attrMap[conventions.AttributeRPCService] = "PullRequestsService"
	attrMap[conventions.AttributeNetPeerIP] = "192.168.1.70"
	attrMap[conventions.AttributeEnduserID] = "unittest"
	return attrMap
}

func generateInternalAttributes() map[string]interface{} {
	attrMap := make(map[string]interface{})
	attrMap["parameters"] = "account=7310,amount=1817.10"
	attrMap[conventions.AttributeEnduserID] = "unittest"
	return attrMap
}

func generateSpanEvents(eventCnt string) []*otlptrace.Span_Event {
	if SpanChildCountNil == eventCnt {
		return nil
	}
	listSize := calculateListSize(eventCnt)
	eventList := make([]*otlptrace.Span_Event, listSize)
	for i := 0; i < listSize; i++ {
		eventList[i] = generateSpanEvent(i)
	}
	return eventList
}

func generateSpanLinks(linkCnt string) []*otlptrace.Span_Link {
	if SpanChildCountNil == linkCnt {
		return nil
	}
	listSize := calculateListSize(linkCnt)
	linkList := make([]*otlptrace.Span_Link, listSize)
	for i := 0; i < listSize; i++ {
		linkList[i] = generateSpanLink(i)
	}
	return linkList
}

func calculateListSize(listCnt string) int {
	if SpanChildCountOne == listCnt {
		return 1
	} else if SpanChildCountTwo == listCnt {
		return 2
	} else if SpanChildCountEight == listCnt {
		return 8
	} else {
		return 0
	}
}

func generateSpanEvent(index int) *otlptrace.Span_Event {
	t := time.Now().Add(-75 * time.Microsecond)
	return &otlptrace.Span_Event{
		TimeUnixNano:           uint64(t.UnixNano()),
		Name:                   "message",
		Attributes:             generateEventAttributes(index),
		DroppedAttributesCount: 0,
	}
}

func generateEventAttributes(index int) []*otlpcommon.AttributeKeyValue {
	attrMap := make(map[string]interface{})
	if index%2 == 0 {
		attrMap[conventions.AttributeMessageType] = "SENT"
	} else {
		attrMap[conventions.AttributeMessageType] = "RECEIVED"
	}
	attrMap[conventions.AttributeMessageID] = int64(index)
	attrMap[conventions.AttributeMessageCompressedSize] = int64(17 * index)
	attrMap[conventions.AttributeMessageUncompressedSize] = int64(24 * index)
	return convertMapToAttributeKeyValues(attrMap)
}

func generateSpanLink(index int) *otlptrace.Span_Link {
	return &otlptrace.Span_Link{
		TraceId:                generateTraceID(),
		SpanId:                 generateSpanID(),
		TraceState:             "",
		Attributes:             generateLinkAttributes(index),
		DroppedAttributesCount: 0,
	}
}

func generateLinkAttributes(index int) []*otlpcommon.AttributeKeyValue {
	attrMap := generateMessagingConsumerAttributes()
	return convertMapToAttributeKeyValues(attrMap)
}
