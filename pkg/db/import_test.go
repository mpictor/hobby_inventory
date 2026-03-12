package db

import (
	"testing"
)

const (
	ttlCSV = `"prefix","series","family","function","sfx","PN","category","qty","pkg","description","origin","location","notes"
"MC",30,,29,"P","MC3029P","buf/drv/inv",2,,"MTTL III 3-in NAND term line drv","srpls",,
,54,"HC",257,,"54HC257","mux",3,,"quad 2-in mux tristate",,"74 series logic box",
,,,,,,,,,,,,
`
	cmosCSV = `
4xxx and other wide-voltage-range CMOS,,,,,,,,,,,,,,,,,,
mpfx,ord,func,sfx,PN,category,qty,,description,location,interesting,Motorola 1978,comments,,,,,,
CD,40,4001,BCN,CD4001BCN,logic,1,,quad 2-in NOR,digi/ana mux/xtal drawer,,,,,,,,,
FT,40,4001,,FT4001,logic,2,,quad 2-in NOR,asmbly box,,,,,,,,,
,,,,MK50282N,,2,,5 function 8-digit calculator,asmbly box,,mostek,,,,,,,
,,,,UCN5810A,shift reg,5,,BiMOS II serial input latched 10-bit driver 60V ,44xx/45xx cmos box,,,,,,,,,
,,,,UCN5821A,shift reg,5,," 8-bit serial-In, latched HV drivers 50v 500mA OC",44xx/45xx cmos box,,,likely similar: allegro A6821,,,,,,
,,,,,,,,,,,,,,,,,,
,,,,,,,,,,,,,,,,,,
,,,,,,,,,,,,,,,,,,
,,,,,,,,,,,,,,,,,,
,,,,,,,,,,,,,,,,,,
,,,,,,,,,,,,,,,,,,
,,,,,,,,,,,,,,,,,,
,,,,,,,,,,,,,,,,,,
,,,,HANDWRITTEN,,,,,,,,,,,,,,
,,,,4017,,5,,decade counter/divider,breadboard,,,,,,,,,
,,,,14049,,16,,hex inv/buffer,44xx/45xx cmos box,,,,,,,,,
,,,,14077,,8,,quad XNOR,44xx/45xx cmos box,,,,,,,,,
,,,,14443,,17,,6-ch 8-10b A/D conv linear subsystem,44xx/45xx cmos box,,7-264,,,,,,,
,,,,14460,,8,,automotive speed ctrl processor,44xx/45xx cmos box,,7-287,flow chart 7-290 (pdf 339),,,,,,
,,,,14490,,17,,hex contact bounce eliminator,44xx/45xx cmos box,,7-326,,,,,,,
,,,,14519,,10,,4-bit AND/OR selector,44xx/45xx cmos box,,7-421,,,,,,,
,,,,14552,,3,,64x4 SRAM,44xx/45xx cmos box,,7-544 (593),,,,,,,
`
)

func TestImportCSV(t *testing.T) {

	creates := []string{
		`CREATE TABLE logic(
id INTEGER PRIMARY KEY,
qty INTEGER NOT NULL,
npkg INTEGER NOT NULL,
package TEXT,
mounting INTEGER,
origin TEXT,
location INTEGER NOT NULL,
datasheet TEXT,
attrs BLOB,
notes TEXT,
vrange INTEGER,
prefix TEXT,
series TEXT,
family TEXT,
func TEXT NOT NULL,
sfx TEXT,
category TEXT,
description INTEGER,
FOREIGN KEY (location) REFERENCES locations (id),
FOREIGN KEY (description) REFERENCES logicDescriptions (id)
)`,
		// 		`CREATE TABLE cmos(
		// id INTEGER PRIMARY KEY,
		// qty INTEGER NOT NULL,
		// npkg INTEGER NOT NULL,
		// package TEXT,
		// mounting INTEGER,
		// origin TEXT,
		// location INTEGER NOT NULL,
		// datasheet TEXT,
		// notes TEXT,
		// prefix TEXT,
		// series TEXT,
		// func TEXT,
		// sfx TEXT,
		// category TEXT,
		// description INTEGER,
		// interesting TEXT,
		// moto1978 TEXT,
		// FOREIGN KEY (location) REFERENCES locations (id),
		// FOREIGN KEY (description) REFERENCES cmosDescriptions (id)
		// )`,
		`CREATE TABLE locations(
id INTEGER PRIMARY KEY,
name TEXT NOT NULL
)`,
		`CREATE TABLE logicdescriptions(
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
			tbl: &Logic{},
			csv: ttlCSV,
			statements: []string{
				"SELECT id FROM logicDescriptions WHERE desc='MTTL III 3-in NAND term line drv'",
				"INSERT INTO logicDescriptions (desc) VALUES('MTTL III 3-in NAND term line drv')",
				"SELECT id FROM logicDescriptions WHERE desc='quad 2-in mux tristate'",
				"INSERT INTO logicDescriptions (desc) VALUES('quad 2-in mux tristate')",

				"SELECT id FROM locations WHERE name='74 series logic box'",
				"INSERT INTO locations (name) VALUES('74 series logic box')",

				"INSERT INTO logic (qty,npkg,mounting,location,origin,vrange,prefix,series,func,sfx,category,description) VALUES(?,?,?,?,?,?,?,?,?,?,?,?); {2,0,0,0,srpls,1,MC,30,29,P,buf/drv/inv,1}",
				"INSERT INTO logic (qty,npkg,mounting,location,vrange,series,family,func,category,description) VALUES(?,?,?,?,?,?,?,?,?,?); {3,0,0,1,1,54,HC,257,mux,2}",
			},
		},
		"cmos": {
			tbl: &Logic{},
			csv: cmosCSV,
			statements: []string{
				"SELECT id FROM logicDescriptions WHERE desc='quad 2-in NOR'",
				"INSERT INTO logicDescriptions (desc) VALUES('quad 2-in NOR')",
				"SELECT id FROM locations WHERE name='digi/ana mux/xtal drawer'",
				"INSERT INTO locations (name) VALUES('digi/ana mux/xtal drawer')",
				"SELECT id FROM logicDescriptions WHERE desc='quad 2-in NOR'",
				"SELECT id FROM locations WHERE name='asmbly box'",
				"INSERT INTO locations (name) VALUES('asmbly box')",
				"SELECT id FROM logicDescriptions WHERE desc='5 function 8-digit calculator'",
				"INSERT INTO logicDescriptions (desc) VALUES('5 function 8-digit calculator')",
				"SELECT id FROM locations WHERE name='asmbly box'",
				"SELECT id FROM logicDescriptions WHERE desc='BiMOS II serial input latched 10-bit driver 60V '",
				"INSERT INTO logicDescriptions (desc) VALUES('BiMOS II serial input latched 10-bit driver 60V ')",
				"SELECT id FROM locations WHERE name='44xx/45xx cmos box'",
				"INSERT INTO locations (name) VALUES('44xx/45xx cmos box')",
				"SELECT id FROM logicDescriptions WHERE desc=' 8-bit serial-In, latched HV drivers 50v 500mA OC'",
				"INSERT INTO logicDescriptions (desc) VALUES(' 8-bit serial-In, latched HV drivers 50v 500mA OC')",
				"SELECT id FROM locations WHERE name='44xx/45xx cmos box'",
				"SELECT id FROM logicDescriptions WHERE desc='decade counter/divider'",
				"INSERT INTO logicDescriptions (desc) VALUES('decade counter/divider')",
				"SELECT id FROM locations WHERE name='breadboard'",
				"INSERT INTO locations (name) VALUES('breadboard')",
				"SELECT id FROM logicDescriptions WHERE desc='hex inv/buffer'",
				"INSERT INTO logicDescriptions (desc) VALUES('hex inv/buffer')",
				"SELECT id FROM locations WHERE name='44xx/45xx cmos box'",
				"SELECT id FROM logicDescriptions WHERE desc='quad XNOR'",
				"INSERT INTO logicDescriptions (desc) VALUES('quad XNOR')",
				"SELECT id FROM locations WHERE name='44xx/45xx cmos box'",
				"SELECT id FROM logicDescriptions WHERE desc='6-ch 8-10b A/D conv linear subsystem'",
				"INSERT INTO logicDescriptions (desc) VALUES('6-ch 8-10b A/D conv linear subsystem')",
				"SELECT id FROM locations WHERE name='44xx/45xx cmos box'",
				"SELECT id FROM logicDescriptions WHERE desc='automotive speed ctrl processor'",
				"INSERT INTO logicDescriptions (desc) VALUES('automotive speed ctrl processor')",
				"SELECT id FROM locations WHERE name='44xx/45xx cmos box'",
				"SELECT id FROM logicDescriptions WHERE desc='hex contact bounce eliminator'",
				"INSERT INTO logicDescriptions (desc) VALUES('hex contact bounce eliminator')",
				"SELECT id FROM locations WHERE name='44xx/45xx cmos box'",
				"SELECT id FROM logicDescriptions WHERE desc='4-bit AND/OR selector'",
				"INSERT INTO logicDescriptions (desc) VALUES('4-bit AND/OR selector')",
				"SELECT id FROM locations WHERE name='44xx/45xx cmos box'",
				"SELECT id FROM logicDescriptions WHERE desc='64x4 SRAM'",
				"INSERT INTO logicDescriptions (desc) VALUES('64x4 SRAM')",
				"SELECT id FROM locations WHERE name='44xx/45xx cmos box'",
				"INSERT INTO logic (qty,npkg,mounting,location,vrange,prefix,series,func,sfx,category,description) VALUES(?,?,?,?,?,?,?,?,?,?,?); {1,0,0,1,2,CD,40,4001,BCN,logic,1}",
				"INSERT INTO logic (qty,npkg,mounting,location,vrange,prefix,series,func,category,description) VALUES(?,?,?,?,?,?,?,?,?,?); {2,0,0,2,2,FT,40,4001,logic,1}",
				"INSERT INTO logic (qty,npkg,mounting,location,datasheet,vrange,func,description) VALUES(?,?,?,?,?,?,?,?); {2,0,0,2,moto78:mostek,2,MK50282N,2}",
				"INSERT INTO logic (qty,npkg,mounting,location,vrange,func,category,description) VALUES(?,?,?,?,?,?,?,?); {5,0,0,3,2,UCN5810A,shift reg,3}",
				"INSERT INTO logic (qty,npkg,mounting,location,notes,vrange,func,category,description) VALUES(?,?,?,?,?,?,?,?,?); {5,0,0,3,likely similar: allegro A6821,2,UCN5821A,shift reg,4}",
				// "INSERT INTO logic (qty,npkg,mounting,location,vrange,func) VALUES(?,?,?,?,?,?); {0,0,0,0,2,}",
				"INSERT INTO logic (qty,npkg,mounting,location,vrange,func,description) VALUES(?,?,?,?,?,?,?); {5,0,0,4,2,4017,5}",
				"INSERT INTO logic (qty,npkg,mounting,location,vrange,func,description) VALUES(?,?,?,?,?,?,?); {16,0,0,3,2,14049,6}",
				"INSERT INTO logic (qty,npkg,mounting,location,vrange,func,description) VALUES(?,?,?,?,?,?,?); {8,0,0,3,2,14077,7}",
				"INSERT INTO logic (qty,npkg,mounting,location,datasheet,vrange,func,description) VALUES(?,?,?,?,?,?,?,?); {17,0,0,3,moto78:7-264,2,14443,8}",
				"INSERT INTO logic (qty,npkg,mounting,location,datasheet,notes,vrange,func,description) VALUES(?,?,?,?,?,?,?,?,?); {8,0,0,3,moto78:7-287,flow chart 7-290 (pdf 339),2,14460,9}",
				"INSERT INTO logic (qty,npkg,mounting,location,datasheet,vrange,func,description) VALUES(?,?,?,?,?,?,?,?); {17,0,0,3,moto78:7-326,2,14490,10}",
				"INSERT INTO logic (qty,npkg,mounting,location,datasheet,vrange,func,description) VALUES(?,?,?,?,?,?,?,?); {10,0,0,3,moto78:7-421,2,14519,11}",
				"INSERT INTO logic (qty,npkg,mounting,location,datasheet,vrange,func,description) VALUES(?,?,?,?,?,?,?,?); {3,0,0,3,moto78:7-544 (593),2,14552,12}",
			},
		},
	}
	for name, td := range testdata {
		t.Run(name, func(t *testing.T) {
			tbl := td.tbl
			db, h, _ := testDB(t)
			if err := tbl.ImportCSV(db, name, []byte(td.csv)); err != nil {
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
