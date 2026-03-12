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

	ic := &initCmd{}
	icFlags := ff.NewFlagSet("init")
	icFlags.BoolVar(&ic.force, 'f', "force", "delete db if it exists")

	rootCmd.Subcommands = append(rootCmd.Subcommands, &ff.Command{
		Name:      "init",
		ShortHelp: "initialize new database",
		Usage:     "init [-f]",
		Exec:      ic.Do,
		Flags:     icFlags,
	})

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

type initCmd struct {
	force bool
}

func (i *initCmd) Do(ctx context.Context, args []string) error {
	_, err := db.Create(i.force)
	return err
}
