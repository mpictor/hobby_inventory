package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	fp "path/filepath"
	"strings"

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
  inv import all path/to/csv-dir/
attempts to determine which table each csv should import into, then imports.

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
	if args[0] == "all" {
		return importMany(args[1])
	} else {
		return importOne(args[0], args[1])
	}
}

func importOne(table, path string) error {
	ttyp, err := db.ParseCompTbl(table)
	if err != nil {
		return err
	}

	tbl := db.GetTbl(ttyp)
	if tbl == nil {
		return fmt.Errorf("%s (%s) is not a known table", table, ttyp)
	}
	dbi, err := db.Open()
	if err != nil {
		return fmt.Errorf("opening database: %w", err)
	}
	defer dbi.Close()

	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("reading csv: %w", err)
	}
	db.Verbose = verbose
	if err := tbl.ImportCSV(dbi, table, data); err != nil {
		return fmt.Errorf("importing csv: %w", err)
	}
	if err := tbl.Store(dbi); err != nil {
		return fmt.Errorf("storing in database: %w", err)
	}

	return nil
}

func importMany(dir string) error {
	csvs, err := fp.Glob(fp.Join(dir, "*.csv"))
	if err != nil {
		return err
	}
	if len(csvs) == 0 {
		return errors.New("no csv files found")
	}
	for _, csv := range csvs {
		tbl := strings.TrimPrefix(fp.Base(csv), "components-")
		tbl = strings.ToLower(strings.TrimSuffix(tbl, ".csv"))
		if tbl == "util" {
			//skip
			continue
		}
		if strings.Contains(tbl, "cmos") {
			tbl = "cmos"
		}
		if strings.Contains(tbl, "74xx") {
			tbl = "ttl"
		}
		_, err := db.ParseCompTbl(tbl)
		if err != nil {
			return err
		}
		if err := importOne(tbl, csv); err != nil {
			return fmt.Errorf("loading %s %s: %w", tbl, csv, err)
		}
	}
	return nil
}
