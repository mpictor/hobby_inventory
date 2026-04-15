package db

import (
	"database/sql"
	"fmt"
	"iter"
	"strings"

	"github.com/jmoiron/sqlx"
)

// mpfx,series,fn,sfx,full pn,qty,n rx,n tx,pkg,proto,drive strength,desc,loc,
type LineDrvrRow struct {
	commonFieldsIn
	ldCommon
}
type ldCommon struct {
	Mpfx, Series, Function, Sfx string
	// skip full pn
	N_TX, N_RX           sql.NullInt64
	Proto, DriveStrength string
	Description          sql.NullString
}

func ldRow(db *sql.DB, rec []string) (LineDrvrRow, error) {
	var ld LineDrvrRow
	if err := ld.commonFieldsIn.read(db, rec, []FldOrder{
		FldNA, FldNA, FldNA, FldNA, FldNA, // 0-4
		FldQty,
		FldNA, FldNA, // 6-7
		FldPkg,
		FldNA, FldNA, FldNA, // 9-11
		FldLocation, FldNotes,
	}); err != nil {
		return LineDrvrRow{}, err
	}
	ld.Mpfx = rec[0]
	ld.Series = rec[1]
	ld.Function = rec[2]
	ld.Sfx = rec[3]
	fe := fieldErr[LineDrvrRow]
	if err := parseNI64(&ld.N_TX, rec[6]); err != nil {
		return fe("n_tx", rec[6], err)
	}
	if err := parseNI64(&ld.N_RX, rec[7]); err != nil {
		return fe("n_rx", rec[7], err)
	}
	ld.Proto = rec[9]
	ld.DriveStrength = rec[10]
	ld.Description = nsFromStr(rec[11])
	return ld, nil
}

func (ld LineDrvrRow) insert() (string, []any, error) {
	cols, vals := ld.commonFieldsIn.insert()
	cols = append(cols, "mpfx", "series", "function", "sfx")
	vals = append(vals, ld.Mpfx, ld.Series, ld.Function, ld.Sfx)
	insertNullInt64(ld.N_TX, "n_tx", &cols, &vals)
	insertNullInt64(ld.N_RX, "n_rx", &cols, &vals)
	cols = append(cols, "proto", "driveStrength")
	vals = append(vals, ld.Proto, ld.DriveStrength)
	insertNullStr(ld.Description, "description", &cols, &vals)
	ph := "?"
	ph += strings.Repeat(",?", len(vals)-1)
	s := fmt.Sprintf("INSERT INTO line_drv (%s) VALUES(%s);", strings.Join(cols, ","), ph)
	return s, vals, nil
}
func (ld LineDrvrOutRow) Strings() []string {
	strs := make([]string, 19)
	ld.commonFieldsOut.Strings(
		[]FldOrder{
			FldID, FldQty, FldNPkg,
			FldNA, FldNA, FldNA, FldNA, FldNA, // 3-11 are set below
			FldNA, FldNA, FldNA, FldNA,
			FldPkg, FldMtg, FldOrigin, FldLocation,
			FldDatasheet, FldNotes, FldAttrs,
		},
		strs)
	strs[3] = ld.Mpfx
	strs[4] = ld.Series
	strs[5] = ld.Function
	strs[6] = ld.Sfx
	strs[7] = strFromNI64(ld.N_TX)
	strs[8] = strFromNI64(ld.N_RX)
	strs[9] = ld.Proto
	strs[10] = ld.DriveStrength
	strs[11] = strFromNS(ld.Description)
	return strs
}

var linedrvSelect = `
SELECT t.id AS id,qty,npkg,
mpfx,series,function,sfx,n_tx,n_rx,proto,driveStrength,description,
package,mounting,origin,locs.name AS location,datasheet,attrs,notes
FROM line_drv AS t
LEFT JOIN locations AS locs ON t.location=locs.id
WHERE ?
ORDER BY pn ASC, qty DESC
`

type Line_Drv []LineDrvrRow

func (lds *Line_Drv) ImportCSV(db *sql.DB, tbl string, in []byte) error {
	ll, err := importCSV(db, in, 1, *lds, ldRow)
	*lds = ll
	return err
}

// store in db; db must have extant but empty table
func (lds *Line_Drv) Store(db *sql.DB) error {
	// TODO check that existing table is empty
	// TODO transaction?
	return lds.Insert(db)
}

// map values to columns and set up a row with the data. must subsequently call Insert or Update.
func (lds *Line_Drv) SetRow(db *sql.DB, kv []string) error {
	panic("unimplemented")
}

// like Store, but adds data to db that may already contain data.
func (lds *Line_Drv) Insert(db *sql.DB) error {
	return insert(db, *lds)
}

func (lds Line_Drv) Len() int {
	return len(lds)
}

func (lds *Line_Drv) TableName() string { return "line_drv" }

var _ mainDBtbl = (*Line_Drv)(nil)

type LineDrvrOutRow struct {
	commonFieldsOut
	ldCommon
}

var _ DefaultRow = (*LineDrvrOutRow)(nil)

type LineDrvrOutput []LineDrvrOutRow

var _ dbOut = (*LineDrvrOutput)(nil)

func (ld *LineDrvrOutput) isDbOut() {}

func (ld *LineDrvrOutput) Scan(r *sql.Rows) error {
	return sqlx.StructScan(r, ld)
}

func (ld *LineDrvrOutput) All() iter.Seq[DefaultRow] {
	return func(yield func(r DefaultRow) bool) {
		for _, r := range *ld {
			if !yield(r) {
				return
			}
		}
	}
}
