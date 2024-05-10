// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package extensions // import "go.opentelemetry.io/collector/service/extensions"

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"gonum.org/v1/gonum/graph"
	"gonum.org/v1/gonum/graph/topo"
)

func TestCycleErr(t *testing.T) {
	err := errors.New("foo")
	assert.Equal(t, err, cycleErr(err, nil), "cycleErr should return the error unchanged when it's unrecognized")

	var topoErr topo.Unorderable = [][]graph.Node{{}}
	assert.Equal(t, topoErr, cycleErr(topoErr, nil), "cycleErr should return topo.Unorderable error unchanged when no cycles are found")
}
