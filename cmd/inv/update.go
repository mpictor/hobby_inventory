package main

import "github.com/peterbourgon/ff/v4"

type upd struct {
	flags *ff.FlagSet
	cmd   *ff.Command
}

var (
	updFlags = ff.NewFlagSet("update")
	updCmd   = &ff.Command{
		Name:      "update",
		ShortHelp: "update quantity of existing component in db",
		Flags:     updFlags,
	}
	compType = updFlags.String('t', "type", "", "component type")
)

func init() { rootCmd.Subcommands = append(rootCmd.Subcommands, updCmd) }
