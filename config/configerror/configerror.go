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

// Package configerror contains the common errors caused by malformed configs.
package configerror

import "errors"

// ErrDataTypeIsNotSupported can be returned by receiver, exporter or processor
// factory methods that create the entity if the particular telemetry
// data type is not supported by the receiver, exporter or processor.
var ErrDataTypeIsNotSupported = errors.New("telemetry type is not supported")
