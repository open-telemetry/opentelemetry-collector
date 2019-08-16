// Copyright 2019, OpenTelemetry Authors
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

package opencensusexporter

import (
	"time"

	"github.com/open-telemetry/opentelemetry-service/config/configmodels"
)

// Config defines configuration for OpenCensus exporter.
type Config struct {
	configmodels.ExporterSettings `mapstructure:",squash"` // squash ensures fields are correctly decoded in embedded struct.

	// The target to which the exporter is going to send traces or metrics,
	// using the gRPC protocol. The valid syntax is described at
	// https://github.com/grpc/grpc/blob/master/doc/naming.md.
	Endpoint string `mapstructure:"endpoint"`

	// The compression key for supported compression types within
	// collector. Currently the only supported mode is `gzip`.
	Compression string `mapstructure:"compression"`

	// The headers associated with gRPC requests.
	Headers map[string]string `mapstructure:"headers"`

	// The number of workers that send the gRPC requests.
	NumWorkers int `mapstructure:"num-workers"`

	// certificate file for TLS credentials of gRPC client. Should
	// only be used if `secure` is set to true.
	CertPemFile string `mapstructure:"cert-pem-file"`

	// Whether to enable client transport security for the exporter's gRPC
	// connection. See [grpc.WithInsecure()](https://godoc.org/google.golang.org/grpc#WithInsecure).
	UseSecure bool `mapstructure:"secure,omitempty"`

	// The time period between each reconnection performed by the exporter.
	ReconnectionDelay time.Duration `mapstructure:"reconnection-delay,omitempty"`

	// The keepalive parameters for client gRPC. See grpc.WithKeepaliveParams
	// (https://godoc.org/google.golang.org/grpc#WithKeepaliveParams).
	KeepaliveParameters *KeepaliveConfig `mapstructure:"keepalive,omitempty"`
}
