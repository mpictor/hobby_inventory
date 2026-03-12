package main

import (
	"math"
	"strings"
	"testing"

	"gonum.org/v1/gonum/optimize"
)

func Test_resistorWattage(t *testing.T) {
	args := strings.Split("rw r=1M p=.5", " ")

	for name, fn := range map[string]func([]string) (string, error){
		"DK":       resistorWattageDKnonlin,
		"amovah":   resistorWattageAmovah,
		"gonum":    resistorWattageGonum,
		"original": resistorWattageOrig,
	} {
		t.Run(name, func(t *testing.T) {
			res, err := fn(args)
			if err != nil {
				t.Fatal(err)
			}
			if res != "asdfklsj" {
				t.Fatal(res)
			}
		})
	}
}

func Benchmark_resistorWattage(b *testing.B) {
	args := strings.Split("rw r=1M p=.5", " ")

	for name, fn := range map[string]func([]string) (string, error){
		"DK": resistorWattageDKnonlin,
		// "amovah":   resistorWattageAmovah,
		"gonum":    resistorWattageGonum,
		"original": resistorWattageOrig,
	} {
		b.Run(name, func(b *testing.B) {
			res, err := fn(args)
			if err != nil {
				b.Fatal(err)
			}
			if len(res) == 0 {
				b.Fatal("no output")
			}
			// if res != "asdfklsj" {
			// 	b.Fatal(res)
			// }
		})
	}

}

func Test_parseEquation(t *testing.T) {
	equation := "x^2 + y^2 = 1"
	variables := []string{"x", "y"}
	systemFunc := parseEquation(equation, variables)
	x := []float64{0.5, 0.5}

	problem := optimize.Problem{Func: systemFunc}
	result, err := optimize.Minimize(problem, x, nil, &optimize.NelderMead{})
	if err != nil {
		t.Fatal(err)
	}
	// t.Error(result)
	o, err := outputPretty(result.X, [][2]string{{"x", ""}, {"y", ""}})
	if err != nil {
		t.Fatal(err)
	}
	t.Error(o)
}

func Test_ohmsLawGonum(t *testing.T) {
	vals := map[string]float64{"r": 1000000, "p": 0.5}
	res, err := ohmsLawGonum(vals)
	if err != nil {
		t.Fatal(err)
	}
	gotV := res.X[v]
	gotI := res.X[i]
	gotR := res.X[r]
	gotP := res.X[p]
	expectI := math.Sqrt(2) / 2000 // 0.0007071...
	if math.Abs(gotI-expectI) > 0.000000001 {
		t.Fatalf("expect %eA, got %e", expectI, gotI) // pass
	}
	expectP := 0.5 // i * i * r
	if math.Abs(gotP-expectP) > 0.0001 {
		t.Fatalf("expect %fW, got %fW", gotP, expectP) // pass
	}
	expectR := 1e6
	if math.Abs(gotR-expectR) > 0.001 {
		t.Fatalf("expect %f ohms, got %f ohms", expectR, gotR) // pass
	}
	expectV := math.Sqrt(2) / 2 * 1000 // 707.1...
	if math.Abs(gotV-expectV) > 0.001 {
		t.Fatalf("expect %fV, got %fV", expectV, gotV) // fails: gotV is ~2.07V
	}
}

func Test_ohmsLawDNnonlin(t *testing.T) {
	vals := map[string]float64{"r": 1000000, "p": 0.5}
	res, err := ohmsLawDKnonlin(vals)
	if err != nil {
		t.Fatal(err)
	}
	gotV := res.X[v]
	gotI := res.X[i]
	gotR := res.X[r]
	gotP := res.X[p]
	expectI := math.Sqrt(2) / 2000 // 0.0007071...
	if math.Abs(gotI-expectI) > 0.000000001 {
		t.Fatalf("expect %eA, got %e", expectI, gotI)
	}
	expectP := 0.5 // i * i * r
	if math.Abs(gotP-expectP) > 0.0001 {
		t.Fatalf("expect %fW, got %fW", gotP, expectP)
	}
	expectR := 1e6
	if math.Abs(gotR-expectR) > 0.001 {
		t.Fatalf("expect %f ohms, got %f ohms", expectR, gotR)
	}
	expectV := math.Sqrt(2) / 2 * 1000 // 707.1...
	if math.Abs(gotV-expectV) > 0.001 {
		t.Fatalf("expect %fV, got %fV", expectV, gotV)
	}
}
