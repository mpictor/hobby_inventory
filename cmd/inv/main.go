package main

import (
	"context"
	"fmt"
	"os"

	"github.com/mpictor/hobby_inventory/pkg/db"
	"github.com/peterbourgon/ff/v4"
	"github.com/peterbourgon/ff/v4/ffhelp"
)

// example commands
//
// query:
// inv q t:bip v:234 ft:456 pol:n
//   -- searches table:transistors_bipolar for v>=234 and ft>=456, polarity npn
//   -- spaces delimit k-v pairs; key is single char, separated from val by (optional?) colon
//   -- can't easily use > or < because shell
//   -- value can be first letters as long as unambiguous. case insensitive
//
// table info:
// inv h t:trans
//   - prints info about columns in given table
//
// interactive add:
// inv add \n
//
// import/export
// inv import <file> // import sql or csv
// inv export <file> // export, sql only?
//
// web view
// inv web\n
//  now serving... addr:
//  http://localhost:5678

// global config vars
var (
	verbose bool
	dryRun  bool
)

var rootCmd = &ff.Command{
	Name: "inventory",
}

func main() {
	// ff.NewFlagSet
	// fs := flag.NewFlagSet("myprogram", flag.ContinueOnError)
	// var (
	// listenAddr = fs.String('l', "listen", "localhost:8080", "listen address")
	// refresh    = fs.Duration('r', "refresh", 15*time.Second, "refresh interval")
	// debug      = fs.Bool('d', "debug", "log debug information")
	// _          =
	// )

	rootFlags := ff.NewFlagSet("inventory")
	rootFlags.String('c', "config", "", "config file (optional)")
	rootFlags.BoolVar(&verbose, 'v', "verbose", "log debug information")
	rootFlags.BoolVar(&dryRun, 'd', "dryrun", "print actions but make no db calls")

	rootCmd.Flags = rootFlags
	// }

	initCmd := &ff.Command{
		Name:      "init",
		ShortHelp: "initialize new database (NOTE: flags other than --verbose and --dryrun are ignored)",
		Usage:     "init",
		Exec:      doInitCmd,
	}
	rootCmd.Subcommands = append(rootCmd.Subcommands, initCmd)

	for _, c := range rootCmd.Subcommands {
		if c.Flags == nil {
			continue
		}
		if fs, ok := c.Flags.(*ff.FlagSet); ok {
			fs.SetParent(rootFlags)
		}
	}

	// ff.Parse(rootFlags, os.Args[1:],
	// 	// ff.WithEnvVarPrefix("MY_PROGRAM"),
	// 	ff.WithConfigFileFlag("config"),
	// 	ff.WithConfigFileParser(ff.PlainParser),
	// )

	if err := rootCmd.ParseAndRun(context.Background(), os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", ffhelp.Command(rootCmd))
		fmt.Fprintf(os.Stderr, "error: %s\n", err)
		os.Exit(0)
	}
}

func doInitCmd(ctx context.Context, args []string) error {
	// TODO check that no flags are specified
	_, err := db.Create()
	return err
}
