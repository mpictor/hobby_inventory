package db

import (
	"database/sql"
	"errors"
	"fmt"
	"iter"
	"log"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"

	"github.com/jmoiron/sqlx"
	"github.com/qustavo/sqlhooks/v2"
	"modernc.org/sqlite"
)

var Verbose bool

var dbPath = func() string {
	home := os.Getenv("HOME")
	return filepath.Join(home, "Documents", "inventory", "db", "db.sqlite")
}()

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
func Create(overwrite bool) (*sql.DB, error) { return createDB(dbPath, overwrite, nil) }

var registerOnce sync.Once

// like Create but takes db path; hooks may be specified
func createDB(path string, overwrite bool, h sqlhooks.Hooks) (*sql.DB, error) {
	exists := false
	if _, err := os.Stat(path); err == nil || !errors.Is(err, os.ErrNotExist) {
		exists = true
	}
	if exists && !overwrite {
		return nil, os.ErrExist
	}
	if overwrite {
		if err := os.Remove(path); err != nil {
			return nil, err
		}
	}
	driver := "sqlite"
	if h != nil {
		driver = "sqliteWithHooks"
		registerOnce.Do(func() {
			sql.Register(driver, sqlhooks.Wrap(&sqlite.Driver{}, h))
		})
	}

	db, err := sql.Open(driver, path)
	if err != nil {
		return nil, err
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
	for _, s := range []dbTbl{
		&Logic{},
		// &CMOS{},
		Locations{},
		logicDescriptions{},
	} {
		tbl, err := structToCreate(s)
		if err != nil {
			return err
		}
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
	ColumnHeaders(ord []int) []string
	ImportCSV(db *sql.DB, tbl string, data []byte) error // import csv. Table is passed to differentiate cmos/ttl csv (merged into logic)
	Store(*sql.DB) error                                 // store in db; db must have extant but empty table
	SetRow(db *sql.DB, kv []string) error                // map values to columns and set up a row with the data. must subsequently call Insert or Update.
	Insert(*sql.DB) error                                // like Store, but adds data to db that may already contain data.
	Update(*sql.DB) error                                // update an existing row. TBD: how to specify the exact row
	Len() int

	// All(ord []int) iter.Seq[[]string] // ord: selects columns to show. TODO better in SELECT, but that means major changes...
}

// func Query(db *sqlx.DB, tbl string, kvs []string) (mainDBtbl, error) {
// 	mt := GetTbl(tbl)
// 	if mt == nil {
// 		return nil, fmt.Errorf("unknown table %s", tbl)
// 	}
// 	where, err := toQueryWhere(kvs)
// 	if err != nil {
// 		return nil, err
// 	}
// 	query := fmt.Sprintf("SELECT * FROM %s WHERE %s", tbl, where)
// 	if Verbose {
// 		log.Printf("query: %s", query)
// 	}
// 	if err := db.Select(mt, query); err != nil {
// 		return nil, err
// 	}
// 	if Verbose {
// 		log.Printf("result: %d rows", mt.Len())
// 	}
// 	return mt, nil
// }

type tRows struct {
	cols []string
	// rows [][]string
	rows []DefaultRow
}

// FIXME cols param ignored
func (tr tRows) ColumnHeaders(cols []int) []string { return tr.cols }
func (tr tRows) Len() int                          { return len(tr.rows) }
func (tr tRows) All(cols []int) iter.Seq[interface{ Strings() []string }] {
	return func(yield func(interface{ Strings() []string }) bool) {
		for _, r := range tr.rows {
			if !yield(r) {
				return
			}
		}
	}
}

func Query(db *sql.DB, tbl string, kvs []string) (tRows, error) {
	where, err := toQueryWhere(kvs)
	if err != nil {
		return tRows{}, err
	}
	query := TblDefaultSelect[tbl]

	// FIXME FIXME turn this into an actual safe query
	query = strings.Replace(query, "?", where, 1)

	if Verbose {
		log.Printf("query: %s", query)
	}

	sRows, err := db.Query(query)
	if err != nil {
		return tRows{}, fmt.Errorf("error %w executing query %s", err, query)
	}
	cols, err := sRows.Columns()
	if err != nil {
		return tRows{}, err
	}
	if Verbose {
		log.Printf("query: cols %v", cols)
	}
	tr := tRows{
		cols: cols,
	}
	// for sRows.Next() {
	// 	row := logicOutRow{}
	// 	if err := sRows.Scan(&row); err != nil {
	// 		return tRows{}, err
	// 	}
	// 	tr.rows = append(tr.rows, row)
	// }
	rows := []logicOutRow{}
	if err := sqlx.StructScan(sRows, &rows); err != nil {
		return tRows{}, fmt.Errorf("StructScan: %w", err)
	}
	for _, r := range rows {
		tr.rows = append(tr.rows, r)
	}
	// tr.rows=([]DefaultRow)rows
	return tr, nil
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
	if typ.Kind() == reflect.Slice { // TODO what about []byte
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
	typVRange   = reflect.TypeFor[VRange]()
	typBlob     = reflect.TypeFor[sqliteBlob]()
)

// get field name and type; also returns foreign key statement when
// struct tag contains ",FK:table:field[,...]"
func fieldNameType(fld reflect.StructField) (col string, fkey string, err error) {
	var name, typ string

	name = strings.ToLower(fld.Name)
	// TODO []byte
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
		case typIntNull, typQty, typMounting, typVRange:
			typ = "INTEGER"
		case typBlob:
			typ = "BLOB"
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
