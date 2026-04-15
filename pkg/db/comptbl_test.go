package db

import (
	"testing"
)

func TestParseCompTbl(t *testing.T) {
	tests := []struct {
		name string // description of this test case
		// Named input parameters for target function.
		s       string
		want    CompTbl
		wantErr bool
	}{
		{name: "q", s: "q", want: CTTransistors},
		{name: "tmr", s: "tmr", want: CTTmrOscPll},
		{name: "d", s: "d", want: CTDiodeTVS},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, gotErr := ParseCompTbl(tt.s)
			if gotErr != nil {
				if !tt.wantErr {
					t.Errorf("ParseCompTbl() failed: %v", gotErr)
				}
				return
			}
			if tt.wantErr {
				t.Fatal("ParseCompTbl() succeeded unexpectedly")
			}
			if tt.want != got {
				t.Errorf("ParseCompTbl() = %v, want %v", got, tt.want)
			}
		})
	}
	t.Run("test aliases", func(t *testing.T) {
		for alias, tbl := range ctAliases {
			if _, ok := ctAbbrev[alias]; !ok {
				t.Errorf("alias %s for %s missing from ctAbbrev", alias, tbl)
			}
		}
		for _, v := range ct2str {
			if _, ok := ctAbbrev[v]; !ok {
				t.Errorf("table %s missing from ctAbbrev", v)
			}
		}
	})
}
