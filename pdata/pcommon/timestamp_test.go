// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package pcommon

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestUnixNanosConverters(t *testing.T) {
	t1 := time.Date(2020, 3, 24, 1, 13, 23, 789, time.UTC)
	tun := Timestamp(t1.UnixNano())

	assert.EqualValues(t, uint64(1585012403000000789), tun)
	assert.Equal(t, tun, NewTimestampFromTime(t1))
	assert.Equal(t, t1, NewTimestampFromTime(t1).AsTime())
	assert.Equal(t, "2020-03-24 01:13:23.000000789 +0000 UTC", t1.String())
}

func TestZeroTimestamp(t *testing.T) {
	assert.Equal(t, time.Unix(0, 0).UTC(), Timestamp(0).AsTime())
	assert.Zero(t, NewTimestampFromTime(time.Unix(0, 0).UTC()))
	assert.Equal(t, "1970-01-01 00:00:00 +0000 UTC", Timestamp(0).String())
}
