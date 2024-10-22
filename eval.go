package main

import (
	"bufio"
	"fmt"
	"os"
)

func eval() {
	fmt.Println("Markdown Parser - Enter a blankline to exit.\nAdd a filename when opening the program to parse a file instead")

	for {
		fmt.Println()
		fmt.Print("> ")

		reader := bufio.NewReader(os.Stdin)
		input, err := reader.ReadString('\n')
		if err != nil {
			fmt.Println(fmt.Errorf("error reading line: %v", err))
			os.Exit(1)
		}

		if input == "\n" {
			os.Exit(0)
		}

		scrubbedData := input[:len(input)-1]
		parser := NewParser(&scrubbedData)
		fmt.Println(parser.parse().toHtml())
	}
}
