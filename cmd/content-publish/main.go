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
	"github.com/thaodangspace/tidekeepers-server/keeper"
)

func main() {
	os.Exit(run(os.Args[1:], os.LookupEnv, os.Stdout, os.Stderr))
}

type lookupFunc func(string) (string, bool)

func run(args []string, lookup lookupFunc, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("content-publish", flag.ContinueOnError)
	flags.SetOutput(stderr)
	release := flags.String("release", "", "approved catalog release to publish (keepers-v1 or sectors-v2)")
	if err := flags.Parse(args); err != nil {
		return 2
	}
	if flags.NArg() != 0 || (*release != "keepers-v1" && *release != "sectors-v2") {
		_, _ = fmt.Fprintln(stderr, "--release=keepers-v1 or --release=sectors-v2 is required")
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

	catalog := keeper.CatalogV1()
	if *release == "sectors-v2" {
		catalog = keeper.CatalogV2()
	}
	result, err := keeper.NewPublisher(pool).Publish(context.Background(), catalog)
	if err != nil {
		logger := slog.New(slog.NewJSONHandler(stderr, nil))
		category := "publication_failed"
		if errors.Is(err, keeper.ErrReleaseConflict) {
			category = "release_conflict"
		}
		logger.Error("catalog publication failed", slog.String("category", category))
		return 1
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
	return 0
}
