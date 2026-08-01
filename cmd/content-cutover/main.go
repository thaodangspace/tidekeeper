// Command content-cutover migrates recognized legacy Daily Tide projections
// and selections to exact schema-2 references in one transaction.
package main

import (
	"context"
	"encoding/hex"
	"errors"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"os"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/thaodangspace/tidekeepers-server/content"
)

func main() {
	os.Exit(run(os.Args[1:], os.LookupEnv, os.Stdout, os.Stderr))
}

type lookupFunc func(string) (string, bool)

func run(args []string, lookup lookupFunc, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("content-cutover", flag.ContinueOnError)
	flags.SetOutput(stderr)
	sourceVersion := flags.Int64("source-version", 0, "legacy source content release version")
	targetVersion := flags.Int64("target-version", 0, "published schema-2 target release version")
	apply := flags.Bool("apply", false, "apply mutations (default: dry-run)")
	if err := flags.Parse(args); err != nil {
		return 2
	}
	if flags.NArg() != 0 || *sourceVersion <= 0 || *targetVersion <= 0 {
		_, _ = fmt.Fprintln(stderr, "--source-version and --target-version are required positive integers")
		return 2
	}
	databaseURL, exists := lookup("DATABASE_URL")
	if !exists || strings.TrimSpace(databaseURL) == "" {
		_, _ = fmt.Fprintln(stderr, "DATABASE_URL is required")
		return 1
	}

	poolConfig, err := pgxpool.ParseConfig(strings.TrimSpace(databaseURL))
	if err != nil {
		_, _ = fmt.Fprintln(stderr, "invalid DATABASE_URL")
		return 1
	}
	poolConfig.MinConns = 0
	poolConfig.MaxConns = 2
	poolConfig.ConnConfig.RuntimeParams["timezone"] = "UTC"
	pool, err := pgxpool.NewWithConfig(context.Background(), poolConfig)
	if err != nil {
		_, _ = fmt.Fprintln(stderr, "connect database")
		return 1
	}
	defer pool.Close()

	result, err := content.NewCutover(pool).Run(context.Background(), content.CutoverParams{
		SourceVersion: *sourceVersion,
		TargetVersion: *targetVersion,
		Apply:         *apply,
	})
	if err != nil {
		logger := slog.New(slog.NewJSONHandler(stderr, nil))
		category := "cutover_failed"
		var cutoverErr *content.CutoverError
		if errors.As(err, &cutoverErr) {
			category = cutoverErr.Category
		}
		logger.Error("content cutover failed", slog.String("category", category), slog.String("detail", err.Error()))
		return 1
	}
	writeReport(stdout, result)
	return 0
}

func writeReport(w io.Writer, result content.CutoverResult) {
	fingerprint := "-"
	if len(result.FingerprintLabels) > 0 {
		fingerprint = strings.Join(result.FingerprintLabels, ", ")
	}
	_, _ = fmt.Fprintf(w, "source version: %d\n", result.SourceVersion)
	_, _ = fmt.Fprintf(w, "target version: %d\n", result.TargetVersion)
	_, _ = fmt.Fprintf(w, "fingerprint: %s\n", fingerprint)
	_, _ = fmt.Fprintf(w, "source catalog: %s\n", result.SourceCatalog)
	_, _ = fmt.Fprintf(w, "target checksum: %s\n", hex.EncodeToString(result.TargetChecksum[:]))
	if result.Idempotent {
		_, _ = fmt.Fprintf(w, "already complete: true\n")
		_, _ = fmt.Fprintf(w, "no changes required\n")
		return
	}
	_, _ = fmt.Fprintf(w, "tides: %d\n", result.TideCount)
	_, _ = fmt.Fprintf(w, "states: %d\n", result.StateCount)
	_, _ = fmt.Fprintf(w, "strategies: %d\n", result.StrategyCount)
	_, _ = fmt.Fprintf(w, "availability: %d\n", result.AvailabilityCount)
	_, _ = fmt.Fprintf(w, "tides to change: %d\n", result.ChangedTideCount)
	if result.Applied {
		_, _ = fmt.Fprintf(w, "tides changed: %d\n", result.ChangedTideCount)
	} else {
		_, _ = fmt.Fprintf(w, "dry-run: no changes applied\n")
	}
}
