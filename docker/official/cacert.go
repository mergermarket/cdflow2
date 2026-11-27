package official

import (
	"archive/tar"
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"sync"
)

const (
	// CACertEnvVar names the host environment variable holding the path to a PEM CA bundle to inject into containers.
	CACertEnvVar = "CDFLOW2_CA_CERT"
	// CACertKeychainEnvVar enables (macOS only) exporting the CA bundle from the keychains when CDFLOW2_CA_CERT is not set.
	CACertKeychainEnvVar = "CDFLOW2_CA_CERT_FROM_KEYCHAIN"
	// caCertDir and caCertFile are where the bundle is placed inside containers.
	caCertDir  = "/etc/cdflow2"
	caCertFile = "ca.pem"
	caCertPath = caCertDir + "/" + caCertFile
)

// caCertEnvNames are the variables commonly used by tools to find a CA bundle.
var caCertEnvNames = []string{
	"SSL_CERT_FILE",
	"AWS_CA_BUNDLE",
	"REQUESTS_CA_BUNDLE",
	"CURL_CA_BUNDLE",
	"NODE_EXTRA_CA_CERTS",
	"GIT_SSL_CAINFO",
}

// caCertFromEnv returns the CA bundle to inject, or nil if none is configured.
// An explicit CDFLOW2_CA_CERT path takes precedence over the keychain export.
func caCertFromEnv() ([]byte, error) {
	if path := os.Getenv(CACertEnvVar); path != "" {
		pem, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("could not read %s (%s): %w", CACertEnvVar, path, err)
		}
		return pem, nil
	}
	if enabled, _ := strconv.ParseBool(os.Getenv(CACertKeychainEnvVar)); enabled {
		return keychainCACerts()
	}
	return nil, nil
}

var (
	keychainOnce sync.Once
	keychainPEM  []byte
	keychainErr  error
)

// keychainCACerts exports all certificates from the macOS keychains as PEM (cached for the process).
func keychainCACerts() ([]byte, error) {
	keychainOnce.Do(func() {
		if runtime.GOOS != "darwin" {
			keychainErr = fmt.Errorf("%s is only supported on macOS", CACertKeychainEnvVar)
			return
		}
		keychains := []string{
			"/Library/Keychains/System.keychain",
			"/System/Library/Keychains/SystemRootCertificates.keychain",
		}
		if home, err := os.UserHomeDir(); err == nil {
			login := filepath.Join(home, "Library/Keychains/login.keychain-db")
			if _, err := os.Stat(login); err == nil {
				keychains = append(keychains, login)
			}
		}
		args := append([]string{"find-certificate", "-a", "-p"}, keychains...)
		var stderr bytes.Buffer
		cmd := exec.Command("/usr/bin/security", args...)
		cmd.Stderr = &stderr
		out, err := cmd.Output()
		if err != nil {
			keychainErr = fmt.Errorf("could not export certificates from keychain: %w: %s", err, stderr.String())
			return
		}
		if len(out) == 0 {
			keychainErr = fmt.Errorf("no certificates found in keychains")
			return
		}
		keychainPEM = out
	})
	return keychainPEM, keychainErr
}

// caCertEnv returns env entries pointing common tools at the injected bundle.
func caCertEnv() []string {
	env := make([]string, 0, len(caCertEnvNames))
	for _, name := range caCertEnvNames {
		env = append(env, name+"="+caCertPath)
	}
	return env
}

// caCertTar builds a tar stream (for CopyToContainer at "/") containing the bundle.
func caCertTar(pem []byte) (*bytes.Buffer, error) {
	var buf bytes.Buffer
	w := tar.NewWriter(&buf)
	if err := w.WriteHeader(&tar.Header{Typeflag: tar.TypeDir, Name: caCertDir + "/", Mode: 0755}); err != nil {
		return nil, err
	}
	if err := w.WriteHeader(&tar.Header{Typeflag: tar.TypeReg, Name: caCertDir + "/" + caCertFile, Mode: 0644, Size: int64(len(pem))}); err != nil {
		return nil, err
	}
	if _, err := w.Write(pem); err != nil {
		return nil, err
	}
	if err := w.Close(); err != nil {
		return nil, err
	}
	return &buf, nil
}
