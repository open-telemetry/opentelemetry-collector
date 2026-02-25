// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package pprofile // import "go.opentelemetry.io/collector/pdata/pprofile"

import "fmt"

// switchDictionary updates the Sample, switching its indices from one
// dictionary to another.
func (ms Sample) switchDictionary(src, dst ProfilesDictionary) error {
	for i, v := range ms.AttributeIndices().All() {
		if src.AttributeTable().Len() <= int(v) {
			return fmt.Errorf("invalid attribute index %d", v)
		}

		attr := src.AttributeTable().At(int(v))
		// Create a copy to avoid modifying the source AttributeTable in-place
		attrCopy := NewKeyValueAndUnit()
		attr.CopyTo(attrCopy)
		err := attrCopy.switchDictionary(src, dst)
		if err != nil {
			return fmt.Errorf("couldn't switch dictionary for attribute %d: %w", i, err)
		}
		idx, err := SetAttribute(dst.AttributeTable(), attrCopy)
		if err != nil {
			return fmt.Errorf("couldn't set attribute %d: %w", i, err)
		}
		ms.AttributeIndices().SetAt(i, idx)
	}

	if ms.LinkIndex() > 0 {
		if src.LinkTable().Len() <= int(ms.LinkIndex()) {
			return fmt.Errorf("invalid link index %d", ms.LinkIndex())
		}

		idx, err := SetLink(dst.LinkTable(), src.LinkTable().At(int(ms.LinkIndex())))
		if err != nil {
			return fmt.Errorf("couldn't set link: %w", err)
		}
		ms.SetLinkIndex(idx)
	}

	if ms.StackIndex() > 0 {
		if src.StackTable().Len() <= int(ms.StackIndex()) {
			return fmt.Errorf("invalid stack index %d", ms.StackIndex())
		}

		stack := src.StackTable().At(int(ms.StackIndex()))
		// Create a copy to avoid modifying the source StackTable in-place
		stackCopy := NewStack()
		stack.CopyTo(stackCopy)
		err := stackCopy.switchDictionary(src, dst)
		if err != nil {
			return fmt.Errorf("couldn't switch stack dictionary: %w", err)
		}

		idx, err := SetStack(dst.StackTable(), stackCopy)
		if err != nil {
			return fmt.Errorf("couldn't set stack: %w", err)
		}
		ms.SetStackIndex(idx)
	}

	return nil
}
