package db

import (
	"database/sql"
	"errors"
	"fmt"
	"os"
	"reflect"
	"strings"

	"github.com/jmoiron/sqlx"
	_ "modernc.org/sqlite"
)

const dbPath = "../../db/db.sqlite"

// Open existing db.
func Open() (*sql.DB, error) { return openDB(dbPath) }

// like open but db path may be specified
func openDB(path string) (*sql.DB, error) {
	if _, err := os.Stat(path); err != nil {
		return nil, err
	}
	return sql.Open("sqlite", path)
}

// Creates new db and creates tables.
func Create(dryRun bool) (*sql.DB, error) { return createDB(dbPath, dryRun) }

// like Create but db path may be specified
func createDB(path string, dryRun bool) (*sql.DB, error) {
	if _, err := os.Stat(path); err == nil || !errors.Is(err, os.ErrNotExist) {
		return nil, os.ErrExist
	}
	var db *sql.DB
	if dryRun {
		// TODO mock db
	} else {
		var err error
		db, err = sql.Open("sqlite", path)
		if err != nil {
			return nil, err
		}
	}
	if err := createTables(db); err != nil {
		return nil, err
	}
	return db, nil
}

// import from csv to one table
// determine column names (and thus columns to skip) from header?
// or hard code each?
// func importCSV() error {}

// TODO need this for round-trip tests
// func exportCSV() error {}

func createTables(db *sql.DB) error {
	// tbls := []string{
	// 	descCreate,
	// }
	for _, s := range []dbTbl{
		&TTL{},
		&CMOS{},
		Locations{},
		TTLdescriptions{},
	} {
		tbl, err := structToCreate(s)
		if err != nil {
			return err
		}
		// 	tbls = append(tbls, tbl)
		// }
		// for _, tbl := range tbls {
		res, err := db.Exec(tbl)
		if err != nil {
			return fmt.Errorf("creating %s: %w", tbl, err)
		}
		ra, err := res.RowsAffected()
		if err != nil {
			return err
		}
		if ra != 0 {
			return fmt.Errorf("rows affected: %d", ra)
		}
	}
	return nil
}

// must be satisfied by any db-mapped struct
type dbTbl interface {
	isDbTbl()
}

// must be satisfied by any queryable, db-mapped struct
type mainDBtbl interface {
	dbTbl
	ImportCSV(DB, []byte) error
	Store(DB) error // store in db; db must have extant but empty table
	ColumnHeaders() ([]string, error)
	Insert(DB) error // like Store, but inserts data.
	Render()         // human readable output, interface TBD
	Update(DB) error // update an existing row. TBD: how to specify the exact row
}

func Query[dT mainDBtbl](db *sqlx.DB, query string) (dT, error) {
	var res, zero dT
	rows, err := db.Queryx(query)
	if err != nil {
		return zero, err
	}
	if err := rows.StructScan(&res); err != nil {
		return zero, err
	}
	return res, nil
}

// return a CREATE TABLE statement corresponding to the given table
func structToCreate[dT dbTbl](tbl dT) (string, error) {
	typ := reflect.TypeOf(tbl)
	if typ.Kind() == reflect.Pointer {
		typ = typ.Elem()
	}
	name := strings.ToLower(typ.Name())
	if len(name) == 0 {
		return "", fmt.Errorf("cannot use unnamed type")
	}
	if typ.Kind() == reflect.Slice {
		typ = typ.Elem()
	}
	if typ.Kind() != reflect.Struct {
		return "", fmt.Errorf("want struct or *struct, got %s", typ.Kind())
	}

	n := countFields(typ)
	elems := make([]string, 0, n)
	var tail []string // things that must go at the end, such as foreign keys

	for fld := range iterFields(typ) {
		fn, fk, err := fieldNameType(fld)
		if err == errSkip {
			// skip this one
			continue
		}
		if err != nil {
			return "", err
		}
		elems = append(elems, fn)
		if len(fk) > 0 {
			tail = append(tail, fk)
		}
	}
	elems = append(elems, tail...)
	return fmt.Sprintf("CREATE TABLE %s(\n%s\n)", name, strings.Join(elems, ",\n")), nil
}

var (
	// sentinel value possibly returned by fieldNameType
	errSkip = fmt.Errorf("skip this field")

	// these are used for comparisons in fieldNameType
	typStrNull  = reflect.TypeFor[sql.NullString]()
	typIntNull  = reflect.TypeFor[sql.NullInt64]()
	typQty      = reflect.TypeFor[Qty]()
	typMounting = reflect.TypeFor[Mounting]()
)

// get field name and type; also returns foreign key statement when
// struct tag contains ",FK:table:field[,...]"
func fieldNameType(fld reflect.StructField) (col string, fkey string, err error) {
	var name, typ string

	name = strings.ToLower(fld.Name)
	switch fld.Type.Kind() {
	case reflect.String:
		typ = "TEXT NOT NULL"
	case reflect.Int64:
		if name == "id" {
			typ = "INTEGER PRIMARY KEY"
		} else {
			typ = "INTEGER NOT NULL"
		}
	default:
		switch fld.Type {
		case typStrNull:
			typ = "TEXT"
		case typIntNull, typQty, typMounting:
			typ = "INTEGER"
		default:
			return "", "", fmt.Errorf("unhandled field type %q", fld.Type)
		}
	}

	fullTag := fld.Tag.Get("db")
	// TODO does sqlx encode the type in this tag?
	tgName, remain, _ := strings.Cut(fullTag, ",") // remove comma and anything after
	if tgName == "-" {
		return "", "", errSkip
	}
	if len(tgName) > 0 {
		name = tgName
		if strings.HasPrefix(remain, "FK:") {
			tbl, fk, ok := strings.Cut(remain[3:], ":")
			if len(fk) == 0 {
				ok = false
			}
			if !ok {
				return "", "", fmt.Errorf("error parsing FK in tag %s", fullTag)
			}
			fk, _, _ = strings.Cut(fk, ",")
			fkey = fmt.Sprintf("FOREIGN KEY (%s) REFERENCES %s (%s)", name, tbl, fk)
		}
	}
	if len(name) == 0 {
		return "", "", fmt.Errorf("require struct tag or name") // is empty name even possible?
	}

	return fmt.Sprintf("%s %s", name, typ), fkey, nil
}

// count exported fields in struct and embedded structs
func countFields(typ reflect.Type) int {
	n := typ.NumField()
	ret := n // ret may change, but we need to iterate over unchanged n
	for i := range n {
		fld := typ.Field(i)
		if fld.Anonymous {
			ret += countFields(fld.Type)
		} else if !fld.IsExported() {
			ret--
		}
	}
	return ret
}

func iterFields(typ reflect.Type) (iter func(yield func(reflect.StructField) bool)) {
	// recursive. to stop range when yield returns false, we have to bubble up the bool.
	var it func(typ reflect.Type, yield func(reflect.StructField) bool) bool
	it = func(typ reflect.Type, yield func(reflect.StructField) bool) bool {
		n := typ.NumField()
		for i := range n {
			fld := typ.Field(i)
			if fld.Anonymous && fld.Type.Kind() == reflect.Struct {
				if !it(fld.Type, yield) {
					return false
				}
			} else if fld.IsExported() {
				if !yield(fld) {
					return false
				}
			}
		}
		return true
	}
	return func(yield func(reflect.StructField) bool) { it(typ, yield) }
}
