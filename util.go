package main

import (
	"math/rand"
	"os"
	"time"
)

func tempdir() (string, error) {
	// not using the native TempDir since the directory is not configured to be sharable with docker
	// containers on OSX by default :-(
	rand.Seed(time.Now().UnixNano())
	var letters = []rune("abcdefghijklmnopqrstuvwxyz")
	b := make([]rune, 10)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	dir := "/tmp/cdflow2-test-" + string(b)
	if err := os.Mkdir(dir, 0777); err != nil {
		return "", err
	}
	return dir, nil
}
