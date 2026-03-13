package main

import (
	"os"

	"github.com/saveyourtokens/syt/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
