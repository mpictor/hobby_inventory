package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/mpictor/hobby_inventory/pkg/db"
	"github.com/peterbourgon/ff/v4"
)

var (
	tableHelpCmd = &ff.Command{
		Name:      "table",
		ShortHelp: "HELP: lists tables",
		Usage:     "inv [-v] table",
		LongHelp:  tableHelp(),
		// Flags: ff.NewFlagSet("table"),
		Exec: func(context.Context, []string) error {
			fmt.Fprintln(os.Stderr, tableHelp())
			return nil
		},
	}
	// tableHelpCmd2 = &ff.Command{
	// 	Name:      "table",
	// 	ShortHelp: tableHelpCmd.ShortHelp,
	// 	Usage:     tableHelpCmd.Usage,
	// 	LongHelp:  tableHelpCmd.LongHelp,
	// }
	parameterHelpCmd = &ff.Command{
		Name:      "parms",
		ShortHelp: "HELP: about parameters",
		Usage:     "inv parm [-h|<table>]",
		Exec:      parmHelp,
		LongHelp:  ``, // TODO
	}
	// parameterHelpCmd2 = &ff.Command{
	// 	Name:      "parm",
	// 	ShortHelp: parameterHelpCmd.ShortHelp,
	// 	Usage:     parameterHelpCmd.Usage,
	// 	LongHelp:  parameterHelpCmd.LongHelp,
	// }
)

func init() {
	// rootCmd.Subcommands = append(rootCmd.Subcommands, tableHelpCmd, tableHelpCmd2, parameterHelpCmd, parameterHelpCmd2)
	rootCmd.Subcommands = append(rootCmd.Subcommands, tableHelpCmd, parameterHelpCmd)
}

func tableHelp() string {
	const (
		head = "All component tables:\n  "
		tail = `

In addition to the given aliases, any unique prefix of a table or
alias will also work. For example: mos, diode, opamp, opa, ...

For an exhaustive list, invoke 'inv -v table'.
`
	)
	lines := strings.Join(db.CompTblsAndAliases(verbose), "\n  ")
	if verbose {
		// skip the bit about prefixes, we're already displaying them
		return head + lines
	}
	return head + lines + tail
}

func parmHelp(_ context.Context, args []string) error {
	genericHelp := `
Parameters are key-value pairs, separated from other pairs by spaces.
The separator between a key and the corresponding value can be one of several:
'=' or '==' for exact match
':' for LIKE (value will be wrapped in %, if not already present, for DB query)
'.' or '>' for greater than
',' or '<' for less than
'.=', '>=', ',=', '<=' for greater/less than or equal

If a key is specified multiple times, the last value wins.

Examples:
fmax.=40M  to select components where the fmax column is 40,000,000 or greater
func:22    for components where the function column contains 22, which may be
             surrounded by any combination of other characters
			 
Specify a table name to see specific parameters.`
	if len(args) == 0 {
		//generic help
		fmt.Fprintln(os.Stderr, genericHelp)
		return nil
	}

	return listTableParams(args[0])
}

func listTableParams(t string) error {
	// TODO include paramAliases
	return fmt.Errorf("unimplemented")
}
