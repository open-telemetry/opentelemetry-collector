// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package pprofile // import "go.opentelemetry.io/collector/pdata/pprofile"

import "fmt"

// switchDictionary updates the Sample, switching its indices from one
// dictionary to another.
func (ms Sample) switchDictionary(src, dst ProfilesDictionary) error {
	if ms.LinkIndex() > 0 {
		idx, err := SetLink(dst.LinkTable(), src.LinkTable().At(int(ms.LinkIndex())))
		if err != nil {
			return fmt.Errorf("couldn't set link: %w", err)
		}
		ms.SetLinkIndex(idx)
	}

	if ms.StackIndex() > 0 {
		stack := src.StackTable().At(int(ms.StackIndex()))
		err := stack.switchDictionary(src, dst)
		if err != nil {
			return fmt.Errorf("couldn't switch stack dictionary: %w", err)
		}

		idx, err := SetStack(dst.StackTable(), stack)
		if err != nil {
			return fmt.Errorf("couldn't set stack: %w", err)
		}
		ms.SetStackIndex(idx)
	}

	return nil
}
