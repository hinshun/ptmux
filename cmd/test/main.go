package main

import (
	"fmt"
	"os"

	"github.com/hinshun/ptmux/uiv2"
)

func main() {
	err := run()
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		os.Exit(1)
	}
}

func run() error {
	ui, err := uiv2.New()
	if err != nil {
		return err
	}

	ui.Loop()
	return nil
}
