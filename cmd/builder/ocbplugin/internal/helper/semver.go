package helper

import (
	"errors"
	"fmt"
	"strings"

	"golang.org/x/mod/semver"
)

// ErrInvalidSemverString is returned when the plugin caller (usually OCB) passed in a bad semver string
// or an invalid MinOCBVersion.
var ErrInvalidSemverString = errors.New("invalid semver string")

func NormalizeAndValidateSemver(ver string) (string, error) {
	normalized := ver
	if !strings.HasPrefix(normalized, "v") {
		normalized = "v" + ver
	}
	if !semver.IsValid(normalized) {
		return ver, fmt.Errorf("%w: %s (attempted to normalize to %s)", ErrInvalidSemverString, ver, normalized)
	}
	return normalized, nil
}
