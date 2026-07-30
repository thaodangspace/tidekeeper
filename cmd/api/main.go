// Command api runs the Tidekeepers HTTP API.
package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/thaodangspace/tidekeepers-server/app"
	"github.com/thaodangspace/tidekeepers-server/buildinfo"
	"github.com/thaodangspace/tidekeepers-server/config"
	"github.com/thaodangspace/tidekeepers-server/observability"
)

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(args []string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("tidekeepers-api", flag.ContinueOnError)
	flags.SetOutput(stderr)
	showVersion := flags.Bool("version", false, "print build metadata and exit")
	if err := flags.Parse(args); err != nil {
		return 2
	}
	if *showVersion {
		_, _ = fmt.Fprintln(stdout, buildinfo.String())
		return 0
	}

	cfg, err := config.Load()
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "invalid configuration: %v\n", err)
		return 1
	}
	logger := observability.NewLogger(stderr, cfg.LogLevel).With(
		slog.String("service", "tidekeepers-api"),
		slog.String("version", cfg.ServiceVersion),
	)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	api, err := app.NewAPI(ctx, cfg, logger)
	if err != nil {
		logger.Error("api composition failed", slog.String("category", "configuration"))
		return 1
	}

	logger.Info("api starting", slog.String("address", cfg.HTTP.Address))
	if err := api.Run(ctx, cfg.HTTP); err != nil {
		logger.Error("api stopped", slog.String("category", "server"))
		return 1
	}
	logger.Info("api stopped")
	return 0
}
