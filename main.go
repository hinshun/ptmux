package main

import (
	"context"
	"fmt"
	"os"
	"syscall"

	"github.com/hinshun/ptmux/command"
	"github.com/rs/zerolog"
)

func init() {
	// UNIX Time is faster and smaller than most timestamps. If you set
	// zerolog.TimeFieldFormat to an empty string, logs will write with UNIX
	// time.
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnixMs
}

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	ih := command.NewInterruptHandler(cancel, syscall.SIGINT, syscall.SIGTERM)
	defer ih.Close()

	rootLogger := zerolog.New(zerolog.ConsoleWriter{Out: os.Stderr}).With().Timestamp().Logger()
	logger := &rootLogger
	ctx = logger.WithContext(ctx)

	app := command.App()
	if err := app.RunContext(ctx, os.Args); err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		os.Exit(1)
	}
}
