package main

import (
	"fmt"
	"os"
)

func main() {
	if len(os.Args) > 2 && os.Args[1] == "filesystem" {
		os.Exit(0)
	} else if len(os.Args) > 2 && os.Args[1] == "image" {
		os.Exit(0)
	} else {
		fmt.Println("message to stdout")
	}

}

// func parseArgs(args []string) (map[string]string, error) {
// 	result := make(map[string]string)
// 	for _, arg := range args {
// 		if strings.Contains(arg, "=") {
// 			kv := strings.SplitN(arg, "=", 2)
// 			if len(kv) != 2 {
// 				return nil, os.ErrInvalid
// 			}
// 			result[kv[0]] = kv[1]
// 		} else {
// 			return nil, os.ErrInvalid
// 		}
// 	}
// 	return result, nil
// }
