package main

import (
	"context"
	"fmt"

	"github.com/mpictor/hobby_inventory/pkg/db"
	"github.com/mpictor/hobby_inventory/pkg/render"
	"github.com/peterbourgon/ff/v4"
)

var queryCmd = &ff.Command{
	Name: "query",
	Exec: execQuery,
}

func init() { rootCmd.Subcommands = append(rootCmd.Subcommands, queryCmd) }

func execQuery(ctx context.Context, args []string) error {
	db.Verbose = verbose

	dbi, err := db.Open()
	if err != nil {
		return err
	}
	defer dbi.Close()
	if len(args) < 2 {
		return fmt.Errorf("too few arguments - see help")
	}
	tbl := args[0]

	res, err := db.Query(dbi, tbl, args[1:])

	if err != nil {
		return err
	}

	render.Verbose = verbose
	return render.Render(res, db.TblOrder[tbl])
}
