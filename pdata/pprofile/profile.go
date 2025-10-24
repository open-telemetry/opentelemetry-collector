// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package pprofile // import "go.opentelemetry.io/collector/pdata/pprofile"

import "fmt"

// switchDictionary updates the Profile, switching its indices from one
// dictionary to another.
func (ms Profile) switchDictionary(src, dst ProfilesDictionary) error {
	for i, v := range ms.AttributeIndices().All() {
		if src.AttributeTable().Len() < int(v) {
			return fmt.Errorf("invalid attribute index %d", v)
		}

		attr := src.AttributeTable().At(int(v))
		err := attr.switchDictionary(src, dst)
		if err != nil {
			return fmt.Errorf("couldn't switch dictionary for attribute %d: %w", i, err)
		}
		idx, err := SetAttribute(dst.AttributeTable(), attr)
		if err != nil {
			return fmt.Errorf("couldn't set attribute %d: %w", i, err)
		}
		ms.AttributeIndices().SetAt(i, idx)
	}

	for i, v := range ms.CommentStrindices().All() {
		if src.StringTable().Len() < int(v) {
			return fmt.Errorf("invalid comment index %d", v)
		}

		idx, err := SetString(dst.StringTable(), src.StringTable().At(int(v)))
		if err != nil {
			return fmt.Errorf("couldn't set key: %w", err)
		}
		ms.CommentStrindices().SetAt(i, idx)
	}

	for i, v := range ms.Sample().All() {
		err := v.switchDictionary(src, dst)
		if err != nil {
			return fmt.Errorf("error switching dictionary for sample %d: %w", i, err)
		}
	}

	return nil
}
