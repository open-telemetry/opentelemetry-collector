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

package opencensusreceiver

import (
	"go.opentelemetry.io/collector/config/configgrpc"
)

// ocOption interface defines for configuration settings to be applied to receivers.
//
// withReceiver applies the configuration to the given receiver.
type ocOption interface {
	withReceiver(*ocReceiver)
}

type corsOrigins struct {
	origins []string
}

var _ ocOption = (*corsOrigins)(nil)

func (co *corsOrigins) withReceiver(ocr *ocReceiver) {
	ocr.corsOrigins = co.origins
}

// withCorsOrigins is an option to specify the allowed origins to enable writing
// HTTP/JSON requests to the grpc-gateway adapter using CORS.
func withCorsOrigins(origins []string) ocOption {
	return &corsOrigins{origins: origins}
}

type grpcServerSettings configgrpc.GRPCServerSettings

func withGRPCServerSettings(settings configgrpc.GRPCServerSettings) ocOption {
	gsvOpts := grpcServerSettings(settings)
	return gsvOpts
}
func (gsvo grpcServerSettings) withReceiver(ocr *ocReceiver) {
	ocr.grpcServerSettings = configgrpc.GRPCServerSettings(gsvo)
}
