// Copyright 2019, OpenCensus Authors
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

package grpc

import (
	"strings"

	"google.golang.org/grpc/encoding/gzip"

	"github.com/census-instrumentation/opencensus-service/internal/compression"
)

var (
	// Map of opencensus compression types to grpc registered compression types
	grpcCompressionKeyMap = map[string]string{
		compression.Gzip: gzip.Name,
	}
)

// GetGRPCCompressionKey returns the grpc registered compression key if the
// passed in compression key is supported, and Unsupported otherwise
func GetGRPCCompressionKey(compressionType string) string {
	compressionKey := strings.ToLower(compressionType)
	if encodingKey, ok := grpcCompressionKeyMap[compressionKey]; ok {
		return encodingKey
	}
	return compression.Unsupported
}
