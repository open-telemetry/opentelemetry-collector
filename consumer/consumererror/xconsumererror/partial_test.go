// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package xconsumererror_test

import (
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"

	"go.opentelemetry.io/collector/consumer/consumererror"
	"go.opentelemetry.io/collector/consumer/consumererror/xconsumererror"
)

func TestPartial(t *testing.T) {
	internalErr := errors.New("some points failed")
	err := xconsumererror.NewPartial(internalErr, 5)
	assert.True(t, consumererror.IsPermanent(err))
	failed, ok := xconsumererror.IsPartial(err)
	assert.True(t, ok)
	assert.Equal(t, 5, failed)
}

func ExampleNewPartial() {
	// Produce a partial error.
	failedItems := 5
	consumptionErr := errors.New("some points failed to be written")
	err := xconsumererror.NewPartial(consumptionErr, failedItems)
	fmt.Println(err)

	// Output: Permanent error: some points failed to be written
}

func ExampleIsPartial() {
	// Produce a partial error.
	partialErr := xconsumererror.NewPartial(errors.New("some points failed to be written"), 10)

	// IsPartial will return the failed item count, and a boolean
	// to indicate whether it is a partial error or not.
	if count, ok := xconsumererror.IsPartial(partialErr); ok {
		fmt.Println(count)
	}

	// Output: 10
}

func ExampleIsPartial_second() {
	// Produce a partial error.
	partialErr := xconsumererror.NewPartial(errors.New("some points failed to be written"), 10)

	// Partial errors wrap a Permanent error, so it can simply be treated
	// as one if the number of failed items isn't important.
	fmt.Println(consumererror.IsPermanent(partialErr))

	// Output: true
}
