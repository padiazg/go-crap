package utils

import (
	"regexp"
	"strings"
)

func BuildExcludeRegex(exclude []string) (*regexp.Regexp, error) {
	if len(exclude) == 0 {
		return nil, nil
	}
	parts := make([]string, len(exclude))
	copy(parts, exclude)

	return regexp.Compile(strings.Join(parts, "|"))
}

func MatchExclude(re *regexp.Regexp, filePath string) bool {
	if re == nil {
		return false
	}
	return re.MatchString(filePath)
}
