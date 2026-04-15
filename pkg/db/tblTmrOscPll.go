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
// PN,category,qty,,description,origin,location
type TmrOscPllRow struct {
	commonFieldsIn
	topCommon
}
type topCommon struct {
	PN, Category string
	Description  sql.NullString
	MaxF         sql.NullFloat64 // not in csv
}

func topRow(db *sql.DB, rec []string) (TmrOscPllRow, error) {
	var or TmrOscPllRow
	if err := or.commonFieldsIn.read(db, rec, []FldOrder{
		FldNA, FldNA, // 0-1
		FldQty,
		FldNA, FldNA, // 3-4
		FldOrigin, FldLocation, FldNotes,
	}); err != nil {
		return TmrOscPllRow{}, err
	}
	or.PN = rec[0]
	or.Category = rec[1]
	or.Description = nsFromStr(rec[4])
	return or, nil
}
func (tr TmrOscPllRow) insert() (string, []any, error) {
	cols, vals := tr.commonFieldsIn.insert()
	cols = append(cols, "pn", "category")
	vals = append(vals, tr.PN, tr.Category)
	insertNullStr(tr.Description, "description", &cols, &vals)
	insertNullFloat(tr.MaxF, "maxf", &cols, &vals)
	ph := "?"
	ph += strings.Repeat(",?", len(vals)-1)
	s := fmt.Sprintf("INSERT INTO tmr_osc_pll (%s) VALUES(%s);", strings.Join(cols, ","), ph)
	return s, vals, nil

}
func (tr TmrOscPllOutRow) Strings() []string {
	strs := make([]string, 14)
	tr.commonFieldsOut.Strings(
		[]FldOrder{
			FldID, FldQty, FldNPkg,
			FldNA, FldNA, FldNA, FldNA, // 3-6 set below
			FldPkg, FldMtg, FldOrigin, FldLocation,
			FldDatasheet, FldAttrs, FldNotes,
		},
		strs)
	strs[3] = tr.PN
	strs[4] = tr.Category
	strs[5] = strFromNS(tr.Description)
	strs[6] = util.NullSI(tr.MaxF, "Hz", 2)

	return strs
}

var tmrOscPllSelect = `
SELECT t.id AS id,qty,npkg,
pn,category,description,maxf,
package,mounting,origin,locs.name AS location,datasheet,attrs,notes
FROM tmr_osc_pll AS t
LEFT JOIN locations AS locs ON t.location=locs.id
WHERE ?
ORDER BY pn ASC, qty DESC
`

type Tmr_Osc_PllDevs []TmrOscPllRow

func (ts *Tmr_Osc_PllDevs) ImportCSV(db *sql.DB, tbl string, in []byte) error {
	tt, err := importCSV(db, in, 1, *ts, topRow)
	*ts = tt
	return err
}

// store in db; db must have extant but empty table
func (ts *Tmr_Osc_PllDevs) Store(db *sql.DB) error {
	// TODO check that existing table is empty
	// TODO transaction?
	return ts.Insert(db)
}

// map values to columns and set up a row with the data. must subsequently call Insert or Update.
func (ts *Tmr_Osc_PllDevs) SetRow(db *sql.DB, kv []string) error {
	panic("unimplemented")
}

// like Store, but adds data to db that may already contain data.
func (ts *Tmr_Osc_PllDevs) Insert(db *sql.DB) error {
	return insert(db, *ts)
}

func (ts Tmr_Osc_PllDevs) Len() int {
	return len(ts)
}

func (ts *Tmr_Osc_PllDevs) TableName() string { return "tmr_osc_pll" }

var _ mainDBtbl = (*Tmr_Osc_PllDevs)(nil)

type TmrOscPllOutRow struct {
	commonFieldsOut
	topCommon
}

var _ DefaultRow = (*TmrOscPllOutRow)(nil)

type TmrOscPllOutput []TmrOscPllOutRow

var _ dbOut = (*TmrOscPllOutput)(nil)

func (to *TmrOscPllOutput) isDbOut() {}

func (to *TmrOscPllOutput) Scan(r *sql.Rows) error {
	return sqlx.StructScan(r, to)
}

func (to *TmrOscPllOutput) All() iter.Seq[DefaultRow] {
	return func(yield func(r DefaultRow) bool) {
		for _, r := range *to {
			if !yield(r) {
				return
			}
		}
	}
}
