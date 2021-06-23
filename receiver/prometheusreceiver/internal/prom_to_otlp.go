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

package internal

import (
	"net"

	"go.opentelemetry.io/collector/model/pdata"
	"go.opentelemetry.io/collector/translator/conventions"
)

func createNodeAndResourcePdata(job, instance, scheme string) pdata.Resource {
	host, port, err := net.SplitHostPort(instance)
	if err != nil {
		host = instance
	}
	resource := pdata.NewResource()
	attrs := resource.Attributes()
	attrs.UpsertString(conventions.AttributeServiceName, job)
	attrs.UpsertString(conventions.AttributeHostName, host)
	attrs.UpsertString(jobAttr, job)
	attrs.UpsertString(instanceAttr, instance)
	attrs.UpsertString(portAttr, port)
	attrs.UpsertString(schemeAttr, scheme)

	return resource
}
