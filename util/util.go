package util

import (
	"fmt"
	"math/rand"
	"strings"
	"time"

	"github.com/rs/xid"
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

func init() {
	rand.Seed(time.Now().UnixNano())
}

// RandomName creates a random name with a prefix so container names don't clash.
func RandomName(prefix string) string {
	return fmt.Sprintf("%s-%s", prefix, xid.New().String())
}
