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

// OpenTelemetry Semantic Convention values for Resource attribute names.
// See: https://github.com/open-telemetry/opentelemetry-specification/tree/main/specification/resource/semantic_conventions/README.md
const (
	AttributeCloudAccount          = "cloud.account.id"
	AttributeCloudAvailabilityZone = "cloud.availability_zone"
	AttributeCloudPlatform         = "cloud.platform"
	AttributeCloudProvider         = "cloud.provider"
	AttributeCloudRegion           = "cloud.region"
	AttributeContainerID           = "container.id"
	AttributeContainerImage        = "container.image.name"
	AttributeContainerName         = "container.name"
	AttributeContainerTag          = "container.image.tag"
	AttributeDeploymentEnvironment = "deployment.environment"
	AttributeFaasID                = "faas.id"
	AttributeFaasInstance          = "faas.instance"
	AttributeFaasName              = "faas.name"
	AttributeFaasVersion           = "faas.version"
	AttributeHostID                = "host.id"
	AttributeHostImageID           = "host.image.id"
	AttributeHostImageName         = "host.image.name"
	AttributeHostImageVersion      = "host.image.version"
	AttributeHostName              = "host.name"
	AttributeHostType              = "host.type"
	AttributeK8sCluster            = "k8s.cluster.name"
	AttributeK8sContainer          = "k8s.container.name"
	AttributeK8sCronJob            = "k8s.cronjob.name"
	AttributeK8sCronJobUID         = "k8s.cronjob.uid"
	AttributeK8sDaemonSet          = "k8s.daemonset.name"
	AttributeK8sDaemonSetUID       = "k8s.daemonset.uid"
	AttributeK8sDeployment         = "k8s.deployment.name"
	AttributeK8sDeploymentUID      = "k8s.deployment.uid"
	AttributeK8sJob                = "k8s.job.name"
	AttributeK8sJobUID             = "k8s.job.uid"
	AttributeK8sNamespace          = "k8s.namespace.name"
	AttributeK8sNodeName           = "k8s.node.name"
	AttributeK8sNodeUID            = "k8s.node.uid"
	AttributeK8sPod                = "k8s.pod.name"
	AttributeK8sPodUID             = "k8s.pod.uid"
	AttributeK8sReplicaSet         = "k8s.replicaset.name"
	AttributeK8sReplicaSetUID      = "k8s.replicaset.uid"
	AttributeK8sStatefulSet        = "k8s.statefulset.name"
	AttributeK8sStatefulSetUID     = "k8s.statefulset.uid"
	AttributeOSDescription         = "os.description"
	AttributeOSType                = "os.type"
	AttributeProcessCommand        = "process.command"
	AttributeProcessCommandLine    = "process.command_line"
	AttributeProcessExecutableName = "process.executable.name"
	AttributeProcessExecutablePath = "process.executable.path"
	AttributeProcessID             = "process.pid"
	AttributeProcessOwner          = "process.owner"
	AttributeServiceInstance       = "service.instance.id"
	AttributeServiceName           = "service.name"
	AttributeServiceNamespace      = "service.namespace"
	AttributeServiceVersion        = "service.version"
	AttributeTelemetryAutoVersion  = "telemetry.auto.version"
	AttributeTelemetrySDKLanguage  = "telemetry.sdk.language"
	AttributeTelemetrySDKName      = "telemetry.sdk.name"
	AttributeTelemetrySDKVersion   = "telemetry.sdk.version"
)

// OpenTelemetry Semantic Convention values for Resource attribute "telemetry.sdk.language" values.
// See: https://github.com/open-telemetry/opentelemetry-specification/tree/main/specification/resource/semantic_conventions/README.md
const (
	AttributeSDKLangValueCPP    = "cpp"
	AttributeSDKLangValueDotNET = "dotnet"
	AttributeSDKLangValueErlang = "erlang"
	AttributeSDKLangValueGo     = "go"
	AttributeSDKLangValueJava   = "java"
	AttributeSDKLangValueNodeJS = "nodejs"
	AttributeSDKLangValuePHP    = "php"
	AttributeSDKLangValuePython = "python"
	AttributeSDKLangValueRuby   = "ruby"
	AttributeSDKLangValueWebJS  = "webjs"
)

// OpenTelemetry Semantic Convention values for Resource attribute "cloud.provider" values.
// See: https://github.com/open-telemetry/opentelemetry-specification/blob/main/specification/resource/semantic_conventions/cloud.md
const (
	AttributeCloudProviderAWS   = "aws"
	AttributeCloudProviderAzure = "azure"
	AttributeCloudProviderGCP   = "gcp"
)

// OpenTelemetry Semantic Convention values for Resource attribute "cloud.platform" values.
// See: https://github.com/open-telemetry/opentelemetry-specification/blob/main/specification/resource/semantic_conventions/cloud.md
const (
	AttributeCloudPlatformAWSEC2                  = "aws_ec2"
	AttributeCloudPlatformAWSECS                  = "aws_ecs"
	AttributeCloudPlatformAWSEKS                  = "aws_eks"
	AttributeCloudPlatformAWSLambda               = "aws_lambda"
	AttributeCloudPlatformAWSElasticBeanstalk     = "aws_elastic_beanstalk"
	AttributeCloudPlatformAzureVM                 = "azure_vm"
	AttributeCloudPlatformAzureContainerInstances = "azure_container_instances"
	AttributeCloudPlatformAzureAKS                = "azure_aks"
	AttributeCloudPlatformAzureFunctions          = "azure_functions"
	AttributeCloudPlatformAzureAppService         = "azure_app_service"
	AttributeCloudPlatformGCPComputeEngine        = "gcp_compute_engine"
	AttributeCloudPlatformGCPCloudRun             = "gcp_cloud_run"
	AttributeCloudPlatformGCPGKE                  = "gcp_gke"
	AttributeCloudPlatformGCPCloudFunctions       = "gcp_cloud_functions"
	AttributeCloudPlatformGCPAppEngine            = "gcp_app_engine"
)

// GetResourceSemanticConventionAttributeNames a slice with all the Resource Semantic Conventions attribute names.
func GetResourceSemanticConventionAttributeNames() []string {
	return []string{
		AttributeCloudAccount,
		AttributeCloudProvider,
		AttributeCloudRegion,
		AttributeCloudAvailabilityZone,
		AttributeCloudPlatform,
		AttributeContainerID,
		AttributeContainerImage,
		AttributeContainerName,
		AttributeContainerTag,
		AttributeDeploymentEnvironment,
		AttributeFaasID,
		AttributeFaasInstance,
		AttributeFaasName,
		AttributeFaasVersion,
		AttributeHostID,
		AttributeHostImageID,
		AttributeHostImageName,
		AttributeHostImageVersion,
		AttributeHostName,
		AttributeHostType,
		AttributeK8sCluster,
		AttributeK8sContainer,
		AttributeK8sCronJob,
		AttributeK8sCronJobUID,
		AttributeK8sDaemonSet,
		AttributeK8sDaemonSetUID,
		AttributeK8sDeployment,
		AttributeK8sDeploymentUID,
		AttributeK8sJob,
		AttributeK8sJobUID,
		AttributeK8sNamespace,
		AttributeK8sNodeName,
		AttributeK8sNodeUID,
		AttributeK8sPod,
		AttributeK8sPodUID,
		AttributeK8sReplicaSet,
		AttributeK8sReplicaSetUID,
		AttributeK8sStatefulSet,
		AttributeK8sStatefulSetUID,
		AttributeOSType,
		AttributeOSDescription,
		AttributeProcessCommand,
		AttributeProcessCommandLine,
		AttributeProcessExecutableName,
		AttributeProcessExecutablePath,
		AttributeProcessID,
		AttributeProcessOwner,
		AttributeServiceInstance,
		AttributeServiceName,
		AttributeServiceNamespace,
		AttributeServiceVersion,
		AttributeTelemetryAutoVersion,
		AttributeTelemetrySDKLanguage,
		AttributeTelemetrySDKName,
		AttributeTelemetrySDKVersion,
	}
}
