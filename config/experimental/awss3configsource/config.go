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

package awss3configsource

import (
	splunkprovider "github.com/signalfx/splunk-otel-collector"
)

type Config struct {
	//AWS variables that likely won't be used but nice to have in case of future modification
	*splunkprovider.Settings
	//Region that the bucket is in e.g us-west-2
	Region string `mapstructure:"region"`
	//Name of the bucket the user is downloading from
	Bucket string `mapstructure:"bucket"`
	//Name of the specific document they are downloading
	Key string `mapstructure:"key"`
	//If the user has several versions of the same document this specifies which one
	VersionID string `mapstructure:"version_id"`
}
