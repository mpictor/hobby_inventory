package main

import (
	"context"
	"fmt"

	"github.com/jmoiron/sqlx"
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
	dbi, err := db.Open()
	if err != nil {
		return err
	}
	if len(args) < 2 {
		return fmt.Errorf("too few arguments - see help")
	}
	tbl := args[0]

	res, err := db.Query(sqlx.NewDb(dbi, "sqlite"), tbl, args[1:])

	if err != nil {
		return err
	}
	render.Render(res)
	return nil
}
