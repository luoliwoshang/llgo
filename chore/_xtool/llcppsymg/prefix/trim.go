package prefix

import (
	"fmt"
	"regexp"
	"strings"
)

func TrimPrefix(org, prefix, repl string) (string, error) {
	exp, err := regexp.Compile(prefix)
	if err != nil {
		if strings.HasPrefix(org, prefix) {
			return strings.ReplaceAll(org, prefix, repl), nil
		}
		return org, fmt.Errorf("trim prefix fail")
	}
	if !exp.MatchString(org) {
		return org, fmt.Errorf("trim prefix fail")
	}
	return exp.ReplaceAllString(org, repl), nil
}

func TrimPrefixes(org string, prefixes []string) string {
	for _, prefix := range prefixes {
		result, err := TrimPrefix(org, prefix, "")
		if err == nil {
			return result
		}
	}
	return org
}

func ReplacePrefixes(org string, prefixes, repls []string) string {
	for i, prefix := range prefixes {
		repl := ""
		if i < len(repls) {
			repl = repls[i]
		}
		result, err := TrimPrefix(org, prefix, repl)
		if err == nil {
			return result
		}
	}
	return org
}
