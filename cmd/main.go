package main

import (
	"os"

	"github.com/go-toolsmith/minformat"
)

func main() {
	if len(os.Args) != 2 {
		panic("needs 1 argument: file to process")
	}

	filename := os.Args[1]

	b, err := os.ReadFile(filename)
	if err != nil {
		panic(err)
	}

	res, err := minformat.Source(b)
	if err != nil {
		panic(err)
	}

	if _, err := os.Stdout.Write(res); err != nil {
		panic(err)
	}
}
