package envvar

import "regexp"

const ValidationPattern = `^[a-zA-Z_][a-zA-Z0-9_]*$`

var ValidationRegexp = regexp.MustCompile(ValidationPattern)
