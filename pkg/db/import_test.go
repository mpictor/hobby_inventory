package db

import (
	"testing"
)

const (
	ttlCSV = `"mpfx","series","family","function","sfx","PN","category","qty","pkg","description","origin","location","notes"
"MC",30,,29,"P","MC3029P","buf/drv/inv",2,,"MTTL III 3-in NAND term line drv","swico",,
,54,"HC",257,,"54HC257","mux",3,,"quad 2-in mux tristate",,"74 series logic box",
`
)

func TestImportCSV(t *testing.T) {
	testdata := map[string]struct {
		tbl        mainDBtbl
		csv        string
		statements []string
	}{
		"ttl": {
			tbl: &TTL{},
			csv: ttlCSV,
			statements: []string{
				// note that some insert values will change if the database mock is improved
				"SELECT id FROM ttlDescription WHERE desc='MTTL III 3-in NAND term line drv'",
				"SELECT id FROM ttlDescription WHERE desc='quad 2-in mux tristate'",
				"SELECT id FROM locations WHERE name='74 series logic box'",
				"INSERT INTO ttlDescription (desc) VALUES('MTTL III 3-in NAND term line drv')",
				"INSERT INTO ttlDescription (desc) VALUES('quad 2-in mux tristate')",
				"INSERT INTO locations (name) VALUES('74 series logic box')",
				"INSERT INTO ttl (qty,npkg,mounting,location,origin,mpfx,series,func,sfx,category,description) VALUES(?,?,?,?,?,?,?,?,?,?,?); {2,0,0,0,swico,MC,30,29,P,buf/drv/inv,0}",
				"INSERT INTO ttl (qty,npkg,mounting,location,series,family,func,category,description) VALUES(?,?,?,?,?,?,?,?,?); {3,0,0,0,54,HC,257,mux,0}",
			},
		},
	}
	for name, td := range testdata {
		t.Run(name, func(t *testing.T) {
			tbl := td.tbl
			db := &mockDB{}
			if err := tbl.ImportCSV(db, []byte(td.csv)); err != nil {
				t.Fatal(err)
			}
			if err := tbl.Insert(db); err != nil {
				t.Fatal(err)
			}
			db.checkStatements(t, td.statements)
			// t.Errorf("\nqueries:\n  %s\nexecs:\n  %s", strings.Join(db.queries, "\n  "), strings.Join(db.execs, "\n  "))
		})
	}
}
