// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package builder

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGoTemplatesUseTabIndentation(t *testing.T) {
	t.Parallel()

	goTemplates := map[string][]byte{
		"components.go.tmpl":   componentsBytes,
		"main.go.tmpl":         mainBytes,
		"main_others.go.tmpl":  mainOthersBytes,
		"main_windows.go.tmpl": mainWindowsBytes,
		// go.mod.tmpl intentionally excluded — not a Go source file
	}

	for name, content := range goTemplates {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			for i, line := range strings.Split(string(content), "\n") {
				assert.False(t, strings.HasPrefix(line, "    "),
					"file %s line %d uses space indentation; use tabs instead:\n  %q",
					name, i+1, line)
			}
		})
	}
}
