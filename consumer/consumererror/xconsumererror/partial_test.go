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
	partialErr, ok := xconsumererror.AsPartial(err)
	assert.True(t, ok)
	assert.Equal(t, 5, partialErr.Failed())
	assert.Equal(t, internalErr, partialErr.Unwrap())
	assert.Equal(t, internalErr.Error(), partialErr.Error())
}

func ExampleNewPartial() {
	// Produce a partial error.
	failedItems := 5
	consumptionErr := errors.New("some points failed to be written")
	err := xconsumererror.NewPartial(consumptionErr, failedItems)
	fmt.Println(err)

	// Output: "Permanent error: some points failed to be written"
}

func ExampleAsPartial() {
	// Produce a partial error.
	partialErr := xconsumererror.NewPartial(errors.New("some points failed to be written"), 10)

	// The result of `xconsumererror.AsPartial` has a method `Failed`
	// which can be used to retrieve the failed item count.
	if err, ok := xconsumererror.AsPartial(partialErr); ok {
		fmt.Println(err.Failed())
	}

	// Output: 10
}

func ExampleAsPartial_second() {
	// Produce a partial error.
	partialErr := xconsumererror.NewPartial(errors.New("some points failed to be written"), 10)

	// Partial errors wrap a Permanent error.
	fmt.Println(consumererror.IsPermanent(partialErr))

	// Output: true
}
