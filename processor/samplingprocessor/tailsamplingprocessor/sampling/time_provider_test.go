package sampling

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestTimeProvider(t *testing.T) {
	clock := MonotonicClock{}
	assert.Greater(t, clock.getCurSecond(), int64(0))
}
