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

// package configload implements the configuration Parser.
package configload

import (
	"github.com/spf13/cast"
	"github.com/spf13/viper"
)

const (
	// KeyDelimiter is used as the default key delimiter in the default viper instance.
	KeyDelimiter = "::"
)

// NewViper creates a new Viper instance with key delimiter KeyDelimiter instead of the
// default ".". This way configs can have keys that contain ".".
func NewViper() *viper.Viper {
	return viper.NewWithOptions(viper.KeyDelimiter(KeyDelimiter))
}

// NewParser creates a new Parser instance.
func NewParser() *Parser {
	return &Parser{
		v: NewViper(),
	}
}

// FromViper creates a Parser from a Viper instance.
func FromViper(v *viper.Viper) *Parser {
	return &Parser{v: v}
}

// Parser loads configuration.
type Parser struct {
	v *viper.Viper
}

// Viper returns the underlying Viper instance.
func (l *Parser) Viper() *viper.Viper {
	return l.v
}

// UnmarshalExact unmarshals the config into a struct, erroring if a field is nonexistent.
func (l *Parser) UnmarshalExact(intoCfg interface{}) error {
	return l.v.UnmarshalExact(intoCfg)
}

// ToStringMap creates a map[string]interface{} from a Parser.
func (l *Parser) ToStringMap() map[string]interface{} {
	return cast.ToStringMap(l.v)
}
