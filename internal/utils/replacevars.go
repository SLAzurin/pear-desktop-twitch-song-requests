package utils

import "strings"

func ReplaceVars(input string, vars map[string]string) string {
	result := input
	for key, value := range vars {
		placeholder := "{" + key + "}"
		result = strings.ReplaceAll(result, placeholder, value)
	}
	return result
}
