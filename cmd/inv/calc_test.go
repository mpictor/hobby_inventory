package main

import (
	"math"
	"strings"
	"testing"
)

func Test_resistorWattage(t *testing.T) {
	args := strings.Split("rw r=1M p=.5", " ")

	for name, fn := range map[string]func([]string) (string, error){
		"DK": resistorWattageDKnonlin,
		// "original": resistorWattageOrig,
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
		// "original": resistorWattageOrig,
	} {
		b.Run(name, func(b *testing.B) {
			res, err := fn(args)
			if err != nil {
				b.Fatal(err)
			}
			if len(res) == 0 {
				b.Fatal("no output")
			}
		})
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
