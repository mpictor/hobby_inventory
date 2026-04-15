package db

import (
	"database/sql"
	"fmt"
	"iter"
	"strings"

	"github.com/jmoiron/sqlx"
	"github.com/mpictor/hobby_inventory/pkg/util"
)

// fields very similar in other, tmr_osc_pll, and opto
// PN,category,qty,,description,location
type OptoRow struct {
	commonFieldsIn
	optCommon
}
type optCommon struct {
	PN, Category string
	Description  sql.NullString
	MaxF         sql.NullFloat64 // not present in csv
}

// FIXME FIXME check for extra records at end of csv line - all tables!
func optRow(db *sql.DB, rec []string) (OptoRow, error) {
	var or OptoRow
	if err := or.commonFieldsIn.read(db, rec, []FldOrder{
		FldNA, FldNA, // 0-1
		FldQty,
		FldNA, FldNA, // 3-4
		FldLocation, FldNotes,
	}); err != nil {
		return OptoRow{}, err
	}
	or.PN = rec[0]
	or.Category = rec[1]
	or.Description = nsFromStr(rec[4])
	return or, nil
}
func (or OptoRow) insert() (string, []any, error) {
	cols, vals := or.commonFieldsIn.insert()
	cols = append(cols, "pn", "category")
	vals = append(vals, or.PN, or.Category)
	insertNullStr(or.Description, "description", &cols, &vals)
	insertNullFloat(or.MaxF, "maxf", &cols, &vals)
	ph := "?"
	ph += strings.Repeat(",?", len(vals)-1)
	s := fmt.Sprintf("INSERT INTO opto (%s) VALUES(%s);", strings.Join(cols, ","), ph)
	return s, vals, nil

}
func (oo OptoOutRow) Strings() []string {
	strs := make([]string, 14)
	oo.commonFieldsOut.Strings(
		[]FldOrder{
			FldID, FldQty, FldNPkg,
			FldNA, FldNA, FldNA, FldNA, // 3-6 set below
			FldPkg, FldMtg, FldOrigin, FldLocation,
			FldDatasheet, FldAttrs, FldNotes,
		},
		strs)
	strs[3] = oo.PN
	strs[4] = oo.Category
	strs[5] = strFromNS(oo.Description)
	strs[6] = util.NullSI(oo.MaxF, "Hz", 2)

	return strs
}

var optoSelect = `
SELECT t.id AS id,qty,npkg,
pn,category,description,maxf,
package,mounting,origin,locs.name AS location,datasheet,attrs,notes
FROM opto AS t
LEFT JOIN locations AS locs ON t.location=locs.id
WHERE ?
ORDER BY pn ASC, qty DESC
`

type OptoDevs []OptoRow

func (ods *OptoDevs) ImportCSV(db *sql.DB, tbl string, in []byte) error {
	os, err := importCSV(db, in, 1, *ods, optRow)
	*ods = os
	return err
}

// store in db; db must have extant but empty table
func (ods *OptoDevs) Store(db *sql.DB) error {
	// TODO check that existing table is empty
	// TODO transaction?
	return ods.Insert(db)
}

// map values to columns and set up a row with the data. must subsequently call Insert or Update.
func (ods *OptoDevs) SetRow(db *sql.DB, kv []string) error {
	panic("unimplemented")
}

// like Store, but adds data to db that may already contain data.
func (ods *OptoDevs) Insert(db *sql.DB) error {
	return insert(db, *ods)
}

func (ods OptoDevs) Len() int {
	return len(ods)
}

func (ods *OptoDevs) TableName() string { return "opto" }

var _ mainDBtbl = (*OptoDevs)(nil)

type OptoOutRow struct {
	commonFieldsOut
	optCommon
}

var _ DefaultRow = (*OptoOutRow)(nil)

type OptoOutput []OptoOutRow

var _ dbOut = (*OptoOutput)(nil)

func (oo *OptoOutput) isDbOut() {}

func (oo *OptoOutput) Scan(r *sql.Rows) error {
	return sqlx.StructScan(r, oo)
}

func (oo *OptoOutput) All() iter.Seq[DefaultRow] {
	return func(yield func(r DefaultRow) bool) {
		for _, r := range *oo {
			if !yield(r) {
				return
			}
		}
	}
}
