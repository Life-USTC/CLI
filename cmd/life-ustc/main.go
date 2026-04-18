package main

import (
	"os"

	"github.com/Life-USTC/CLI/internal/cmd/root"
)

func main() {
	if err := root.NewCmdRoot().Execute(); err != nil {
		os.Exit(1)
	}
}
