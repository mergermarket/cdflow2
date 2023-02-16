package util

import (
	"fmt"
	"math/rand"
	"strings"
	"time"

	"github.com/logrusorgru/aurora"
	"github.com/rs/xid"

	"github.com/mergermarket/cdflow2/docker"
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

// FormatInfo colours info about what cdflow2 is doing, so you can pick it out in the output.
func FormatInfo(info string) string {
	au := aurora.NewAurora(true)
	return au.Sprintf("%s", au.Bold("cdflow2: "+info))
}

// FormatWarning colours info about what cdflow2 is doing, so you can pick it out in the output.
func FormatWarning(warning string) string {
	au := aurora.NewAurora(true)
	return au.Sprintf("%s", au.Bold("cdflow2: "+warning).BrightYellow())
}

// FormatCommand colours a command, so you can pick it out in the output.
func FormatCommand(command string) string {
	au := aurora.NewAurora(true)
	return au.Sprintf("%s %s", au.Bold("$"), au.BrightCyan(command))
}

const cacheVolumeName = "cdflow2-cache"

// GetCacheVolume returns the volume for cache at /cache (e.g. terraform providers).
func GetCacheVolume(dockerClient docker.Iface) (string, error) {
	exists, err := dockerClient.VolumeExists(cacheVolumeName)
	if err != nil {
		return "", err
	}
	if exists {
		return cacheVolumeName, nil
	}
	if _, err := dockerClient.CreateVolume(cacheVolumeName); err != nil {
		return "", err
	}
	return cacheVolumeName, nil
}
