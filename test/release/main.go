package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
)

func main() {
	fmt.Println("message to stdout from release")
	fmt.Fprintln(os.Stderr, "message to stderr from release")
	fmt.Fprintf(os.Stderr, "docker status: %v\n", dockerStatus())
	encoded, err := json.Marshal(map[string]string{
		"release_var_from_env": "release value from env",
		// test environment variables passed by default
		"version_from_defaults":   os.Getenv("VERSION"),
		"team_from_defaults":      os.Getenv("TEAM"),
		"component_from_defaults": os.Getenv("COMPONENT"),
		"commit_from_defaults":    os.Getenv("COMMIT"),
		// test environment variable passed through from config
		"test_from_config": os.Getenv("TEST_VERSION"),
	})
	if err != nil {
		log.Fatalln("error encoding json:", err)
	}
	fmt.Println(string(encoded))
}

func dockerStatus() string {
	client := http.Client{
		Transport: &http.Transport{
			DialContext: func(_ context.Context, _, _ string) (net.Conn, error) {
				return net.Dial("unix", "/var/run/docker.sock")
			},
		},
	}
	response, err := client.Get("http://unix/_ping")
	if err != nil {
		log.Fatalln("error pinging docker:", err)
	}
	var buffer bytes.Buffer
	io.Copy(&buffer, response.Body)
	return buffer.String()
}
