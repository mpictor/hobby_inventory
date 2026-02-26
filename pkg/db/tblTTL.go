package db

import (
	"bytes"
	"database/sql"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"strings"
)

type TTLRow struct {
	commonFields
	Mpfx        sql.NullString
	Series      sql.NullString // 54, 74, etc
	Family      sql.NullString // F, LS, ACT, etc
	Func        string         // 00 (quad NAND) etc
	Sfx         sql.NullString // suffix if any
	Category    sql.NullString // buffer, flipflop, etc TODO enum or separate table??
	Description sql.NullInt64  `db:"description,FK:ttlDescription:id"` // foreign key
}

func (r TTLRow) insert() (string, []any, error) {
	cols, vals := r.commonFields.insert()
	insertNullStr(&r.Mpfx, "mpfx", &cols, &vals)
	insertNullStr(&r.Series, "series", &cols, &vals)
	insertNullStr(&r.Family, "family", &cols, &vals)
	cols = append(cols, "func")
	vals = append(vals, r.Func)
	insertNullStr(&r.Sfx, "sfx", &cols, &vals)
	insertNullStr(&r.Category, "category", &cols, &vals)
	if r.Description.Valid {
		cols = append(cols, "description")
		vals = append(vals, r.Description.Int64)
	}
	if len(cols) == 0 {
		return "", nil, fmt.Errorf("nothing to insert for %s", r)
	}
	ph := "?"
	ph += strings.Repeat(",?", len(vals)-1)
	s := fmt.Sprintf("INSERT INTO ttl (%s) VALUES(%s);", strings.Join(cols, ","), ph)
	return s, vals, nil
}

// TODO make pretty
func (r TTLRow) String() string {
	j, err := json.Marshal(r)
	if err != nil {
		panic(err)
	}
	return string(j)
}

type TTL []TTLRow

func (*TTL) isDbTbl() {}
func (ttl *TTL) ImportCSV(db DB, in []byte) error {
	r := csv.NewReader(bytes.NewReader(in))
	// skip header
	if _, err := r.Read(); err != nil {
		return err
	}
	for {
		record, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
			// log.Fatal(err)
		}

		// fmt.Println(record)
		row, err := ttlRow(db, record)
		if err != nil {
			return err
		}
		*ttl = append(*ttl, row)
	}
	return nil
}

// "mpfx","series","family","function","sfx","PN","category","qty","pkg","description","origin","location","notes"
func ttlRow(db DB, rec []string) (TTLRow, error) {
	var tr, zero TTLRow
	if len(rec) == 0 {
		return zero, fmt.Errorf("no data")
	}
	// did I actually find a use for the for-case paradigm?!
	// https://thedailywtf.com/articles/The_FOR-CASE_paradigm
	for i, v := range rec {
		if len(v) == 0 {
			continue
		}
		switch i {
		case 0:
			tr.Mpfx = sql.NullString{String: v, Valid: true}
		case 1:
			tr.Series = sql.NullString{String: v, Valid: true}
		case 2:
			tr.Family = sql.NullString{String: v, Valid: true}
		case 3:
			tr.Func = v
		case 4:
			tr.Sfx = sql.NullString{String: v, Valid: true}
		// case 5: PN (discard)
		case 6:
			tr.Category = sql.NullString{String: v, Valid: true}
		case 7:
			q, err := ParseQty(v)
			if err != nil {
				return zero, err
			}
			tr.Qty = q
		case 8:
			tr.Package = sql.NullString{String: v, Valid: true}
		case 9:
			desc, err := auxTblVal(db, "ttlDescription", "desc", v)
			if err != nil {
				// return zero, err
				// TODO log error?
			} else {
				tr.Description = sql.NullInt64{Int64: desc, Valid: true}
			}
		case 10:
			tr.Origin = sql.NullString{String: v, Valid: true}
		case 11:
			loc, err := auxTblVal(db, "locations", "name", v)
			if err != nil {
				return zero, err
			}
			tr.Location = loc
		case 12:
			// concatenate any additional cells (TODO improve?)
			v = strings.Join(rec[12:], ";")
			tr.Notes = sql.NullString{String: v, Valid: true}
		}
	}
	return tr, nil
}

func (ttl *TTL) Store(db DB) error {
	// TODO check that existing table is empty
	// TODO compose INSERT statements
	// TODO transaction?
	panic("unimplemented")
}
func (ttl *TTL) ColumnHeaders() ([]string, error) { panic("unimplemented") }
func (ttl *TTL) Insert(db DB) error {
	// var stmts []string
	rows := 0
	for _, r := range *ttl {
		stmt, vals, err := r.insert()
		if err != nil {
			return err
		}

		// stmts = append(stmts, ins)
		res, err := db.Exec(stmt, vals...)
		if err != nil {
			return err
		}
		ra, err := res.RowsAffected()
		if err != nil {
			return err
		}
		rows += int(ra)
	}
	if rows != len(*ttl) {
		return fmt.Errorf("expect %d rows affected, got %d", len(*ttl), rows)
	}
	return nil
}

func (ttl *TTL) Render()         { panic("unimplemented") }
func (ttl *TTL) Update(DB) error { panic("unimplemented") }

// must implement
var _ mainDBtbl = (*TTL)(nil)

type ttlDesc struct {
	ID   int64
	Desc string
}
type TTLdescriptions []ttlDesc

// must implement
var _ dbTbl = (TTLdescriptions)(nil)

func (TTLdescriptions) isDbTbl() {}
