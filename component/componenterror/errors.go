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
	"fmt"
	"strings"

	"go.opentelemetry.io/collector/consumer/consumererror"
)

var (
	// ErrAlreadyStarted indicates an error on starting an already-started component.
	ErrAlreadyStarted = errors.New("already started")

	// ErrAlreadyStopped indicates an error on stoping an already-stopped component.
	ErrAlreadyStopped = errors.New("already stopped")

	// ErrNilNextConsumer indicates an error on nil next consumer.
	ErrNilNextConsumer = errors.New("nil nextConsumer")
)

// CombineErrors converts a list of errors into one error.
func CombineErrors(errs []error) error {
	numErrors := len(errs)
	if numErrors == 0 {
		// No errors
		return nil
	}

	if numErrors == 1 {
		return errs[0]
	}

	errMsgs := make([]string, 0, numErrors)
	permanent := false
	for _, err := range errs {
		if !permanent && consumererror.IsPermanent(err) {
			permanent = true
		}
		errMsgs = append(errMsgs, err.Error())
	}
	err := fmt.Errorf("[%s]", strings.Join(errMsgs, "; "))
	if permanent {
		err = consumererror.Permanent(err)
	}
	return err
}
