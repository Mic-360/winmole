package main

import (
	"fmt"
	"os"

	"github.com/mic-360/wimo/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}
}
