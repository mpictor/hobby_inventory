package db

import (
	"database/sql"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func ns(s string) sql.NullString { return sql.NullString{Valid: len(s) > 0, String: s} }

func TestTTLRow_parsePN(t *testing.T) {
	for name, td := range map[string]struct {
		want      LogicRow
		expectErr bool
	}{
		"MC3029p": {
			want: LogicRow{
				Prefix: ns("MC"),
				Series: ns("30"),
				Func:   "29",
				Sfx:    ns("P"),
			},
		},
		"54HC257": {
			want: LogicRow{
				Series: ns("54"),
				Family: ns("HC"),
				Func:   "257",
			},
		},
		"DM74LS02N": {
			want: LogicRow{
				Prefix: ns("DM"),
				Series: ns("74"),
				Family: ns("LS"),
				Func:   "02",
				Sfx:    ns("N"),
			},
		},
		"74HC4543": {
			want: LogicRow{
				Series: ns("74"),
				Family: ns("HC"),
				Func:   "4543",
			},
		},
		"SN75150P": {
			want: LogicRow{
				Prefix: ns("SN"),
				Series: ns("75"),
				Func:   "150",
				Sfx:    ns("P"),
			},
		},
	} {
		t.Run(name, func(t *testing.T) {
			row := &LogicRow{}
			err := row.parsePN(name)
			if err != nil != td.expectErr {
				t.Fatalf("expectErr=%t but got %s", td.expectErr, err)
			}
			if d := cmp.Diff(td.want, *row, cmp.AllowUnexported(LogicRow{})); len(d) > 0 {
				t.Errorf("--want ++got\n%s", d)
			}
		})
	}
}

func TestLogic_SetRow(t *testing.T) {
	for name, td := range map[string]struct {
		kvs       []string
		want      LogicRow
		expectErr bool
	}{
		"unhandled key": {
			kvs:       []string{"pn=F74AHCTLS999UB", "qty=55", "fmax=1.21JigoHertz", "pkg=SO-8765", "loc=nowhere"},
			expectErr: true,
		},
		"pn conflicts with other part number parameters": {
			kvs:       []string{"pn=F74AHCTLS999UB", "qty=55", "pfx=aaa", "pkg=SO-8765", "loc=nowhere"},
			expectErr: true,
		},
		"converts pn": {
			kvs:       []string{"pn=F74AHCTLS999UB", "qty=55", "pkg=SO-8765", "loc=nowhere"},
			expectErr: false,
			want: LogicRow{
				commonFields: commonFields{
					Qty:      55,
					Package:  ns("SO-8765"),
					Location: 1,
				},
				Prefix: ns("F"),
				Series: ns("74"),
				Family: ns("AHCTLS"),
				Func:   "999",
				Sfx:    ns("UB"),
			},
		},
	} {
		t.Run(name, func(t *testing.T) {
			logic := &Logic{}
			db, _, _ := testDB(t)
			if err := logic.SetRow(db, td.kvs); err != nil != td.expectErr {
				t.Fatalf("expectErr=%t but err=%s", td.expectErr, err)
			}
			expectRows := 1
			if td.expectErr {
				expectRows = 0
			}
			if len(*logic) != expectRows {
				t.Fatalf("expect %d rows, got %d", expectRows, len(*logic))
			}
			if len(*logic) > 0 {
				if d := cmp.Diff(td.want, (*logic)[0], cmp.AllowUnexported(LogicRow{})); len(d) > 0 {
					t.Fatalf("row content differs: --want ++got\n%s", d)
				}
			}
			// TODO
		})
	}
}

// func TestTTLRow_Strings(t *testing.T) {
// 	tr := LogicRow{
// 		commonFields: commonFields{
// 			Qty:      55,
// 			Package:  ns("SO-8765"),
// 			Location: 1,
// 		},
// 		Prefix: ns("F"),
// 		Series: ns("74"),
// 		Family: ns("AHCTLS"),
// 		Func:   "999",
// 		Sfx:    ns("UB"),
// 	}
// 	want := []string{
// 		"0", "55", "", "SO-8765",
// 		"", "", "1", "",
// 		"", "F", "74", "AHCTLS",
// 		"999", "UB", "", "0",
// 	}

//		got := tr.Strings([]int{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15})
//		if d := cmp.Diff(want, got); len(d) > 0 {
//			t.Fatalf("result differs: --want ++got\n%s", d)
//		}
//	}
func TestTTL_ch(t *testing.T) {
	ttl := &Logic{}
	want := []string{
		"id", "qty", "n/pkg", "pkg", "mounting", "origin",
		"loc", "ds", "notes", "pfx", "series",
		"fam", "func", "sfx", "category", "desc",
	}

	ch := ttl.ColumnHeaders([]int{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15})

	if d := cmp.Diff(want, ch); len(d) > 0 {
		t.Fatalf("result differs: --want ++got\n%s", d)
	}
}
