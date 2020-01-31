package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
)

func main() {
	fmt.Println("message to stdout")
	fmt.Fprintln(os.Stderr, "message to stderr")
	encoded, err := json.Marshal(map[string]string{
		"release_var_from_env": "release value from env",
	})
	if err != nil {
		log.Fatalln("error encoding json:", err)
	}
	fmt.Println(string(encoded))
}
