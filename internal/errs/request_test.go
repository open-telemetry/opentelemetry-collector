// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package errs

import (
	"errors"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewRequestError(t *testing.T) {
	// prepare
	err := errors.New("boo")

	// test
	reqerr := NewRequestError(http.StatusInternalServerError, err)

	// verify
	assert.Error(t, reqerr)
	assert.Equal(t, err, reqerr.Unwrap())
	assert.Equal(t, http.StatusInternalServerError, reqerr.StatusCode())
	assert.Equal(t, "boo", reqerr.Error())
}
