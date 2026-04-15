package db

import (
	"database/sql"
	"fmt"
	"iter"
	"strings"

	"github.com/jmoiron/sqlx"
	"github.com/mpictor/hobby_inventory/pkg/util"
)

// PN,category,class,n/pkg,qty,"Gbp, MHz","slew, V/us",pkg,min Vcc,R-R?,description,origin,location,
type OpampRow struct {
	commonFieldsIn
	opaCommon
}
type opaCommon struct {
	PN, Category, Class        string
	GainBandwidthProduct, Slew sql.NullFloat64
	MinVcc                     sql.NullFloat64
	RailRail                   sql.NullString // TODO enum? don't think bool will work
	Description                sql.NullString
	// TODO
}

func opaRow(db *sql.DB, rec []string) (OpampRow, error) {
	var or OpampRow
	if err := or.commonFieldsIn.read(db, rec, []FldOrder{
		FldNA, FldNA, FldNA, // 0-2
		FldNPkg, FldQty,
		FldNA, FldNA, // 5-6
		FldPkg,
		FldNA, FldNA, FldNA, // 8-10
		FldOrigin, FldLocation, FldNotes,
	}); err != nil {
		return OpampRow{}, err
	}
	or.PN = rec[0]
	or.Category = rec[1]
	or.Class = rec[2]
	fe := fieldErr[OpampRow]
	if err := parseNF(&or.GainBandwidthProduct, rec[5]); err != nil {
		return fe("gbp", rec[5], err)
	}
	if err := parseNF(&or.Slew, rec[6]); err != nil {
		return fe("slew", rec[6], err)
	}
	if err := parseNF(&or.MinVcc, rec[8]); err != nil {
		return fe("minVcc", rec[8], err)
	}
	or.RailRail = nsFromStr(rec[9])
	or.Description = nsFromStr(rec[10])
	return or, nil
}
func (or OpampRow) insert() (string, []any, error) {
	cols, vals := or.commonFieldsIn.insert()
	cols = append(cols, "pn", "category", "class")
	vals = append(vals, or.PN, or.Category, or.Class)
	insertNullFloat(or.GainBandwidthProduct, "gainBandwidthProduct", &cols, &vals)
	insertNullFloat(or.Slew, "slew", &cols, &vals)
	insertNullFloat(or.MinVcc, "minVcc", &cols, &vals)
	insertNullStr(or.RailRail, "railRail", &cols, &vals)
	insertNullStr(or.Description, "description", &cols, &vals)
	ph := "?"
	ph += strings.Repeat(",?", len(vals)-1)
	s := fmt.Sprintf("INSERT INTO opamps (%s) VALUES(%s);", strings.Join(cols, ","), ph)
	return s, vals, nil

}
func (oo OpampOutRow) Strings() []string {
	strs := make([]string, 18)
	oo.commonFieldsOut.Strings(
		[]FldOrder{
			FldID, FldQty, FldNPkg,
			FldNA, FldNA, FldNA, FldNA, // 3-10 are set below
			FldNA, FldNA, FldNA, FldNA,
			FldPkg, FldMtg, FldOrigin, FldLocation,
			FldDatasheet, FldAttrs, FldNotes,
		},
		strs)
	strs[3] = oo.PN
	strs[4] = oo.Category
	strs[5] = oo.Class
	strs[6] = util.NullSI(oo.GainBandwidthProduct, "", 2)
	strs[7] = util.NullSI(oo.Slew, "", 2)
	strs[8] = util.NullSI(oo.MinVcc, "V", 2)
	strs[9] = strFromNS(oo.RailRail)
	strs[10] = strFromNS(oo.Description)
	return strs
}

var opampSelect = `
SELECT t.id AS id,qty,npkg,
pn,category,class,gainbandwidthproduct,slew,minvcc,railrail,description,
package,mounting,origin,locs.name AS location,datasheet,attrs,notes
FROM opamps AS t
LEFT JOIN locations AS locs ON t.location=locs.id
WHERE ?
ORDER BY pn ASC, qty DESC
`

type Opamps []OpampRow

func (ops *Opamps) ImportCSV(db *sql.DB, tbl string, in []byte) error {
	oa, err := importCSV(db, in, 1, *ops, opaRow)
	*ops = oa
	return err
}

// store in db; db must have extant but empty table
func (ops *Opamps) Store(db *sql.DB) error {
	// TODO check that existing table is empty
	// TODO transaction?
	return ops.Insert(db)
}

// map values to columns and set up a row with the data. must subsequently call Insert or Update.
func (ops *Opamps) SetRow(db *sql.DB, kv []string) error {
	panic("unimplemented")
}

// like Store, but adds data to db that may already contain data.
func (ops *Opamps) Insert(db *sql.DB) error {
	return insert(db, *ops)
}

func (ops Opamps) Len() int {
	return len(ops)
}

func (ops *Opamps) TableName() string { return "opamps" }

var _ mainDBtbl = (*Opamps)(nil)

type OpampOutRow struct {
	commonFieldsOut
	opaCommon
}

var _ DefaultRow = (*OpampOutRow)(nil)

type OpampOutput []OpampOutRow

var _ dbOut = (*OpampOutput)(nil)

func (oo *OpampOutput) isDbOut() {}

func (oo *OpampOutput) Scan(r *sql.Rows) error {
	return sqlx.StructScan(r, oo)
}

func (oo *OpampOutput) All() iter.Seq[DefaultRow] {
	return func(yield func(r DefaultRow) bool) {
		for _, r := range *oo {
			if !yield(r) {
				return
			}
		}
	}
}
