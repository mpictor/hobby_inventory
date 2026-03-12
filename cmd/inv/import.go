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

To export all spreadsheet tabs, you can do
  soffice --convert-to csv:"Text - txt - csv (StarCalc)":44,34,UTF8,1,,0,false,true,false,false,false,-1 components.ods 
This saves each tab as a separate CSV.
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
	defer dbi.Close()

	data, err := os.ReadFile(args[1])
	if err != nil {
		return fmt.Errorf("reading csv: %w", err)
	}
	db.Verbose = verbose
	if err := tbl.ImportCSV(dbi, args[0], data); err != nil {
		return fmt.Errorf("importing csv: %w", err)
	}
	if err := tbl.Store(dbi); err != nil {
		return fmt.Errorf("storing in database: %w", err)
	}

	return nil
}
