package db

import (
	"database/sql"
	"fmt"
	"strconv"
	"strings"
)

// fields common to all:
type commonFields struct {
	ID        int64          // row id in db
	Qty       Qty            // quantity
	NPkg      int64          // number per package, typically 1
	Package   sql.NullString // TO-220, DIP-14, SOIC-8, etc
	Mounting  Mounting       // through hole, smt, etc
	Origin    sql.NullString // freeform, where I got it from - digikey, estate, etc
	Location  int64          `db:"location,FK:location:id"` // maps to db table location
	Datasheet sql.NullString // local file name, or databook and page
	Notes     sql.NullString
	// Description - not in commonFields, as multiple TTL or CMOS will share one description
}

func (c commonFields) insert() (cols []string, vals []any) {
	// always skip the id?
	if c.Qty != QtyUnknown {
		cols = append(cols, "qty")
		vals = append(vals, int64(c.Qty))
	}
	cols = append(cols, "npkg", "mounting", "location")
	vals = append(vals, c.NPkg, int64(c.Mounting), c.Location)
	insertNullStr(&c.Package, "package", &cols, &vals)
	insertNullStr(&c.Origin, "origin", &cols, &vals)
	insertNullStr(&c.Datasheet, "datasheet", &cols, &vals)
	insertNullStr(&c.Notes, "notes", &cols, &vals)

	return cols, vals
}

// insert if valid
func insertNullStr(s *sql.NullString, fld string, cols *[]string, vals *[]any) {
	if !s.Valid {
		return
	}
	*cols = append(*cols, fld)
	*vals = append(*vals, s.String)
}

type Qty int

// qty
const (
	QtyMany    Qty = -99
	QtyUnknown Qty = -999
)

func ParseQty(s string) (Qty, error) {
	s = strings.TrimSpace(s)
	s = strings.ToLower(s)
	switch s {
	case "many":
		return QtyMany, nil
	case "?", "":
		return QtyUnknown, nil
	}
	q, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return QtyUnknown, err
	}
	return Qty(q), nil
}

// undefined --> use NULL

// TODO can we have array fields? array of comments would be useful

type Mounting int

const (
	MtgUnspecified Mounting = iota
	MtgUnknown
	MtgSMT
	MtgTH
	MtgPanel
	MtgChassis
	MtgOther
)

type Locations struct {
	ID          int64
	Name        string
	Description sql.NullString
}

var _ dbTbl = (*Locations)(nil)

func (Locations) isDbTbl() {}

// TODO any reason to actually have table/field descriptions in the db itself?
//
// 	descCreate = `CREATE TABLE descriptions (
// 	id integer,
// 	tblname bpchar,
// 	col bpchar, -- 'top' for description of entire table
// 	desc bpchar
// );
// `
// )

type DB interface {
	Query(query string, args ...any) (dbRows, error)
	Exec(query string, args ...any) (sql.Result, error)
}
type dbRows interface {
	Close() error
	Next() bool
	Scan(dest ...any) error
	Err() error
}

// retrieve a value from an auxiliary table, e.g. location or ttlDescription
func auxTblVal(db DB, tbl, f, v string) (int64, error) {
	q := fmt.Sprintf(`SELECT id FROM %s WHERE %s='%s'`, tbl, f, v)
	rows, err := db.Query(q)
	if err != nil {
		return 0, err
	}
	defer rows.Close()

	var id, n int64
	for rows.Next() {
		n++
		if n > 1 {
			return 0, fmt.Errorf("too many results for query %q", q)
		}
		if err := rows.Scan(&id); err != nil {
			return 0, err
		}
	}
	if rows.Err() != nil {
		return 0, rows.Err()
	}
	if n == 0 {
		// no result, add it
		q = fmt.Sprintf("INSERT INTO %s (%s) VALUES('%s')", tbl, f, v)
		res, err := db.Exec(q)
		if err != nil {
			return 0, err
		}
		id, err = res.LastInsertId()
		if err != nil {
			return 0, err
		}
	}
	return id, nil
}
