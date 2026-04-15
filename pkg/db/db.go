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

// all tables the user may reference on the command line
// var ComponentTables = map[string]bool{
// 	"dev_kits":    true,
// 	"diode_tvs":   true,
// 	"line_drv":    true,
// 	"logic":       true,
// 	"opamps":      true,
// 	"opto":        true,
// 	"others":      true,
// 	"passive":     true,
// 	"power":       true,
// 	"tmr_osc_pll": true,
// 	"transistors": true,
// }

func createTables(db *sql.DB) (err error) {
	if _, err := db.Exec("BEGIN TRANSACTION"); err != nil {
		return err
	}
	defer func() {
		_, err = db.Exec("COMMIT")
	}()
	for _, s := range []dbTbl{
		&Logic{},
		&Transistors{},
		Locations{},
		logicDescriptions{},
		&Dev_kits{},
		&Diode_TVSDevs{},
		&Line_Drv{},
		&Opamps{},
		&OptoDevs{},
		&Others{},
		&PassiveDevs{},
		&PowerDevs{},
		&Tmr_Osc_PllDevs{},
	} {
		var tbl string
		tbl, err = structToCreate(s)
		if err != nil {
			return err
		}
		var res sql.Result
		res, err = db.Exec(tbl)
		if err != nil {
			return fmt.Errorf("creating %s: %w", tbl, err)
		}
		var ra int64
		ra, err = res.RowsAffected()
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
	TableName() string
}

// must be satisfied by any queryable, db-mapped struct
type mainDBtbl interface {
	dbTbl
	// ColumnHeaders(ord []int) []string
	ImportCSV(db *sql.DB, tbl string, data []byte) error // import csv. Table is passed to differentiate cmos/ttl csv (merged into logic)
	Store(*sql.DB) error                                 // store in db; db must have extant but empty table
	SetRow(db *sql.DB, kv []string) error                // map values to columns and set up a row with the data. must subsequently call Insert or Update.
	Insert(*sql.DB) error                                // like Store, but adds data to db that may already contain data.
	//Update(*sql.DB) error                                // update an existing row. TBD: how to specify the exact row
	Len() int

	// All(ord []int) iter.Seq[[]string] // ord: selects columns to show. TODO better in SELECT, but that means major changes...
}

// FIXME test this
func checkEmpty(db *sql.DB, tbl dbTbl) error {
	// if _, err := db.Exec("BEGIN TRANSACTION"); err == nil {
	// 	defer func() {
	// 		if _, err := db.Exec("COMMIT"); err != nil {
	// 			panic(err.Error())
	// 		}
	// 	}()
	// } else {
	// 	return err
	// }

	res, err := db.Query(fmt.Sprintf("SELECT COUNT(*) FROM %s", tbl.TableName()))
	if err != nil {
		return err
	}
	if !res.Next() {
		return fmt.Errorf("no result?!")
	}
	var n = 888
	if err := res.Scan(&n); err != nil {
		// n, err := res.RowsAffected()
		// if err != nil {
		return err
	}
	if n != 0 {
		return fmt.Errorf("table is not empty")
	}
	return nil
}

func insert[T dbInserter](db *sql.DB, rows []T) error {
	nrows := 0
	for _, r := range rows {
		stmt, vals, err := r.insert()
		if err != nil {
			return err
		}

		res, err := db.Exec(stmt, vals...)
		if err != nil {
			return fmt.Errorf("exec %s %v: %w", stmt, vals, err)
		}
		ra, err := res.RowsAffected()
		if err != nil {
			return err
		}
		nrows += int(ra)
	}
	if nrows != len(rows) {
		return fmt.Errorf("expect %d rows affected, got %d", len(rows), nrows)
	}
	return nil
}

type dbOut interface {
	isDbOut()
	All() iter.Seq[DefaultRow]
	Scan(*sql.Rows) error
}

type tRows struct {
	cols []string
	rows []DefaultRow
}

// FIXME ord param ignored
func (tr tRows) ColumnHeaders([]int) []string { return tr.cols }
func (tr tRows) Len() int                     { return len(tr.rows) }
func (tr tRows) All([]int) iter.Seq[interface{ Strings() []string }] {
	return func(yield func(interface{ Strings() []string }) bool) {
		for _, r := range tr.rows {
			if !yield(r) {
				return
			}
		}
	}
}

func Query(db *sql.DB, tbl CompTbl, kvs []string) (tRows, error) {
	where, err := toQueryWhere(kvs)
	if err != nil {
		return tRows{}, err
	}
	if len(where) == 0 {
		// match anything
		where = "TRUE = TRUE"
	}
	query := TblDefaultSelect[tbl]
	if len(query) == 0 {
		return tRows{}, fmt.Errorf("query template undefined for %s", tbl)
	}

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

	rows := OutputRowsType(tbl)
	if rows == nil {
		return tRows{}, fmt.Errorf("no table defined for %s", tbl)
	}

	tr := tRows{
		cols: colXlate(tbl, cols),
	}
	if err := rows.Scan(sRows); err != nil {
		return tRows{}, fmt.Errorf("StructScan: %w", err)
	}

	for r := range rows.All() {
		tr.rows = append(tr.rows, r)
	}
	return tr, nil
}

func colXlate(ct CompTbl, in []string) []string {
	xc, ok := xlateCols[ct]
	if !ok {
		return in
	}
	for i, s := range in {
		if subst, present := xc[s]; present {
			in[i] = subst
		}
	}
	return in
}

var xlateCols = map[CompTbl]map[string]string{
	CTOpamp: {
		"gainbandwidthproduct": "GBP",
		"railrail":             "R-R",
	},
	// TODO: other tables
}

// var colXlate=map[CompTbl]func([]string)[]string{
// 	CTOpamp:func(in []string) []string {
// 		for i,s:=range in{
// 			switch s{
// case
// 			default:
// 				// no action
// 			}
// 		}
// 		return in
// 	},
// }

// return a CREATE TABLE statement corresponding to the given table
func structToCreate[dT dbTbl](tbl dT) (string, error) {
	typ := reflect.TypeOf(tbl)
	if typ.Kind() == reflect.Pointer {
		typ = typ.Elem()
	}
	name := strings.ToLower(typ.Name())
	name = strings.TrimSuffix(name, "devs")
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
	typStrNull   = reflect.TypeFor[sql.NullString]()
	typIntNull   = reflect.TypeFor[sql.NullInt64]()
	typFloatNull = reflect.TypeFor[sql.NullFloat64]()
	typQty       = reflect.TypeFor[Qty]()
	typMounting  = reflect.TypeFor[Mounting]()
	typVRange    = reflect.TypeFor[VRange]()
	typBlob      = reflect.TypeFor[sqliteBlob]()
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
		case typIntNull, typQty, typMounting, typVRange:
			typ = "INTEGER"
		case typFloatNull:
			typ = "FLOAT64"
		case typBlob:
			typ = "BLOB"
		default:
			return "", "", fmt.Errorf("fieldNameType: unhandled field type %q", fld.Type)
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
