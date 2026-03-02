package main

import (
	"context"
	"fmt"
	"os"

	"github.com/mpictor/hobby_inventory/pkg/db"
	"github.com/peterbourgon/ff/v4"
)

var importCmd = &ff.Command{
	Name:      "import",
	ShortHelp: "bulk add components to db from CSV",
	Usage:     "inv import <table> <file.csv>",
	Exec:      execImport,
	LongHelp: `
example:
  inv import ttl path/to/ttl.csv
adds to the ttl table from given csv. columns must match expectations!
`,
}

func init() { rootCmd.Subcommands = append(rootCmd.Subcommands, importCmd) }

func execImport(ctx context.Context, args []string) error {
	if len(args) != 2 {
		return fmt.Errorf("requires two args: table and csv path")
	}
	tbl := db.GetTbl(args[0])
	if tbl == nil {
		return fmt.Errorf("%s is not a known table", args[0])
	}
	dbi, err := db.Open()
	if err != nil {
		return fmt.Errorf("opening database: %w", err)
	}
	data, err := os.ReadFile(args[1])
	if err != nil {
		return fmt.Errorf("reading csv: %w", err)
	}
	if err := tbl.ImportCSV(dbi, data); err != nil {
		return fmt.Errorf("importing csv: %w", err)
	}
	if err := tbl.Store(dbi); err != nil {
		return fmt.Errorf("storing in database: %w", err)
	}

	// if err := tbl.ImportCSV(db, []byte(td.csv)); err != nil {
	return nil
}
