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

package confmaptest // import "go.opentelemetry.io/collector/confmap/confmaptest"

import (
	"context"

	"go.opentelemetry.io/collector/confmap"
	"go.opentelemetry.io/collector/confmap/provider/fileprovider"
)

// LoadConf loads a confmap.Conf from file, and does NOT validate the configuration.
func LoadConf(fileName string) (*confmap.Conf, error) {
	ret, err := fileprovider.New().Retrieve(context.Background(), "file:"+fileName, nil)
	if err != nil {
		return nil, err
	}
	return ret.AsConf()
}
