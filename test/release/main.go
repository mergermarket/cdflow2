package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
)

func main() {
	fmt.Println("message to stdout from release")
	fmt.Fprintln(os.Stderr, "message to stderr from release")
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
