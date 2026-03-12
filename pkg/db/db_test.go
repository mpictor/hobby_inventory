package db

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
)

// CAUTION race condition if multiple testDB()'s are in use
var hks *SQLHooks

// CAUTION do not call in nested test when parent has called!
func testDB(tb testing.TB) (db *sql.DB, h *SQLHooks, path string) {
	tmp := tb.TempDir()
	path = filepath.Join(tmp, "test.db")
	var err error
	if hks == nil {
		hks = &SQLHooks{}
	} else {
		// drop any existing records
		*hks = (*hks)[:0]
	}
	db, err = createDB(path, false, hks)
	if err != nil {
		tb.Fatal(err)
	}
	tb.Cleanup(func() { db.Close() })
	return db, hks, path
}

func Test_createDB(t *testing.T) {
	for name, td := range map[string]struct {
		dbName string
		init   func(t *testing.T, tmpdir string)
		check  func(t *testing.T, tmpdir string, db *sql.DB, err error)
	}{
		"already exists": {
			dbName: "exist.db",
			init: func(t *testing.T, tmpdir string) {
				if err := os.WriteFile(filepath.Join(tmpdir, "exist.db"), nil, 0o644); err != nil {
					t.Fatal(err)
				}
			},
			check: func(t *testing.T, tmpdir string, db *sql.DB, err error) {
				if err == nil {
					t.Fatal("expect error, got none")
				}
				if db != nil {
					t.Fatalf("expected nil db, got %#v", db)
				}
				fi, err := os.Stat(filepath.Join(tmpdir, "exist.db"))
				if err != nil {
					t.Fatal(err)
				}
				if fi.Size() != 0 {
					t.Fatalf("expected file size 0, got %d", fi.Size())
				}
			},
		},
		"cannot create": {
			dbName: "dir/file.db",
			check: func(t *testing.T, tmpdir string, db *sql.DB, err error) {
				if err == nil {
					t.Fatal("create should fail, but got nil")
				}
				if db != nil {
					t.Fatalf("db is not nil: %#v", db)
				}
				if _, err := os.Stat(filepath.Join(tmpdir, "dir")); err == nil {
					t.Fatalf("%s/dir should not exist but does", tmpdir)
				}
			},
		},
		"happy path": {
			dbName: "tst.db",
			check: func(t *testing.T, tmpdir string, db *sql.DB, err error) {
				if err != nil {
					t.Fatal(err)
				}
				if db == nil {
					t.Fatal("db is nil")
				}
			},
		},
	} {
		t.Run(name, func(t *testing.T) {
			tmp := t.TempDir()
			if td.init != nil {
				td.init(t, tmp)
			}
			db, err := createDB(filepath.Join(tmp, td.dbName), false, nil)
			td.check(t, tmp, db, err)
			if db != nil {
				if err := db.Close(); err != nil {
					t.Fatal(err)
				}
			}
		})
	}
	tableCols := map[string][]string{
		"logic": {
			// "id", "qty", "npkg", "package", "mounting", "origin", "location", "datasheet", "notes", "prefix", "series", "family", "func", "sfx", "category", "description",
			"id", "qty", "npkg", "package", "mounting", "origin", "location", "datasheet", "attrs", "notes", "vrange", "prefix", "series", "family", "func", "sfx", "category", "description",
		},
		// "cmos": {
		// 	"id", "qty", "npkg", "package", "mounting", "origin", "location", "datasheet", "notes", "prefix", "series", "func", "sfx", "category", "description",
		// 	"interesting", "moto1978",
		// },
		"locations": {"id", "name"}, //, "description"},
		// "descriptions": {"id", "tblname", "col", "desc"},
		"logicdescriptions": {"id", "desc"},
	}
	t.Run("table count", func(t *testing.T) {
		db, _, _ := testDB(t)
		if db == nil {
			t.Fatal("db is nil")
		}
		rows, err := db.Query("SELECT name FROM sqlite_master WHERE type='table'")
		if err != nil {
			t.Fatal(err)
		}
		numTables := 0
		for rows.Next() {
			var name string
			rows.Scan(&name)
			if _, ok := tableCols[name]; !ok {
				t.Errorf("table %q present in db, missing from tableCols", name)
			}
			if _, ok := allTableStructs[name]; !ok {
				t.Errorf("table missing from allTableStructs: %s", name)
			}
			numTables++
		}
		if len(tableCols) != numTables {
			t.Errorf("database contains different tables than tableCols")
		}
		if len(allTableStructs) != numTables {
			t.Errorf("database contains different tables than allTableStructs")
		}
	})

	for name, cols := range tableCols {
		t.Run("check table "+name, func(t *testing.T) {
			// could be shared
			tmp := t.TempDir()
			db, err := createDB(filepath.Join(tmp, "test.db"), false, nil)
			if err != nil {
				t.Fatal(err)
			}
			if db == nil {
				t.Fatal("db is nil")
			}
			rows, err := db.Query("SELECT * FROM " + name + " LIMIT 1")
			if err != nil {
				t.Fatal(err)
			}
			dbCols, err := rows.ColumnTypes()
			if err != nil {
				t.Fatal(err)
			}
			for i, c := range dbCols {
				if len(cols) > i && c.Name() != cols[i] {
					t.Errorf("col %d: %s vs %s", i, cols[i], c.Name())
				}
			}
			if len(cols) < len(dbCols) {
				tc := make([]string, len(dbCols))
				for i, c := range dbCols {
					tc[i] = c.Name()
				}
				t.Errorf(`db cols: "%s"`, strings.Join(tc, `", "`))
			}
			// TODO add tests of table fields?
		})
	}
}

const commonFieldsTxt = `id INTEGER PRIMARY KEY,
qty INTEGER NOT NULL,
npkg INTEGER NOT NULL,
package TEXT,
mounting INTEGER,
origin TEXT,
location INTEGER NOT NULL,
datasheet TEXT,
attrs BLOB,
notes TEXT,
`

func Test_structToCreate(t *testing.T) {
	testdata := []struct {
		name      string
		in        dbTbl
		fields    string
		addCommon bool
	}{
		{
			name: "logic",
			in:   &Logic{},
			fields: `vrange INTEGER,
prefix TEXT,
series TEXT,
family TEXT,
func TEXT NOT NULL,
sfx TEXT,
category TEXT,
description INTEGER,
FOREIGN KEY (location) REFERENCES locations (id),
FOREIGN KEY (description) REFERENCES logicDescriptions (id)`,
			addCommon: true,
		},
		// 		{
		// 			name: "cmos",
		// 			in:   &CMOS{},
		// 			fields: `prefix TEXT,
		// series TEXT,
		// func TEXT,
		// sfx TEXT,
		// category TEXT,
		// description INTEGER,
		// interesting TEXT,
		// moto1978 TEXT,
		// FOREIGN KEY (location) REFERENCES locations (id),
		// FOREIGN KEY (description) REFERENCES cmosDescriptions (id)`,
		// 			addCommon: true,
		// 		},
		{
			name: "locations",
			in:   &Locations{},
			fields: `id INTEGER PRIMARY KEY,
name TEXT NOT NULL`,
		},
	}

	for _, td := range testdata {
		t.Run(td.name, func(t *testing.T) {
			got, err := structToCreate(td.in)
			if err != nil {
				t.Fatal(err)
			}
			if strings.Contains(strings.ToLower(got), "commonfields") {
				t.Fatal("contains commonFields, should be expanded")
			}
			common := ""
			if td.addCommon {
				common = commonFieldsTxt
			}
			want := fmt.Sprintf("CREATE TABLE %s(\n%s%s\n)", td.name, common, td.fields)
			if d := cmp.Diff(want, got); len(d) > 0 {
				t.Errorf("differs: ---want +++got\n%s", d)
			}
		})
	}
}

var allTableStructs = map[string]dbTbl{
	"logic": &Logic{},
	// "cmos":            &CMOS{},
	"locations":         &Locations{},
	"logicdescriptions": &logicDescriptions{},
}

func TestQuery(t *testing.T) {
	db, err := openDB("../../db/db.sqlite")
	if err != nil {
		t.Fatal(err)
	}
	// db, _, _ := testDB(t)
	rows, err := Query(db, "logic", []string{"func=29"})
	if err != nil {
		t.Fatal(err)
	}
	t.Error(rows)
}
