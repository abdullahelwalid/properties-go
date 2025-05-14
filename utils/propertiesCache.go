package utils

import (
	"fmt"
	"sort"
	"strings"
)


func GeneratePropertiesCacheKey(params map[string]string) string {
	// Sort keys for consistent cache keys
	var keys []string
	for k := range params {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	
	// Build key string
	var keyParts []string
	for _, k := range keys {
		if params[k] != "" {
			keyParts = append(keyParts, fmt.Sprintf("%s:%s", k, params[k]))
		}
	}
	
	if len(keyParts) == 0 {
		return "properties:all"
	}
	
	return "properties:" + strings.Join(keyParts, ":")
}
