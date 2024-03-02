package scraperhelper

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/component"
)

type scraperTypeTestCase struct {
	componentId component.ID
	shouldError bool
}

func TestScraperTypes(t *testing.T) {
	testCases := []scraperTypeTestCase{
		{
			componentId: component.NewIDWithName(component.NoType, "testname"),
			shouldError: true,
		},
		{
			componentId: component.NewIDWithName("123abc", "testname"),
			shouldError: true,
		},
		{
			componentId: component.NewIDWithName("sample_Component_0", ""),
			shouldError: false,
		},
		{
			componentId: component.NewIDWithName(component.Type(strings.Repeat("a", 257)), strings.Repeat("testname", 257)),
			shouldError: false,
		},
	}
	for _, tc := range testCases {
		_, err := WithID(tc.componentId)
		if tc.shouldError {
			require.Error(t, err)
		} else {
			require.NoError(t, err)
		}
	}
}
