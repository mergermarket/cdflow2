package util

import (
	"strings"
)

// GetEnv takes the environment as a slice of strings (as returned by os.Environ) and returns it as a map.
func GetEnv(env []string) map[string]string {
	result := make(map[string]string)
	for _, e := range env {
		pair := strings.SplitN(e, "=", 2)
		result[pair[0]] = pair[1]
	}
	return result
}
