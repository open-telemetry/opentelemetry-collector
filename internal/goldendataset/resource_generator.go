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

package goldendataset

import (
	"go.opentelemetry.io/collector/model/pdata"
	conventions "go.opentelemetry.io/collector/translator/conventions/v1.5.0"
)

// GenerateResource generates a PData Resource object with representative attributes for the
// underlying resource type specified by the rscID input parameter.
func GenerateResource(rscID PICTInputResource) pdata.Resource {
	resource := pdata.NewResource()
	switch rscID {
	case ResourceEmpty:
		break
	case ResourceVMOnPrem:
		appendOnpremVMAttributes(resource.Attributes())
	case ResourceVMCloud:
		appendCloudVMAttributes(resource.Attributes())
	case ResourceK8sOnPrem:
		appendOnpremK8sAttributes(resource.Attributes())
	case ResourceK8sCloud:
		appendCloudK8sAttributes(resource.Attributes())
	case ResourceFaas:
		appendFassAttributes(resource.Attributes())
	case ResourceExec:
		appendExecAttributes(resource.Attributes())
	}
	return resource
}

func appendOnpremVMAttributes(attrMap pdata.AttributeMap) {
	attrMap.UpsertString(conventions.AttributeServiceName, "customers")
	attrMap.UpsertString(conventions.AttributeServiceNamespace, "production")
	attrMap.UpsertString(conventions.AttributeServiceVersion, "semver:0.7.3")
	subMap := pdata.NewAttributeValueMap()
	subMap.MapVal().InsertString("public", "tc-prod9.internal.example.com")
	subMap.MapVal().InsertString("internal", "172.18.36.18")
	attrMap.Upsert(conventions.AttributeHostName, subMap)
	attrMap.UpsertString(conventions.AttributeHostImageID, "661ADFA6-E293-4870-9EFA-1AA052C49F18")
	attrMap.UpsertString(conventions.AttributeTelemetrySDKLanguage, conventions.AttributeTelemetrySDKLanguageJava)
	attrMap.UpsertString(conventions.AttributeTelemetrySDKName, "opentelemetry")
	attrMap.UpsertString(conventions.AttributeTelemetrySDKVersion, "0.3.0")
}

func appendCloudVMAttributes(attrMap pdata.AttributeMap) {
	attrMap.UpsertString(conventions.AttributeServiceName, "shoppingcart")
	attrMap.UpsertString(conventions.AttributeServiceName, "customers")
	attrMap.UpsertString(conventions.AttributeServiceNamespace, "production")
	attrMap.UpsertString(conventions.AttributeServiceVersion, "semver:0.7.3")
	attrMap.UpsertString(conventions.AttributeTelemetrySDKLanguage, conventions.AttributeTelemetrySDKLanguageJava)
	attrMap.UpsertString(conventions.AttributeTelemetrySDKName, "opentelemetry")
	attrMap.UpsertString(conventions.AttributeTelemetrySDKVersion, "0.3.0")
	attrMap.UpsertString(conventions.AttributeHostID, "57e8add1f79a454bae9fb1f7756a009a")
	attrMap.UpsertString(conventions.AttributeHostName, "env-check")
	attrMap.UpsertString(conventions.AttributeHostImageID, "5.3.0-1020-azure")
	attrMap.UpsertString(conventions.AttributeHostType, "B1ms")
	attrMap.UpsertString(conventions.AttributeCloudProvider, "azure")
	attrMap.UpsertString(conventions.AttributeCloudAccountID, "2f5b8278-4b80-4930-a6bb-d86fc63a2534")
	attrMap.UpsertString(conventions.AttributeCloudRegion, "South Central US")
}

func appendOnpremK8sAttributes(attrMap pdata.AttributeMap) {
	attrMap.UpsertString(conventions.AttributeContainerName, "cert-manager")
	attrMap.UpsertString(conventions.AttributeContainerImageName, "quay.io/jetstack/cert-manager-controller:v0.14.2")
	attrMap.UpsertString(conventions.AttributeK8SClusterName, "docker-desktop")
	attrMap.UpsertString(conventions.AttributeK8SNamespaceName, "cert-manager")
	attrMap.UpsertString(conventions.AttributeK8SDeploymentName, "cm-1-cert-manager")
	attrMap.UpsertString(conventions.AttributeK8SPodName, "cm-1-cert-manager-6448b4949b-t2jtd")
	attrMap.UpsertString(conventions.AttributeHostName, "docker-desktop")
}

func appendCloudK8sAttributes(attrMap pdata.AttributeMap) {
	attrMap.UpsertString(conventions.AttributeContainerName, "otel-collector")
	attrMap.UpsertString(conventions.AttributeContainerImageName, "otel/opentelemetry-collector-contrib")
	attrMap.UpsertString(conventions.AttributeContainerImageTag, "0.4.0")
	attrMap.UpsertString(conventions.AttributeK8SClusterName, "erp-dev")
	attrMap.UpsertString(conventions.AttributeK8SNamespaceName, "monitoring")
	attrMap.UpsertString(conventions.AttributeK8SDeploymentName, "otel-collector")
	attrMap.UpsertString(conventions.AttributeK8SDeploymentUID, "4D614B27-EDAF-409B-B631-6963D8F6FCD4")
	attrMap.UpsertString(conventions.AttributeK8SReplicasetName, "otel-collector-2983fd34")
	attrMap.UpsertString(conventions.AttributeK8SReplicasetUID, "EC7D59EF-D5B6-48B7-881E-DA6B7DD539B6")
	attrMap.UpsertString(conventions.AttributeK8SPodName, "otel-collector-6484db5844-c6f9m")
	attrMap.UpsertString(conventions.AttributeK8SPodUID, "FDFD941E-2A7A-4945-B601-88DD486161A4")
	attrMap.UpsertString(conventions.AttributeHostID, "ec2e3fdaffa294348bdf355156b94cda")
	attrMap.UpsertString(conventions.AttributeHostName, "10.99.118.157")
	attrMap.UpsertString(conventions.AttributeHostImageID, "ami-011c865bf7da41a9d")
	attrMap.UpsertString(conventions.AttributeHostType, "m5.xlarge")
	attrMap.UpsertString(conventions.AttributeCloudProvider, "aws")
	attrMap.UpsertString(conventions.AttributeCloudAccountID, "12345678901")
	attrMap.UpsertString(conventions.AttributeCloudRegion, "us-east-1")
	attrMap.UpsertString(conventions.AttributeCloudAvailabilityZone, "us-east-1c")
}

func appendFassAttributes(attrMap pdata.AttributeMap) {
	attrMap.UpsertString(conventions.AttributeFaaSID, "https://us-central1-dist-system-demo.cloudfunctions.net/env-vars-print")
	attrMap.UpsertString(conventions.AttributeFaaSName, "env-vars-print")
	attrMap.UpsertString(conventions.AttributeFaaSVersion, "semver:1.0.0")
	attrMap.UpsertString(conventions.AttributeCloudProvider, "gcp")
	attrMap.UpsertString(conventions.AttributeCloudAccountID, "opentelemetry")
	attrMap.UpsertString(conventions.AttributeCloudRegion, "us-central1")
	attrMap.UpsertString(conventions.AttributeCloudAvailabilityZone, "us-central1-a")
}

func appendExecAttributes(attrMap pdata.AttributeMap) {
	attrMap.UpsertString(conventions.AttributeProcessExecutableName, "otelcol")
	parts := pdata.NewAttributeValueArray()
	parts.ArrayVal().AppendEmpty().SetStringVal("otelcol")
	parts.ArrayVal().AppendEmpty().SetStringVal("--config=/etc/otel-collector-config.yaml")
	attrMap.Upsert(conventions.AttributeProcessCommandLine, parts)
	attrMap.UpsertString(conventions.AttributeProcessExecutablePath, "/usr/local/bin/otelcol")
	attrMap.UpsertInt(conventions.AttributeProcessPID, 2020)
	attrMap.UpsertString(conventions.AttributeProcessOwner, "otel")
	attrMap.UpsertString(conventions.AttributeOSType, "LINUX")
	attrMap.UpsertString(conventions.AttributeOSDescription,
		"Linux ubuntu 5.4.0-42-generic #46-Ubuntu SMP Fri Jul 10 00:24:02 UTC 2020 x86_64 x86_64 x86_64 GNU/Linux")
}
