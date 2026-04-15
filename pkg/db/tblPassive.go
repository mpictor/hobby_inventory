package db

import (
	"database/sql"
	"fmt"
	"iter"
	"strings"

	"github.com/jmoiron/sqlx"
)

// PN,FUNC,TYP,QTY,value,rating,NOTE,FORM,F FACT,STORAGE,LOC,
type PassiveRow struct {
	commonFieldsIn
	pasCommon
}
type pasCommon struct {
	PN, Function, Type, Value, Rating string
	// Form -> mounting, FormFactor -> package
	Storage string
	// TODO
}

func pRow(db *sql.DB, rec []string) (PassiveRow, error) {
	var pr PassiveRow
	if len(rec) > 0 && rec[0] == "Resistor 10:1 atten" {
		return pr, ErrImportDone
	}
	if err := pr.commonFieldsIn.read(db, rec, []FldOrder{
		FldNA, FldNA, FldNA, // 0-2
		FldQty,
		FldNA, FldNA, // 4-5
		FldNotes, FldMtg, FldPkg,
		FldNA, // 9
		FldLocation, FldNotes,
	}); err != nil {
		return PassiveRow{}, err
	}
	pr.PN = rec[0]
	pr.Function = rec[1]
	pr.Type = rec[2]
	pr.Value = rec[4]
	pr.Rating = rec[5]
	pr.Storage = rec[9]
	return pr, nil
}
func (pr PassiveRow) insert() (string, []any, error) {
	cols, vals := pr.commonFieldsIn.insert()
	cols = append(cols, "pn", "function", "type", "value", "rating", "storage")
	vals = append(vals, pr.PN, pr.Function, pr.Type, pr.Value, pr.Rating, pr.Storage)
	ph := "?"
	ph += strings.Repeat(",?", len(vals)-1)
	s := fmt.Sprintf("INSERT INTO passive (%s) VALUES(%s);", strings.Join(cols, ","), ph)
	return s, vals, nil
}
func (pr PassiveOutRow) Strings() []string {
	strs := make([]string, 16)
	pr.commonFieldsOut.Strings(
		[]FldOrder{
			FldID, FldQty, FldNPkg,
			FldNA, FldNA, FldNA, // 3-8 set below
			FldNA, FldNA, FldNA,
			FldPkg, FldMtg, FldOrigin,
			FldLocation, FldDatasheet,
			FldAttrs, FldNotes,
		},
		strs)
	strs[3] = pr.PN
	strs[4] = pr.Function
	strs[5] = pr.Type
	strs[6] = pr.Value
	strs[7] = pr.Rating
	strs[8] = pr.Storage

	return strs
}

var passiveSelect = `
SELECT t.id AS id,qty,npkg,
pn,function,type,value,rating,storage,
package,mounting,origin,locs.name AS location,datasheet,attrs,notes
FROM passive AS t
LEFT JOIN locations AS locs ON t.location=locs.id
WHERE ?
ORDER BY pn ASC, qty DESC
`

type PassiveDevs []PassiveRow

func (pds *PassiveDevs) ImportCSV(db *sql.DB, _ string, in []byte) error {
	pp, err := importCSV(db, in, 1, *pds, pRow)
	*pds = pp
	return err
}

// store in db; db must have extant but empty table
func (pds *PassiveDevs) Store(db *sql.DB) error {
	// TODO check that existing table is empty
	// TODO transaction?
	return pds.Insert(db)
}

// map values to columns and set up a row with the data. must subsequently call Insert or Update.
func (pds *PassiveDevs) SetRow(db *sql.DB, kv []string) error {
	panic("unimplemented")
}

// like Store, but adds data to db that may already contain data.
func (pds *PassiveDevs) Insert(db *sql.DB) error {
	return insert(db, *pds)
}

func (pds PassiveDevs) Len() int {
	return len(pds)
}

func (pds *PassiveDevs) TableName() string { return "passive" }

var _ mainDBtbl = (*PassiveDevs)(nil)

type PassiveOutRow struct {
	commonFieldsOut
	pasCommon
}

var _ DefaultRow = (*PassiveOutRow)(nil)

type PassiveOutput []PassiveOutRow

var _ dbOut = (*PassiveOutput)(nil)

func (po *PassiveOutput) isDbOut() {}

func (po *PassiveOutput) Scan(r *sql.Rows) error {
	return sqlx.StructScan(r, po)
}

func (po *PassiveOutput) All() iter.Seq[DefaultRow] {
	return func(yield func(r DefaultRow) bool) {
		for _, r := range *po {
			if !yield(r) {
				return
			}
		}
	}
}
