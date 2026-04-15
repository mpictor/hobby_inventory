package db

import (
	"database/sql"
	"fmt"
	"iter"
	"strings"

	"github.com/jmoiron/sqlx"
)

// fields very similar in other, tmr_osc_pll, and opto
// PN,category,qty,,description,location,origin,notes
type OtherRow struct {
	commonFieldsIn
	othCommon
}
type othCommon struct {
	PN, Category string
	Description  sql.NullString
}

func othRow(db *sql.DB, rec []string) (OtherRow, error) {
	var or OtherRow
	if err := or.commonFieldsIn.read(db, rec, []FldOrder{
		FldNA, FldNA, // 0-1
		FldQty,
		FldNA, FldNA, // 3-4
		FldLocation, FldOrigin, FldNotes,
	}); err != nil {
		return OtherRow{}, err
	}
	or.PN = rec[0]
	or.Category = rec[1]
	or.Description = nsFromStr(rec[4])
	return or, nil
}
func (or OtherRow) insert() (string, []any, error) {
	cols, vals := or.commonFieldsIn.insert()
	cols = append(cols, "pn", "category")
	vals = append(vals, or.PN, or.Category)
	insertNullStr(or.Description, "description", &cols, &vals)
	ph := "?"
	ph += strings.Repeat(",?", len(vals)-1)
	s := fmt.Sprintf("INSERT INTO others (%s) VALUES(%s);", strings.Join(cols, ","), ph)
	return s, vals, nil

}
func (oo OtherOutRow) Strings() []string {
	strs := make([]string, 13)
	oo.commonFieldsOut.Strings(
		[]FldOrder{
			FldID, FldQty, FldNPkg,
			FldNA, FldNA, FldNA, // 3-5 set below
			FldPkg, FldMtg, FldOrigin, FldLocation,
			FldDatasheet, FldAttrs, FldNotes,
		},
		strs)
	strs[3] = oo.PN
	strs[4] = oo.Category
	strs[5] = strFromNS(oo.Description)

	return strs
}

var otherSelect = `
SELECT t.id AS id,qty,npkg,
pn,category,description,
package,mounting,origin,locs.name AS location,datasheet,attrs,notes
FROM others AS t
LEFT JOIN locations AS locs ON t.location=locs.id
WHERE ?
ORDER BY pn ASC, qty DESC
`

type Others []OtherRow

func (os *Others) ImportCSV(db *sql.DB, tbl string, in []byte) error {
	oo, err := importCSV(db, in, 1, *os, othRow)
	*os = oo
	return err
}

// store in db; db must have extant but empty table
func (os *Others) Store(db *sql.DB) error {
	// TODO check that existing table is empty
	// TODO transaction?
	return os.Insert(db)
}

// map values to columns and set up a row with the data. must subsequently call Insert or Update.
func (os *Others) SetRow(db *sql.DB, kv []string) error {
	panic("unimplemented")
}

// like Store, but adds data to db that may already contain data.
func (os *Others) Insert(db *sql.DB) error {
	return insert(db, *os)
}

func (os Others) Len() int {
	return len(os)
}

func (os *Others) TableName() string { return "others" }

var _ mainDBtbl = (*Others)(nil)

type OtherOutRow struct {
	commonFieldsOut
	othCommon
}

var _ DefaultRow = (*OtherOutRow)(nil)

type OtherOutput []OtherOutRow

var _ dbOut = (*OtherOutput)(nil)

func (oo *OtherOutput) isDbOut() {}

func (oo *OtherOutput) Scan(r *sql.Rows) error {
	return sqlx.StructScan(r, oo)
}

func (oo *OtherOutput) All() iter.Seq[DefaultRow] {
	return func(yield func(r DefaultRow) bool) {
		for _, r := range *oo {
			if !yield(r) {
				return
			}
		}
	}
}
