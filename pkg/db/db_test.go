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
			db, err := createDB(filepath.Join(tmp, td.dbName), false)
			td.check(t, tmp, db, err)
			if db != nil {
				if err := db.Close(); err != nil {
					t.Fatal(err)
				}
			}
		})
	}
	tableCols := map[string][]string{
		"ttl": {
			"id", "qty", "npkg", "package", "mounting", "origin", "location", "datasheet", "notes", "mpfx", "series", "family", "func", "sfx", "category", "description",
		},
		"cmos": {
			"id", "qty", "npkg", "package", "mounting", "origin", "location", "datasheet", "notes", "mpfx", "series", "func", "sfx", "category", "description", "interesting", "moto1978",
		},
		"locations": {"id", "name", "description"},
		// "descriptions": {"id", "tblname", "col", "desc"},
		"ttldescriptions": {"id", "desc"},
	}
	t.Run("table count", func(t *testing.T) {
		tmp := t.TempDir()
		db, err := createDB(filepath.Join(tmp, "test.db"), false)
		if err != nil {
			t.Fatal(err)
		}
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
			db, err := createDB(filepath.Join(tmp, "test.db"), false)
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
qty INTEGER,
npkg INTEGER NOT NULL,
package TEXT,
mounting INTEGER,
origin TEXT,
location INTEGER NOT NULL,
datasheet TEXT,
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
			name: "ttl",
			in:   &TTL{},
			fields: `mpfx TEXT,
series TEXT,
family TEXT,
func TEXT NOT NULL,
sfx TEXT,
category TEXT,
description INTEGER,
FOREIGN KEY (location) REFERENCES location (id),
FOREIGN KEY (description) REFERENCES ttlDescription (id)`,
			addCommon: true,
		},
		{
			name: "cmos",
			in:   &CMOS{},
			fields: `mpfx TEXT,
series TEXT,
func TEXT,
sfx TEXT,
category TEXT,
description INTEGER,
interesting TEXT,
moto1978 TEXT,
FOREIGN KEY (location) REFERENCES location (id),
FOREIGN KEY (description) REFERENCES cmosDescription (id)`,
			addCommon: true,
		},
		{
			name: "locations",
			in:   &Locations{},
			fields: `id INTEGER PRIMARY KEY,
name TEXT NOT NULL,
description TEXT`,
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
	"ttl":             &TTL{},
	"cmos":            &CMOS{},
	"locations":       &Locations{},
	"ttldescriptions": &TTLdescriptions{},
}
