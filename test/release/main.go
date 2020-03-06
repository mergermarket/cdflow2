package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"os"
)

func main() {
	if len(os.Args) == 2 && os.Args[1] == "requirements" {
		// requirements is a way for the release container to communciate its requirements to the
		// config container
		if err := json.NewEncoder(os.Stdout).Encode(map[string]interface{}{
			"env": []string{"FOO", "BAR"},
		}); err != nil {
			log.Panicln("error encoding requirements:", err)
		}
		return
	}
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
	if err := ioutil.WriteFile("/release-metadata.json", encoded, 0644); err != nil {
		log.Fatalln("error writing release metadata:", err)
	}
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
