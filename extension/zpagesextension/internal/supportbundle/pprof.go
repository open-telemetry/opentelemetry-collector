// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package supportbundle // import "go.opentelemetry.io/collector/extension/zpagesextension/internal/supportbundle"

import (
	"bytes"
	"context"
	"fmt"
	"runtime/pprof"
)

// PProfName is the name of the pprof gatherer.
const PProfName = "pprof"

// snapshotProfiles are the profiles that need no collection window.
var snapshotProfiles = []string{"heap", "goroutine"}

// PProfGatherer collects point-in-time runtime profiles in the pprof format.
type PProfGatherer struct{}

func (PProfGatherer) Name() string { return PProfName }

func (PProfGatherer) Gather(ctx context.Context) ([]File, error) {
	files := make([]File, 0, len(snapshotProfiles))
	for _, name := range snapshotProfiles {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		p := pprof.Lookup(name)
		if p == nil {
			return nil, fmt.Errorf("profile %q not found", name)
		}
		var buf bytes.Buffer
		// A debug value of 0 writes the gzipped protobuf format that "go tool pprof" reads.
		if err := p.WriteTo(&buf, 0); err != nil {
			return nil, fmt.Errorf("write %s profile: %w", name, err)
		}
		files = append(files, File{Path: "pprof/" + name + ".pprof", Content: buf.Bytes()})
	}
	return files, nil
}
