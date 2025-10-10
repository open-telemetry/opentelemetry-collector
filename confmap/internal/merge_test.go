package internal

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestFetchMergePaths(t *testing.T) {
	tests := []struct {
		name        string
		yaml        string
		expectErr   string
		expectedMap map[string]*MergeOptions
	}{
		{
			name: "valid tag with append mode",
			yaml: `
foo:
  bar: !mode=append [1, 2, 3]
`,
			expectedMap: map[string]*MergeOptions{
				"foo::bar": {mode: "append"},
			},
		},
		{
			name: "invalid tag with no mode",
			yaml: `
foo:
  bar: !mode= [1, 2, 3]
`,
			expectErr: "failed to validate tag",
		},
		{
			name: "valid tag syntax",
			yaml: `
foo:
  bar: !invalidtag [value]
`,
			expectErr: "failed to validate tag",
		},
		{
			name: "completely malformed tag",
			yaml: `
foo:
  bar: !@#badtag [value]
`,
			expectErr: "did not find expected whitespace or line break",
		},
		{
			name: "unknown merge mode",
			yaml: `
foo:
  bar: !mode=prepend [value]
`,
			expectErr: "failed to validate tag",
		},
		{
			name: "no tags at all",
			yaml: `
foo:
  bar:
    baz: 123
`,
			expectedMap: map[string]*MergeOptions{},
		},
		{
			name: "nested tag with valid mode",
			yaml: `
foo:
  bar:
    baz: !mode=append [1,2,3]
`,
			expectedMap: map[string]*MergeOptions{
				"foo::bar::baz": {mode: "append"},
			},
		},
		{
			name: "nested tag with invalid mode",
			yaml: `
foo:
  bar:
    baz: !nope [1,2,3]
`,
			expectErr: "failed to validate tag",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			paths, err := FetchMergePaths([]byte(tt.yaml))

			if tt.expectErr != "" {
				require.Error(t, err)
				require.ErrorContains(t, err, tt.expectErr)
			} else {
				require.NoError(t, err)
				require.Equal(t, tt.expectedMap, paths)
			}
		})
	}
}
