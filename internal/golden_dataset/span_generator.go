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

package golden_dataset

import (
	"github.com/open-telemetry/opentelemetry-collector/translator/conventions"
)

const (
	SpanAttrNil       = "Nil"
	SpanAttrEmpty     = "Empty"
	SpanAttrDatabase  = "Database"
	SpanAttrFaaS      = "FaaS"
	SpanAttrHTTP      = "HTTP"
	SpanAttrMessaging = "Messaging"
	SpanAttrGRPC      = "gRPC"
)

func generateDatabaseAttributes() map[string]interface{} {
	attrMap := make(map[string]interface{})
	attrMap[conventions.AttributeDBType] = "sql"
	return attrMap
}

func generateFaaSAttributes() map[string]interface{} {
	attrMap := make(map[string]interface{})
	return attrMap
}

func generateHTTPClientAttributes() map[string]interface{} {
	attrMap := make(map[string]interface{})
	attrMap[conventions.AttributeHTTPMethod] = "GET"
	return attrMap
}

func generateHTTPServerAttributes() map[string]interface{} {
	attrMap := make(map[string]interface{})
	attrMap[conventions.AttributeHTTPMethod] = "GET"
	return attrMap
}

func generateMessagingAttributes() map[string]interface{} {
	attrMap := make(map[string]interface{})
	return attrMap
}

func generateGRPCAttributes() map[string]interface{} {
	attrMap := make(map[string]interface{})
	return attrMap
}
