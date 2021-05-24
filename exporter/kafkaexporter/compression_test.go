// Copyright  The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package kafkaexporter

import (
	"testing"

	"github.com/Shopify/sarama"
)

func TestConfigureCompression(t *testing.T) {
	saramaSample := &sarama.Config{}

	tests := []struct {
		compression  string
		saramaConfig *sarama.Config
		wantErr bool
	}{
		{
			compression: "none",
			saramaConfig: saramaSample,
		},
		{
			compression: "gzip",
			saramaConfig: saramaSample,
		},
		{
			compression: "snappy",
			saramaConfig: saramaSample,
		},
		{
			compression: "lz4",
			saramaConfig: saramaSample,
		},
		{
			compression: "zstd",
			saramaConfig: saramaSample,
		},
		{
			compression: "random",
			saramaConfig: saramaSample,
		},
	}
	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			if err := ConfigureCompression(tt.compression, tt.saramaConfig); (err != nil) != tt.wantErr {
				t.Errorf("ConfigureCompression() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}