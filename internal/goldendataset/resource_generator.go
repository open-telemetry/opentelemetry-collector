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
	otlpcommon "go.opentelemetry.io/collector/internal/data/protogen/common/v1"
	otlpresource "go.opentelemetry.io/collector/internal/data/protogen/resource/v1"
	"go.opentelemetry.io/collector/translator/conventions"
)

// GenerateResource generates a OTLP Resource object with representative attributes for the
// underlying resource type specified by the rscID input parameter.
func GenerateResource(rscID PICTInputResource) otlpresource.Resource {
	var attrs map[string]interface{}
	switch rscID {
	case ResourceNil:
		attrs = generateNilAttributes()
	case ResourceEmpty:
		attrs = generateEmptyAttributes()
	case ResourceVMOnPrem:
		attrs = generateOnpremVMAttributes()
	case ResourceVMCloud:
		attrs = generateCloudVMAttributes()
	case ResourceK8sOnPrem:
		attrs = generateOnpremK8sAttributes()
	case ResourceK8sCloud:
		attrs = generateCloudK8sAttributes()
	case ResourceFaas:
		attrs = generateFassAttributes()
	case ResourceExec:
		attrs = generateExecAttributes()
	default:
		attrs = generateEmptyAttributes()
	}
	var dropped uint32
	if len(attrs) < 10 {
		dropped = 0
	} else {
		dropped = uint32(len(attrs) % 4)
	}
	return otlpresource.Resource{
		Attributes:             convertMapToAttributeKeyValues(attrs),
		DroppedAttributesCount: dropped,
	}
}

func generateNilAttributes() map[string]interface{} {
	return nil
}

func generateEmptyAttributes() map[string]interface{} {
	attrMap := make(map[string]interface{})
	return attrMap
}

func generateOnpremVMAttributes() map[string]interface{} {
	attrMap := make(map[string]interface{})
	attrMap[conventions.AttributeServiceName] = "customers"
	attrMap[conventions.AttributeServiceNamespace] = "production"
	attrMap[conventions.AttributeServiceVersion] = "semver:0.7.3"
	subMap := make(map[string]interface{})
	subMap["public"] = "tc-prod9.internal.example.com"
	subMap["internal"] = "172.18.36.18"
	attrMap[conventions.AttributeHostName] = &otlpcommon.KeyValueList{
		Values: convertMapToAttributeKeyValues(subMap),
	}
	attrMap[conventions.AttributeHostImageID] = "661ADFA6-E293-4870-9EFA-1AA052C49F18"
	attrMap[conventions.AttributeTelemetrySDKLanguage] = conventions.AttributeSDKLangValueJava
	attrMap[conventions.AttributeTelemetrySDKName] = "opentelemetry"
	attrMap[conventions.AttributeTelemetrySDKVersion] = "0.3.0"
	return attrMap
}

func generateCloudVMAttributes() map[string]interface{} {
	attrMap := make(map[string]interface{})
	attrMap[conventions.AttributeServiceName] = "shoppingcart"
	attrMap[conventions.AttributeServiceName] = "customers"
	attrMap[conventions.AttributeServiceNamespace] = "production"
	attrMap[conventions.AttributeServiceVersion] = "semver:0.7.3"
	attrMap[conventions.AttributeTelemetrySDKLanguage] = conventions.AttributeSDKLangValueJava
	attrMap[conventions.AttributeTelemetrySDKName] = "opentelemetry"
	attrMap[conventions.AttributeTelemetrySDKVersion] = "0.3.0"
	attrMap[conventions.AttributeHostID] = "57e8add1f79a454bae9fb1f7756a009a"
	attrMap[conventions.AttributeHostName] = "env-check"
	attrMap[conventions.AttributeHostImageID] = "5.3.0-1020-azure"
	attrMap[conventions.AttributeHostType] = "B1ms"
	attrMap[conventions.AttributeCloudProvider] = "azure"
	attrMap[conventions.AttributeCloudAccount] = "2f5b8278-4b80-4930-a6bb-d86fc63a2534"
	attrMap[conventions.AttributeCloudRegion] = "South Central US"
	return attrMap
}

func generateOnpremK8sAttributes() map[string]interface{} {
	attrMap := make(map[string]interface{})
	attrMap[conventions.AttributeContainerName] = "cert-manager"
	attrMap[conventions.AttributeContainerImage] = "quay.io/jetstack/cert-manager-controller:v0.14.2"
	attrMap[conventions.AttributeK8sCluster] = "docker-desktop"
	attrMap[conventions.AttributeK8sNamespace] = "cert-manager"
	attrMap[conventions.AttributeK8sDeployment] = "cm-1-cert-manager"
	attrMap[conventions.AttributeK8sPod] = "cm-1-cert-manager-6448b4949b-t2jtd"
	attrMap[conventions.AttributeHostName] = "docker-desktop"
	return attrMap
}

func generateCloudK8sAttributes() map[string]interface{} {
	attrMap := make(map[string]interface{})
	attrMap[conventions.AttributeContainerName] = "otel-collector"
	attrMap[conventions.AttributeContainerImage] = "otel/opentelemetry-collector-contrib"
	attrMap[conventions.AttributeContainerTag] = "0.4.0"
	attrMap[conventions.AttributeK8sCluster] = "erp-dev"
	attrMap[conventions.AttributeK8sNamespace] = "monitoring"
	attrMap[conventions.AttributeK8sDeployment] = "otel-collector"
	attrMap[conventions.AttributeK8sDeploymentUID] = "4D614B27-EDAF-409B-B631-6963D8F6FCD4"
	attrMap[conventions.AttributeK8sReplicaSet] = "otel-collector-2983fd34"
	attrMap[conventions.AttributeK8sReplicaSetUID] = "EC7D59EF-D5B6-48B7-881E-DA6B7DD539B6"
	attrMap[conventions.AttributeK8sPod] = "otel-collector-6484db5844-c6f9m"
	attrMap[conventions.AttributeK8sPodUID] = "FDFD941E-2A7A-4945-B601-88DD486161A4"
	attrMap[conventions.AttributeHostID] = "ec2e3fdaffa294348bdf355156b94cda"
	attrMap[conventions.AttributeHostName] = "10.99.118.157"
	attrMap[conventions.AttributeHostImageID] = "ami-011c865bf7da41a9d"
	attrMap[conventions.AttributeHostType] = "m5.xlarge"
	attrMap[conventions.AttributeCloudProvider] = "aws"
	attrMap[conventions.AttributeCloudAccount] = "12345678901"
	attrMap[conventions.AttributeCloudRegion] = "us-east-1"
	attrMap[conventions.AttributeCloudZone] = "us-east-1c"
	return attrMap
}

func generateFassAttributes() map[string]interface{} {
	attrMap := make(map[string]interface{})
	attrMap[conventions.AttributeFaasID] = "https://us-central1-dist-system-demo.cloudfunctions.net/env-vars-print"
	attrMap[conventions.AttributeFaasName] = "env-vars-print"
	attrMap[conventions.AttributeFaasVersion] = "semver:1.0.0"
	attrMap[conventions.AttributeCloudProvider] = "gcp"
	attrMap[conventions.AttributeCloudAccount] = "opentelemetry"
	attrMap[conventions.AttributeCloudRegion] = "us-central1"
	attrMap[conventions.AttributeCloudZone] = "us-central1-a"
	return attrMap
}

func generateExecAttributes() map[string]interface{} {
	attrMap := make(map[string]interface{})
	attrMap[conventions.AttributeProcessExecutableName] = "otelcol"
	parts := make([]otlpcommon.AnyValue, 3)
	parts[0] = otlpcommon.AnyValue{Value: &otlpcommon.AnyValue_StringValue{StringValue: "otelcol"}}
	parts[1] = otlpcommon.AnyValue{Value: &otlpcommon.AnyValue_StringValue{StringValue: "--config=/etc/otel-collector-config.yaml"}}
	parts[2] = otlpcommon.AnyValue{Value: &otlpcommon.AnyValue_StringValue{StringValue: "--mem-ballast-size-mib=683"}}
	attrMap[conventions.AttributeProcessCommandLine] = &otlpcommon.ArrayValue{
		Values: parts,
	}
	attrMap[conventions.AttributeProcessExecutablePath] = "/usr/local/bin/otelcol"
	attrMap[conventions.AttributeProcessID] = 2020
	attrMap[conventions.AttributeProcessOwner] = "otel"
	attrMap[conventions.AttributeOSType] = "LINUX"
	attrMap[conventions.AttributeOSDescription] =
		"Linux ubuntu 5.4.0-42-generic #46-Ubuntu SMP Fri Jul 10 00:24:02 UTC 2020 x86_64 x86_64 x86_64 GNU/Linux"
	return attrMap
}
