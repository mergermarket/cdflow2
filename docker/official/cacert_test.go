package official_test

import (
	"bytes"
	"os"
	"runtime"
	"strings"
	"testing"

	"github.com/mergermarket/cdflow2/docker"
	"github.com/mergermarket/cdflow2/docker/official"
)

func TestRunInjectsCACert(t *testing.T) {
	pemFile := t.TempDir() + "/ca.pem"
	if err := osWriteFile(pemFile, "test-pem-content\n"); err != nil {
		t.Fatal(err)
	}
	t.Setenv(official.CACertEnvVar, pemFile)

	dockerClient, err := official.NewClient()
	if err != nil {
		t.Fatal(err)
	}
	image := "alpine:latest"
	if err := dockerClient.EnsureImage(image, nil); err != nil {
		t.Fatal(err)
	}

	var out, errOut bytes.Buffer
	if err := dockerClient.Run(&docker.RunOptions{
		Image:        image,
		OutputStream: &out,
		ErrorStream:  &errOut,
		Cmd:          []string{"/bin/sh", "-c", `echo $SSL_CERT_FILE; cat $SSL_CERT_FILE`},
		NamePrefix:   "cdflow2-test-cacert",
	}); err != nil {
		t.Fatal(err, errOut.String())
	}
	if got := out.String(); !strings.Contains(got, "/etc/cdflow2/ca.pem\ntest-pem-content") {
		t.Fatalf("unexpected output: %q", got)
	}
}

func osWriteFile(path, content string) error { return os.WriteFile(path, []byte(content), 0600) }

func TestRunInjectsCACertFromKeychain(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("macOS only")
	}
	t.Setenv(official.CACertEnvVar, "")
	t.Setenv(official.CACertKeychainEnvVar, "true")

	dockerClient, err := official.NewClient()
	if err != nil {
		t.Fatal(err)
	}
	image := "alpine:latest"
	if err := dockerClient.EnsureImage(image, nil); err != nil {
		t.Fatal(err)
	}
	var out, errOut bytes.Buffer
	if err := dockerClient.Run(&docker.RunOptions{
		Image:        image,
		OutputStream: &out,
		ErrorStream:  &errOut,
		Cmd:          []string{"/bin/sh", "-c", `grep -c 'BEGIN CERTIFICATE' $SSL_CERT_FILE`},
		NamePrefix:   "cdflow2-test-cacert",
	}); err != nil {
		t.Fatal(err, errOut.String())
	}
	if strings.TrimSpace(out.String()) == "0" {
		t.Fatalf("no certificates injected: %q", out.String())
	}
}
