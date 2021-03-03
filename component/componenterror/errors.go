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

// Package oterr provides helper functions to create and process
// OpenTelemetry specific errors
package componenterror

import (
	"errors"
)

var (
	// ErrAlreadyStarted indicates an error on starting an already-started component.
	ErrAlreadyStarted = errors.New("already started")

	// ErrAlreadyStopped indicates an error on stoping an already-stopped component.
	ErrAlreadyStopped = errors.New("already stopped")

	// ErrNilNextConsumer indicates an error on nil next consumer.
	ErrNilNextConsumer = errors.New("nil nextConsumer")
)
