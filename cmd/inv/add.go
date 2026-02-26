package main

import (
	"context"

	"github.com/mpictor/hobby_inventory/pkg/db"
	"github.com/peterbourgon/ff/v4"
)

var addCmd = &ff.Command{
	Name:      "add",
	ShortHelp: "add new component to db",
	Exec:      execAdd,
}

func init() { rootCmd.Subcommands = append(rootCmd.Subcommands, addCmd) }

func execAdd(ctx context.Context, args []string) error {
	instance, err := db.Open()
	if err != nil {
		return err
	}
	_, err = instance.Exec("INSERT INTO ? ?", table)
	return err
}
