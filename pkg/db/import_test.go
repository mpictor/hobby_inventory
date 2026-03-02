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

	creates := []string{
		`CREATE TABLE ttl(
id INTEGER PRIMARY KEY,
qty INTEGER,
npkg INTEGER NOT NULL,
package TEXT,
mounting INTEGER,
origin TEXT,
location INTEGER NOT NULL,
datasheet TEXT,
notes TEXT,
mpfx TEXT,
series TEXT,
family TEXT,
func TEXT NOT NULL,
sfx TEXT,
category TEXT,
description INTEGER,
FOREIGN KEY (location) REFERENCES locations (id),
FOREIGN KEY (description) REFERENCES ttlDescriptions (id)
)`,
		`CREATE TABLE cmos(
id INTEGER PRIMARY KEY,
qty INTEGER,
npkg INTEGER NOT NULL,
package TEXT,
mounting INTEGER,
origin TEXT,
location INTEGER NOT NULL,
datasheet TEXT,
notes TEXT,
mpfx TEXT,
series TEXT,
func TEXT,
sfx TEXT,
category TEXT,
description INTEGER,
interesting TEXT,
moto1978 TEXT,
FOREIGN KEY (location) REFERENCES locations (id),
FOREIGN KEY (description) REFERENCES cmosDescriptions (id)
)`,
		`CREATE TABLE locations(
id INTEGER PRIMARY KEY,
name TEXT NOT NULL
)`,
		`CREATE TABLE ttldescriptions(
id INTEGER PRIMARY KEY,
desc TEXT NOT NULL
)`,
	}

	testdata := map[string]struct {
		tbl        mainDBtbl
		csv        string
		statements []string
	}{
		"ttl": {
			tbl: &TTL{},
			csv: ttlCSV,
			statements: []string{
				"SELECT id FROM ttlDescriptions WHERE desc='MTTL III 3-in NAND term line drv'",
				"INSERT INTO ttlDescriptions (desc) VALUES('MTTL III 3-in NAND term line drv')",
				"SELECT id FROM ttlDescriptions WHERE desc='quad 2-in mux tristate'",
				"INSERT INTO ttlDescriptions (desc) VALUES('quad 2-in mux tristate')",

				"SELECT id FROM locations WHERE name='74 series logic box'",
				"INSERT INTO locations (name) VALUES('74 series logic box')",

				"INSERT INTO ttl (qty,npkg,mounting,location,origin,mpfx,series,func,sfx,category,description) VALUES(?,?,?,?,?,?,?,?,?,?,?); {2,0,0,0,swico,MC,30,29,P,buf/drv/inv,1}",
				"INSERT INTO ttl (qty,npkg,mounting,location,series,family,func,category,description) VALUES(?,?,?,?,?,?,?,?,?); {3,0,0,1,54,HC,257,mux,2}",
			},
		},
	}
	for name, td := range testdata {
		t.Run(name, func(t *testing.T) {
			tbl := td.tbl
			db, h, _ := testDB(t)
			if err := tbl.ImportCSV(db, []byte(td.csv)); err != nil {
				t.Fatal(err)
			}
			if err := tbl.Insert(db); err != nil {
				t.Fatal(err)
			}
			stmts := creates
			stmts = append(stmts, td.statements...)
			h.checkStatements(t, stmts)
		})
	}
}
