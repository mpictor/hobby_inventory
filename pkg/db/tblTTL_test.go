package db

import (
	"database/sql"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

func TestTTLRow_parsePN(t *testing.T) {
	ns := func(s string) sql.NullString { return sql.NullString{Valid: len(s) > 0, String: s} }
	for name, td := range map[string]struct {
		want      TTLRow
		expectErr bool
	}{
		"MC3029p": {
			want: TTLRow{
				Mpfx:   ns("MC"),
				Series: ns("30"),
				Func:   "29",
				Sfx:    ns("P"),
			},
		},
		"54HC257": {
			want: TTLRow{
				Series: ns("54"),
				Family: ns("HC"),
				Func:   "257",
			},
		},
		"DM74LS02N": {
			want: TTLRow{
				Mpfx:   ns("DM"),
				Series: ns("74"),
				Family: ns("LS"),
				Func:   "02",
				Sfx:    ns("N"),
			},
		},
		"74HC4543": {
			want: TTLRow{
				Series: ns("74"),
				Family: ns("HC"),
				Func:   "4543",
			},
		},
		"SN75150P": {
			want: TTLRow{
				Mpfx:   ns("SN"),
				Series: ns("75"),
				Func:   "150",
				Sfx:    ns("P"),
			},
		},
	} {
		t.Run(name, func(t *testing.T) {
			row := &TTLRow{}
			err := row.parsePN(name)
			if err != nil != td.expectErr {
				t.Fatalf("expectErr=%t but got %s", td.expectErr, err)
			}
			if d := cmp.Diff(td.want, *row, cmp.AllowUnexported(TTLRow{})); len(d) > 0 {
				t.Errorf("--want ++got\n%s", d)
			}
		})
	}
}

func TestTTL_SetRow(t *testing.T) {
	ns := func(s string) sql.NullString { return sql.NullString{Valid: len(s) > 0, String: s} }
	for name, td := range map[string]struct {
		kvs       []string
		want      TTLRow
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
			want: TTLRow{
				commonFields: commonFields{
					Qty:      55,
					Package:  ns("SO-8765"),
					Location: 1,
				},
				Mpfx:   ns("F"),
				Series: ns("74"),
				Family: ns("AHCTLS"),
				Func:   "999",
				Sfx:    ns("UB"),
			},
		},
	} {
		t.Run(name, func(t *testing.T) {
			ttl := &TTL{}
			db, _, _ := testDB(t)
			if err := ttl.SetRow(db, td.kvs); err != nil != td.expectErr {
				t.Fatalf("expectErr=%t but err=%s", td.expectErr, err)
			}
			expectRows := 1
			if td.expectErr {
				expectRows = 0
			}
			if len(*ttl) != expectRows {
				t.Fatalf("expect %d rows, got %d", expectRows, len(*ttl))
			}
			if len(*ttl) > 0 {
				if d := cmp.Diff(td.want, (*ttl)[0], cmp.AllowUnexported(commonFields{}), cmpopts.EquateComparable(TTLRow{})); len(d) > 0 {
					t.Fatalf("row content differs: --want ++got\n%s", d)
				}
			}
			// TODO
		})
	}
}
