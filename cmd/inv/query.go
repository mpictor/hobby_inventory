package main

import (
	"context"
	"fmt"

	"github.com/mpictor/hobby_inventory/pkg/db"
	"github.com/mpictor/hobby_inventory/pkg/render"
	"github.com/peterbourgon/ff/v4"
)

var queryCmd = &ff.Command{
	Name:      "query",
	Exec:      execQuery,
	ShortHelp: "search the given component table and show matches",
	Usage:     "inv query <table> [parameters...]",
	LongHelp: `See 'inv table -h' for a list of tables, and 'inv parm -h' for parameter info.

query searches one component table and returns matches.
`,
}

func init() { rootCmd.Subcommands = append(rootCmd.Subcommands, queryCmd) }

func execQuery(ctx context.Context, args []string) error {
	db.Verbose = verbose

	dbi, err := db.Open()
	if err != nil {
		return err
	}
	defer dbi.Close()
	if len(args) < 1 {
		return fmt.Errorf("too few arguments - see help")
	}

	tbl, err := db.ParseCompTbl(args[0])
	if err != nil {
		return err
	}

	res, err := db.Query(dbi, tbl, args[1:])
	if err != nil {
		return err
	}
	if res.Len() == 0 {
		return fmt.Errorf("no results")
	}
	fmt.Printf("%d results\n", res.Len()) // TODO also query time??
	render.Verbose = verbose
	return render.Render(res, nil)
}
