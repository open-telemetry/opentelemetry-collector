// Copyright 2019 OpenTelemetry Authors
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

package configtestutils

import (
	"os"
	"path/filepath"

	"github.com/spf13/viper"
)

// CreateViperYamlUnmarshaler creates a viper instance that reads the given fileName as yaml config
// and can then be used to unmarshal the file contents to objects.
// Example usage for testing can be found in config_testutils_test.go
func CreateViperYamlUnmarshaler(fileName string) (*viper.Viper, error) {
	// Open the file for reading.
	file, err := os.Open(filepath.Clean(fileName))
	if err != nil {
		return nil, err
	}

	// Read yaml config from file
	v := viper.New()
	v.SetConfigType("yaml")
	err = v.ReadConfig(file)
	if err != nil {
		return nil, err
	}

	return v, nil
}
