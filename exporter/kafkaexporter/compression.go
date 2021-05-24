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
	"fmt"
	"strings"

	"github.com/Shopify/sarama"
)

func configureCompression(compression string, saramaConfig *sarama.Config) error {
	switch strings.ToLower(compression) {
	case "none":
		saramaConfig.Producer.Compression = sarama.CompressionNone
	case "gzip":
		saramaConfig.Producer.Compression = sarama.CompressionGZIP
	case "snappy":
		saramaConfig.Producer.Compression = sarama.CompressionSnappy
	case "lz4":
		saramaConfig.Producer.Compression = sarama.CompressionLZ4
	case "zstd":
		saramaConfig.Producer.Compression = sarama.CompressionZSTD
	default:
		return fmt.Errorf("invalid compression %q: can be one of \"none\" , \"gzip\", \"snappy\", \"lz4\" or \"zstd\"", compression)
	}

	return nil
}
