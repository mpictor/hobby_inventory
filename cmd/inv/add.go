package main

import (
	"context"
	"fmt"

	"github.com/mpictor/hobby_inventory/pkg/db"
	"github.com/peterbourgon/ff/v4"
)

var addCmd = &ff.Command{
	Name:      "add",
	ShortHelp: "add new component to db",
	Usage:     "inv add <table> [<parameter=value>...]",
	Exec:      execAdd,
	LongHelp: `
example:
  inv add ttl pn=F74AHCTLS999UB qty=55 fmax=1.21JigoHertz pkg=SO-8765 loc=nowhere
adds to the ttl table using parameters pn, qty, fmax, pkg, and loc.

Use 'inv help tables' to list tables, or 'inv help parameters <table>'
to list parameters existing in given table.
`,
}

func init() { rootCmd.Subcommands = append(rootCmd.Subcommands, addCmd) }

func execAdd(ctx context.Context, args []string) error {
	dbi, err := db.Open()
	if err != nil {
		return err
	}
	if len(args) < 2 {
		return fmt.Errorf("too few arguments - see help")
	}
	defer dbi.Close()

	tbl := args[0]
	vals, err := db.ParamsToRow(dbi, tbl, args[1:])
	if err != nil {
		return err
	}
	err = vals.Insert(dbi)
	return err
}
