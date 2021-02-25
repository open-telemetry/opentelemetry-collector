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

package conventions

// OpenTelemetry Semantic Convention attribute names for FaaS related attributes
// See: https://github.com/open-telemetry/opentelemetry-specification/blob/main/specification/trace/semantic_conventions/faas.md
const (
	AttributeFaaSCron               = "faas.cron"
	AttributeFaaSDocumentCollection = "faas.document.collection"
	AttributeFaaSDocumentName       = "faas.document.name"
	AttributeFaaSDocumentOperation  = "faas.document.operation"
	AttributeFaaSDocumentTime       = "faas.document.time"
	AttributeFaaSExecution          = "faas.execution"
	AttributeFaaSTime               = "faas.time"
	AttributeFaaSTrigger            = "faas.trigger"
	FaaSTriggerDataSource           = "datasource"
	FaaSTriggerHTTP                 = "http"
	FaaSTriggerOther                = "other"
	FaaSTriggerPubSub               = "pubsub"
	FaaSTriggerTimer                = "timer"
)
