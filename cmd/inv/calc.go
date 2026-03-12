package main

import (
	"context"
	"fmt"
	"math"
	"strings"

	"github.com/amovah/equation"
	"github.com/amovah/equation/constants"
	"github.com/amovah/equation/operators"
	"github.com/davidkleiven/gononlin/nonlin"
	"github.com/dustin/go-humanize"
	"github.com/expr-lang/expr"
	"github.com/mpictor/hobby_inventory/pkg/db"
	"github.com/peterbourgon/ff/v4"
	"gonum.org/v1/gonum/optimize"
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

// type missingParam string

func resistorWattageOrig(args []string) (string, error) {
	args = args[1:]
	if len(args) != 2 {
		return "", errNumArgs
	}
	pm := db.ToParamMap(args, false)
	// rs := pm["r"]
	// TODO si notation?
	r, _, err := humanize.ParseSI(pm["r"])
	// r, err := strconv.ParseFloat(pm["r"], 64)
	if err != nil {
		return "", fmt.Errorf("param r: %w", err)
	}
	w, _, err := humanize.ParseSI(pm["p"])
	// w, err := strconv.ParseFloat(pm["p"], 64)
	if err != nil {
		return "", fmt.Errorf("param p: %w", err)
	}
	// if !ok {
	// 	return fmt.Errorf("require params r= and w=")
	// }
	// ws := pm["w"]
	// if len(w)==0||len(r)==0 {
	// 	return fmt.Errorf("require params r= and w=")
	// }
	// w = i^2 * r
	// i = sqrt(w/r)
	i := math.Sqrt(w / r)
	v := i * r
	// v= i*r
	res := fmt.Sprintf("v=%f\ni=%f\n", v, i)
	return res, nil
}

// slower yet always converges... but wrong answer
// FIXME v is very wrong
// Root: (v,i,r,p)=(2.073472 , 707.106074 µ, 1000 k, 499.999499 m)
// Residual: (497.07147 n)
func resistorWattageGonum(args []string) (string, error) {
	args = args[1:]
	if len(args) != 2 {
		return "", errNumArgs
	}
	pm := db.ToParamMap(args, false)
	vals := make(map[string]float64, len(pm))
	for k, v := range pm {
		val, unit, err := humanize.ParseSI(v)
		if err != nil {
			return "", err
		}
		if unit != "" {
			return "", fmt.Errorf("unhandled unit %s", unit)
		}
		vals[k] = val
	}
	res, err := ohmsLawGonum(vals)

	if err != nil {
		return "", err
	}
	// solX := make([]string, 0, len(res.X))
	// for _, x := range res.X {
	// 	solX = append(solX, strings.TrimSpace(humanize.SI(x, "")))
	// }
	// solF := humanize.SI(res.F, "")
	// rs := fmt.Sprintf("Root: (v,i,r,p)=(%s)\nResidual: %s\n", strings.Join(solX, ", "), solF)
	meta := [][2]string{{"v", "V"}, {"i", "A"}, {"r", "Ω"}, {"p", "W"}}
	out, err := outputPretty(res.X, meta)
	if err != nil {
		return "", err
	}
	rs := fmt.Sprintf("%s\nResidual=%f\n", out, res.F)

	return rs, nil
}

// humanize.SI, but without the space after the float
func si(input float64, unit string, decimals int) string {
	value, prefix := humanize.ComputeSI(input)
	return humanize.FtoaWithDigits(value, decimals) + prefix + unit
}

func outputPretty(X []float64, meta [][2]string) (string, error) {
	if len(X) != len(meta) {
		return "", fmt.Errorf("length mismatch: %d vs %d", len(X), len(meta))
	}
	outs := make([]string, len(X))
	for i, x := range X {
		outs[i] = fmt.Sprintf("%s=%s", meta[i][0], si(x, meta[i][1], 3))
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

func ohmsLawGonum(vals map[string]float64) (*optimize.Result, error) {
	if len(vals) != 2 {
		return nil, fmt.Errorf("len(vals)!=2")
	}
	problem := optimize.Problem{
		Func: func(vars []float64) float64 {
			out := make([]float64, len(vars))
			// define our 2 equations:
			// i=v/r   --> 0 = v / r - i
			out[0] = vars[v]/vars[r] - vars[i]
			// p=i^2*r --> 0 = i^2 * r - p
			out[1] = math.Pow(vars[i], 2)*vars[r] - vars[p]
			j := 2
			// define an equation for each of the vars with known values,
			// in zero-equation form (e.g. r=5 --> 0 = r - 5)
			for k, v := range vals {
				pos := varLoc(k)
				out[j] = vars[pos] - v
				j++
			}
			// sum of squares of errors
			res := 0.0
			for _, o := range out {
				// if vars[n] < 0 {
				// penalize negative values
				// o -= 100
				// }
				res += o * o
			}
			return res
		},
	}
	// set initial values
	x0 := []float64{1, 1, 1, 1}
	for k, v := range vals {
		x0[varLoc(k)] = v
	}
	return optimize.Minimize(problem, x0, nil, &optimize.NelderMead{})
}

func ohmsLawDKnonlin(vals map[string]float64) (nonlin.Result, error) {
	problem := nonlin.Problem{
		F: func(out, x []float64) {
			// x[0]:v, x[1]:i, x[2]: r, x[3]: p
			// i=v/r   --> 0 = v / r - i
			// p=i^2*r --> 0 = i^2 * r - p
			out[0] = x[0]/x[2] - x[1]
			out[1] = math.Pow(x[1], 2)*x[2] - x[3]
			i := 2
			// convert inputs ('v=5') to 0-based, i.e. '0=v-5'
			for k, v := range vals {
				pos := varLoc(k)
				// if first {
				// 	h := humanize.SI(v, "")
				// 	fmt.Printf("%d. %s=%s  -> 0=x[%d]-%s\n", i, k, h, pos, h)
				// 	first = false
				// }
				out[i] = x[pos] - v
				i++
			}
		},
	}
	solver := nonlin.NewtonKrylov{
		Maxiter:  10000,
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
	pm := db.ToParamMap(args, false)
	vals := make(map[string]float64, len(pm))
	for k, v := range pm {
		val, unit, err := humanize.ParseSI(v)
		if err != nil {
			return "", err
		}
		if unit != "" {
			return "", fmt.Errorf("unhandled unit %s", unit)
		}
		vals[k] = val
	}
	// varLoc := func(k string) int {
	// 	switch k {
	// 	case "v":
	// 		return 0
	// 	case "i":
	// 		return 1
	// 	case "r":
	// 		return 2
	// 	case "p":
	// 		return 3
	// 	}
	// 	return -1
	// }
	// first := true
	// problem := nonlin.Problem{
	// 	F: func(out, x []float64) {
	// 		// x[0]:v, x[1]:i, x[2]: r, x[3]: p
	// 		// i=v/r   --> 0 = v / r - i
	// 		// p=i^2*r --> 0 = i^2 * r - p
	// 		out[0] = x[0]/x[2] - x[1]
	// 		out[1] = math.Pow(x[1], 2)*x[2] - x[3]
	// 		i := 2
	// 		// convert inputs ('v=5') to 0-based, i.e. '0=v-5'
	// 		for k, v := range vals {
	// 			pos := varLoc(k)
	// 			// if first {
	// 			// 	h := humanize.SI(v, "")
	// 			// 	fmt.Printf("%d. %s=%s  -> 0=x[%d]-%s\n", i, k, h, pos, h)
	// 			// 	first = false
	// 			// }
	// 			out[i] = x[pos] - v
	// 			i++
	// 		}
	// 	},
	// }
	// solver := nonlin.NewtonKrylov{
	// 	Maxiter:  10000,
	// 	StepSize: 1e-2,
	// 	Tol:      1e-7,
	// }
	// x0 := []float64{1, 1, 1, 1}
	// for k, v := range vals {
	// 	x0[varLoc(k)] = v
	// }
	// res, err := solver.Solve(problem, x0)
	res, err := ohmsLawDKnonlin(vals)
	if err != nil {
		return "", err
	}
	meta := [][2]string{{"v", "V"}, {"i", "A"}, {"r", "Ω"}, {"p", "W"}}
	out, err := outputPretty(res.X, meta)
	if err != nil {
		return "", err
	}

	// solX := make([]string, 0, len(res.X))
	// for _, x := range res.X {
	// 	// solX = append(solX, humanize.SI(x, ""))
	// 	solX = append(solX, strings.TrimSpace(humanize.SI(x, "")))
	// }
	solF := make([]string, 0, len(res.F))
	for _, f := range res.F {
		solF = append(solF, si(f, "", 5))
	}
	rs := fmt.Sprintf("%s\nremainder: %s\n", out, strings.Join(solF, ", "))
	// fmt.Printf("Root: (v, i, r, p) = (%.2f, %.2f, %.2f, %.2f)\n", res.X[0], res.X[1], res.X[2], res.X[3])
	// fmt.Printf("Function value: (%s)\n", strings.Join(solF, ", "))
	// fmt.Printf("Function value: (%.2f, %.2f, %.2f, %.2f)\n", res.F[0], res.F[1], res.F[2], res.F[3])

	return rs, nil
}

// crashes, can't handle multi-eqn systems
func resistorWattageAmovah(args []string) (string, error) {
	args = args[1:]
	if len(args) != 2 {
		return "", errNumArgs
	}
	pm := db.ToParamMap(args, false)
	vals := make(map[string]float64, len(pm))
	for k, v := range pm {
		val, unit, err := humanize.ParseSI(v)
		if err != nil {
			return "", err
		}
		if unit != "" {
			return "", fmt.Errorf("unhandled unit %s", unit)
		}
		vals[k] = val
	}

	eqn := "0=i/r-v"
	consts := constants.Defaults()
	for k, v := range vals {
		consts = append(consts, constants.Constant{
			Symbol: k,
			Value:  v,
		})
	}
	soln := equation.Solve(eqn, operators.Defaults(), consts)
	// fmt.Println(soln)
	return humanize.SI(soln, ""), nil
}

func parseEquation(equation string, variables []string) func([]float64) float64 {
	// Assume equation is "left = right", split into left and right
	parts := strings.Split(equation, "=")
	if len(parts) != 2 {
		panic("Invalid equation format")
	}
	leftExpr, err := expr.Compile(parts[0])
	// leftExpr, err := expr.NewEvaluableExpression(parts[0])
	if err != nil {
		panic(err)
	}
	rightExpr, err := expr.Compile(parts[1])
	// rightExpr, err := expr.NewEvaluableExpression(parts[1])
	if err != nil {
		panic(err)
	}

	return func(vars []float64) float64 {
		params := make(map[string]interface{})
		for i, v := range variables {
			params[v] = vars[i]
		}
		// leftExpr.Arguments
		left, err := expr.Run(leftExpr, params)
		if err != nil {
			panic(err)
		}
		// left, _ := leftExpr.Evaluate(params)
		right, err := expr.Run(rightExpr, params)
		if err != nil {
			panic(err)
		}
		// right, _ := rightExpr.Evaluate(params)
		lf, err := toFloat(left)
		if err != nil {
			panic(err)
		}
		rf, err := toFloat(right)
		if err != nil {
			panic(err)
		}
		residual := lf - rf
		return residual * residual // Sum of squares for minimization
	}
}

func toFloat(a any) (float64, error) {
	switch x := a.(type) {
	case float64:
		return x, nil
	case int:
		return float64(x), nil
	default:
		return 0, fmt.Errorf("toFloat: unhandled type %T", a)
	}
}
