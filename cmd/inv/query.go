package main

import "github.com/peterbourgon/ff/v4"

var queryCmd = &ff.Command{
	Name: "query",
}

func init() { rootCmd.Subcommands = append(rootCmd.Subcommands, queryCmd) }
