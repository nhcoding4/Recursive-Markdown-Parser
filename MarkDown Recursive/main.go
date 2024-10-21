package main

import (
	"os"
)

func main() {
	args := os.Args[1:]
	if len(args) == 0 {
		eval()
	} else {
		files := NewFiles(args)
		files.createFiles()
	}
}
