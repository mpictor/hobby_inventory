package main

import (
	"context"
	"fmt"
	"math"
	"strings"

	"github.com/davidkleiven/gononlin/nonlin"
	"github.com/dustin/go-humanize"
	"github.com/mpictor/hobby_inventory/pkg/db"
	"github.com/mpictor/hobby_inventory/pkg/util"
	"github.com/peterbourgon/ff/v4"
)

var calcCmd = &ff.Command{
	Name:      "calc",
	ShortHelp: "perform common calculation",
	Usage:     "inv calc <mode> <data>",
	Exec:      execCalc,
	LongHelp: `
inv calc <mode> <data>
where mode is one of rw, TODO
example:
  inv calc rw r=3.3 w=55
calculates voltage drop and current for given resistance and wattage
`,
}

// TODO tolerance? e.g. tol 5 10k -> displays 10k +- 5%, but also x+5%=10k and y-5%=10k (e.g min/max common value within % of given value)

func init() { rootCmd.Subcommands = append(rootCmd.Subcommands, calcCmd) }

var errNumArgs = fmt.Errorf("wrong number of args")

func execCalc(ctx context.Context, args []string) error {
	if len(args) < 1 {
		return errNumArgs
	}
	switch args[0] {
	case "rw":
		rw, err := resistorWattageDKnonlin(args)
		// rw, err := resistorWattage(args)
		if err != nil {
			return err
		}
		fmt.Println(rw)
		return nil
	default:
		return fmt.Errorf("unknown mode in %v", args)
	}
}

// func resistorWattageOrig(args []string) (string, error) {
// 	args = args[1:]
// 	if len(args) != 2 {
// 		return "", errNumArgs
// 	}
// 	pm := db.ToParamMap(args, false)
// 	// rs := pm["r"]
// 	// TODO si notation?
// 	r, _, err := humanize.ParseSI(pm["r"])
// 	// r, err := strconv.ParseFloat(pm["r"], 64)
// 	if err != nil {
// 		return "", fmt.Errorf("param r: %w", err)
// 	}
// 	w, _, err := humanize.ParseSI(pm["p"])
// 	// w, err := strconv.ParseFloat(pm["p"], 64)
// 	if err != nil {
// 		return "", fmt.Errorf("param p: %w", err)
// 	}
// 	// if !ok {
// 	// 	return fmt.Errorf("require params r= and w=")
// 	// }
// 	// ws := pm["w"]
// 	// if len(w)==0||len(r)==0 {
// 	// 	return fmt.Errorf("require params r= and w=")
// 	// }
// 	// w = i^2 * r
// 	// i = sqrt(w/r)
// 	i := math.Sqrt(w / r)
// 	v := i * r
// 	// v= i*r
// 	res := fmt.Sprintf("v=%f\ni=%f\n", v, i)
// 	return res, nil
// }

func outputPretty(X []float64, meta [][2]string) (string, error) {
	if len(X) != len(meta) {
		return "", fmt.Errorf("length mismatch: %d vs %d", len(X), len(meta))
	}
	outs := make([]string, len(X))
	for i, x := range X {
		outs[i] = fmt.Sprintf("%s=%s", meta[i][0], util.SI(x, meta[i][1], 3))
	}
	return strings.Join(outs, ", "), nil
}

const (
	// x[0]:v, x[1]:i, x[2]: r, x[3]: p
	v = iota
	i
	r
	p
)

func varLoc(k string) int {
	switch k {
	case "v":
		return v
	case "i":
		return i
	case "r":
		return r
	case "p":
		return p
	}
	return -1
}

func ohmsLawDKnonlin(vals map[string]float64) (nonlin.Result, error) {
	problem := nonlin.Problem{
		F: func(out, vars []float64) {
			// i=v/r   --> 0 = v / r - i
			out[0] = vars[v]/vars[r] - vars[i]
			// p=i^2*r --> 0 = i^2 * r - p
			out[1] = math.Pow(vars[i], 2)*vars[r] - vars[p]
			i := 2
			for k, v := range vals {
				idx := varLoc(k)
				out[i] = vars[idx] - v
				i++
			}
		},
	}
	solver := nonlin.NewtonKrylov{
		StepSize: 1e-2,
		Tol:      1e-7,
	}
	x0 := []float64{1, 1, 1, 1}
	for k, v := range vals {
		x0[varLoc(k)] = v
	}
	return solver.Solve(problem, x0)
}

// 10x faster than gonum but seems tempermental? need more testing
//
// Root: (v,i,r,p)=(707.106781 , 707.106781 µ, 1 M, 500 m)
// Function value: (-108.420217 z, 59.812266 p, 0 , 0 )
func resistorWattageDKnonlin(args []string) (string, error) {
	args = args[1:]
	if len(args) != 2 {
		return "", errNumArgs
	}
	pm := db.ToParamMap(args, false, false)
	vals := make(map[string]float64, len(pm))
	for k, v := range pm {
		val, unit, err := humanize.ParseSI(v.Val)
		if err != nil {
			return "", err
		}
		if unit != "" {
			return "", fmt.Errorf("unhandled unit %s", unit)
		}
		vals[k] = val
	}
	res, err := ohmsLawDKnonlin(vals)
	if err != nil {
		return "", err
	}
	meta := [][2]string{{"v", "V"}, {"i", "A"}, {"r", "Ω"}, {"p", "W"}}
	out, err := outputPretty(res.X, meta)
	if err != nil {
		return "", err
	}

	solF := make([]string, 0, len(res.F))
	for _, f := range res.F {
		solF = append(solF, util.SI(f, "", 5))
	}
	rs := fmt.Sprintf("%s\nremainder: %s\n", out, strings.Join(solF, ", "))

	return rs, nil
}
