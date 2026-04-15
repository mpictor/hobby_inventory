package abbrev

import (
	"regexp"
	"testing"
)

func TestReturnsDataWithOne(t *testing.T) {
	testStrings := []string{"ruby"}
	abbreved := Abbrev(testStrings)
	if len(abbreved) == 0 {
		t.Errorf("abbreved data is empty.")
	}
}

func TestReturnsDataWithMultiple(t *testing.T) {
	testStrings := []string{"ruby", "python", "rules"}
	abbreved := Abbrev(testStrings)
	if len(abbreved) == 0 {
		t.Errorf("abbreved data is empty.")
	}
}
func TestReturnsDataUnique(t *testing.T) {
	testStrings := []string{"ruby", "python", "rules"}
	abbreved := Abbrev(testStrings)
	if _, ok := abbreved["ru"]; ok {
		t.Errorf("ru isn't unique between ruby and rul.")
	}
}

func TestAbbrevMatching(t *testing.T) {
	testStrings := []string{"ruby", "python", "rules"}
	pat := "p+"
	abbreved := AbbrevMatching(testStrings, pat)
	for k := range abbreved {
		if matched, _ := regexp.MatchString(pat, k); !matched {
			t.Errorf("key %s is added even if it doesn't match the pattern %s", k, pat)
		}
	}
}

func TestAbbrev(t *testing.T) {
	in := []string{
		"logic",
		"transistors",
		"dev_kits",
		"diode_tvs",
		"line_drv",
		"opamps",
		"opto",
		"others",
		"passive",
		"power",
		"tmr_osc_pll",
		"dk",
		"d",
		"tvs",
		"ttl",
		"cmos",
		"devkit",
		"pwr",
		"osc",
		"pll",
		"q",
		"bipolar",
		"fet",
	}
	abbr := Abbrev(in)
	maxLen := 0
	for _, k := range in {
		maxLen = max(maxLen, len(k))
		v, ok := abbr[k]
		if !ok {
			t.Errorf("input kw %s missing in output", k)
		} else if v != k {
			// for inputs, value should always match key
			t.Errorf("mismatch: %s -> %s", k, v)
		}
	}
	if _, ok := abbr[""]; ok {
		t.Errorf("empty string in output")
	}
	lengths := make([]int, maxLen)
	for k := range abbr {
		lengths[len(k)-1]++
	}
	for i, l := range lengths {
		t.Logf("%d of length %d", l, i+1)
	}
}
