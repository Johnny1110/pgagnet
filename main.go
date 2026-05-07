package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"

	"pg-agent/internal/config"
	"pg-agent/internal/db"
	"pg-agent/internal/safesql"
)

func main() {
	os.Exit(run())
}

func run() int {
	dbName := flag.String("db", "", "database name as defined in config.yml")
	query := flag.String("sql", "", "read-only SQL to execute")
	cfgPath := flag.String("config", defaultConfigPath(), "path to config.yml")
	flag.Usage = usage
	flag.Parse()

	if *dbName == "" || *query == "" {
		usage()
		return 2
	}

	if err := safesql.EnsureReadOnly(*query); err != nil {
		fmt.Fprintln(os.Stderr, "rejected:", err)
		return 1
	}

	finalSQL, injected := safesql.AddDefaultLimit(*query, safesql.DefaultRowLimit)
	if injected {
		fmt.Fprintf(os.Stderr, "note: no LIMIT specified; appending LIMIT %d\n", safesql.DefaultRowLimit)
	}

	cfg, err := config.Load(*cfgPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, "config:", err)
		return 1
	}

	target, ok := cfg.Databases[*dbName]
	if !ok {
		fmt.Fprintf(os.Stderr, "unknown db %q (configured: %s)\n", *dbName, strings.Join(cfg.Names(), ", "))
		return 1
	}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	if err := db.RunQuery(ctx, target, finalSQL, os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	return 0
}

func usage() {
	fmt.Fprintf(os.Stderr, `pgagent — read-only PostgreSQL query proxy

Usage:
  pgagent -db <name> -sql "<query>" [-config <path>]

Examples:
  pgagent -db leetcode -sql "SELECT * FROM users WHERE id = 12231;"
  pgagent -db school -sql "SELECT * FROM class WHERE id = 33541;"

Flags:
`)
	flag.PrintDefaults()
}

func defaultConfigPath() string {
	if v := os.Getenv("PGAGENT_CONFIG"); v != "" {
		return v
	}
	if home, err := os.UserHomeDir(); err == nil {
		return filepath.Join(home, ".pgagent", "config.yml")
	}
	return "config.yml"
}
