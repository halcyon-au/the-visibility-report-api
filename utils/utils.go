package utils

import (
	"os"
	"regexp"
	"strings"
)

var MODES = map[string]bool{ // Not sure if this is the best way to do this :/
	"local":       true,
	"development": true,
	"production":  true,
}

func RegexOriginWrapper(regex string) func(string) (bool, error) {
	allowOrigin := func(origin string) (bool, error) {
		return regexp.MatchString(regex, origin)
	}
	return allowOrigin
}

func RetrieveOrigins() func(string) (bool, error) {
	mode, found := os.LookupEnv("mode")
	if !found || mode == "" {
		panic("mode environment variable is not defined")
	}
	if _, exists := MODES[strings.ToLower(mode)]; !exists {
		panic("mode is not local, development or production")
	}
	switch mode { // ^https:\/\/labstack\.(net|com)$
	case "local":
		return RegexOriginWrapper(`^http:\/\/localhost:[0-9]+$`)
	case "development":
		return RegexOriginWrapper(`^https?:\/\/halcyon.*$`)
	default:
		return RegexOriginWrapper(`^https?:\/\/halcyon.*$`)
	}
}
