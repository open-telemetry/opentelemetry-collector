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

// Package receiver contains implementations of Receiver components.
//
// To implement a custom receiver you will need to implement component.ReceiverFactory
// interface and component.Receiver interface.
//
// To make the custom receiver part of the Collector build the factory must be added
// to defaultcomponents.Components() function.
package receiver
