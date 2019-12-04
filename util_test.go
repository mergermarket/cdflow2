package main

import (
	"log"
	"os"
)

func getConfig(name string) string {
	value := os.Getenv(name)
	if value == "" {
		log.Fatalf("environment variable %v not set - did you run ./test.sh?", name)
	}
	return value
}
