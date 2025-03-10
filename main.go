package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/urfave/cli/v2"
)

func main() {
	app := &cli.App{
		// --- CLI help info
		Name:      "aika",
		Version:   "0.2.0",
		Usage:     "your anime waifu assistant",
		UsageText: "aika <discord> <options>",
		Authors: []*cli.Author{
			{
				Name:  "Aika",
				Email: "aika@lystic.dev",
			},
			{
				Name:  "Kegan Hollern",
				Email: "keganhollern@gmail.com",
			},
		},
		// -- enable tab completion in bash
		EnableBashCompletion: true,
		// -- entrypoint
		Action: func(ctx *cli.Context) error {
			fmt.Println("entrypoint: ", ctx.Args().Slice())
			return nil
		},
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := app.RunContext(ctx, os.Args); err != nil {
		slog.Error("failed to run app", slog.Any("error", err))
		os.Exit(1)
	}
}
