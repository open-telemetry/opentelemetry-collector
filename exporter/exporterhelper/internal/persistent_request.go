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

package internal // import "go.opentelemetry.io/collector/exporter/exporterhelper/internal"

// PersistentRequest defines capabilities required for persistent storage of a request
type PersistentRequest interface {
	// Marshal serializes the current request into a byte stream
	Marshal() ([]byte, error)
	// OnProcessingFinished calls the optional callback function to handle cleanup after all processing is finished
	OnProcessingFinished()
	// SetOnProcessingFinished allows to set an optional callback function to do the cleanup (e.g. remove the item from persistent queue)
	SetOnProcessingFinished(callback func())
}

// RequestUnmarshaler defines a function which takes a byte slice and unmarshals it into a relevant request
type RequestUnmarshaler func([]byte) (PersistentRequest, error)
