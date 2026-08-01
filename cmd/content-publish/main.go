// Command content-publish publishes developer-approved static game content.
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
	"github.com/thaodangspace/tidekeepers-server/keeper"
)

func main() {
	os.Exit(run(os.Args[1:], os.LookupEnv, os.Stdout, os.Stderr))
}

type lookupFunc func(string) (string, bool)

func run(args []string, lookup lookupFunc, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("content-publish", flag.ContinueOnError)
	flags.SetOutput(stderr)
	release := flags.String("release", "", "approved release to publish (keepers-v1, sectors-v2, baseline-v1, or baseline-v2)")
	if err := flags.Parse(args); err != nil {
		return 2
	}
	if flags.NArg() != 0 || !validRelease(*release) {
		_, _ = fmt.Fprintln(stderr, "--release=keepers-v1, sectors-v2, baseline-v1, or baseline-v2 is required")
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

	switch *release {
	case "keepers-v1", "sectors-v2":
		catalog := keeper.CatalogV1()
		if *release == "sectors-v2" {
			catalog = keeper.CatalogV2()
		}
		result, err := keeper.NewPublisher(pool).Publish(context.Background(), catalog)
		if err != nil {
			return fail(stderr, "catalog publication failed", errors.Is(err, keeper.ErrReleaseConflict))
		}
		_, _ = fmt.Fprintf(stdout,
			"catalog release %d published (idempotent=%t checksum=%s definitions=%d mappings=%d components=%d)\n",
			result.Version,
			result.Idempotent,
			hex.EncodeToString(result.Checksum[:]),
			result.DefinitionCount,
			result.MappingCount,
			result.ComponentCount,
		)
	case "baseline-v1", "baseline-v2":
		candidate := content.CompleteReleaseV1(content.BaselineV1Version)
		if *release == "baseline-v2" {
			candidate = content.CompleteReleaseV2(content.BaselineV2Version)
		}
		result, err := content.NewPublisher(pool).Publish(context.Background(), *candidate)
		if err != nil {
			return fail(stderr, "content publication failed", errors.Is(err, content.ErrReleaseConflict))
		}
		_, _ = fmt.Fprintf(stdout,
			"content release %d published (idempotent=%t checksum=%s modifiers=%d objectives=%d game_rule_sets=%d)\n",
			result.Version,
			result.Idempotent,
			hex.EncodeToString(result.Checksum[:]),
			result.ModifierCount,
			result.ObjectiveCount,
			result.GameRuleSetCount,
		)
	}
	return 0
}

func validRelease(release string) bool {
	switch release {
	case "keepers-v1", "sectors-v2", "baseline-v1", "baseline-v2":
		return true
	}
	return false
}

func fail(stderr io.Writer, message string, conflict bool) int {
	logger := slog.New(slog.NewJSONHandler(stderr, nil))
	category := "publication_failed"
	if conflict {
		category = "release_conflict"
	}
	logger.Error(message, slog.String("category", category))
	return 1
}
