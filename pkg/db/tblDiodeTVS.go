package db

import (
	"database/sql"
	"fmt"
	"iter"
	"strings"

	"github.com/jmoiron/sqlx"
	"github.com/mpictor/hobby_inventory/pkg/util"
)

// "Diodes, TVS, surge",,,,,,,,,,
// PN,category,qty,,V,I,"trr, ns",pkg,description,origin,location
type DiodeTVSRow struct {
	commonFieldsIn
	diodeTVScommon
}
type diodeTVScommon struct {
	PN, Category             string
	MaxV, MaxA, RecoveryTime sql.NullFloat64
	Description              sql.NullString
}

func dtRow(db *sql.DB, rec []string) (DiodeTVSRow, error) {
	var dt DiodeTVSRow
	if err := dt.commonFieldsIn.read(db, rec, []FldOrder{
		FldNA, FldNA, // 0, 1
		FldQty,
		FldNA, FldNA, FldNA, FldNA, // 3-6
		FldPkg,
		FldNA, // 8
		FldOrigin, FldLocation, FldNotes,
	}); err != nil {
		return DiodeTVSRow{}, err
	}
	dt.PN = rec[0]
	dt.Category = rec[1]
	fe := fieldErr[DiodeTVSRow]
	// skip 3
	if err := parseNF(&dt.MaxV, rec[4]); err != nil {
		return fe("maxv", rec[4], err)
	}
	if err := parseNF(&dt.MaxA, rec[5]); err != nil {
		return fe("maxa", rec[5], err)
	}
	if err := parseNF(&dt.RecoveryTime, rec[6]); err != nil {
		return fe("trr", rec[6], err)
	}
	dt.Description = nsFromStr(rec[8])
	return dt, nil
}
func (dt DiodeTVSRow) insert() (string, []any, error) {
	cols, vals := dt.commonFieldsIn.insert()
	cols = append(cols, "pn", "category")
	vals = append(vals, dt.PN, dt.Category)
	insertNullFloat(dt.MaxV, "maxv", &cols, &vals)
	insertNullFloat(dt.MaxA, "maxa", &cols, &vals)
	insertNullFloat(dt.RecoveryTime, "recoveryTime", &cols, &vals)
	insertNullStr(dt.Description, "description", &cols, &vals)
	ph := "?"
	ph += strings.Repeat(",?", len(vals)-1)
	s := fmt.Sprintf("INSERT INTO diode_tvs (%s) VALUES(%s);", strings.Join(cols, ","), ph)
	return s, vals, nil
}

type Diode_TVSDevs []DiodeTVSRow

func (dts *Diode_TVSDevs) ImportCSV(db *sql.DB, tbl string, in []byte) error {
	d, err := importCSV(db, in, 2, *dts, dtRow)
	*dts = d
	return err
}

// store in db; db must have extant but empty table
func (dts *Diode_TVSDevs) Store(db *sql.DB) error {
	// TODO check that existing table is empty
	// TODO transaction?
	return dts.Insert(db)
}

// map values to columns and set up a row with the data. must subsequently call Insert or Update.
func (dts *Diode_TVSDevs) SetRow(db *sql.DB, kv []string) error {
	panic("unimplemented")
}

// like Store, but adds data to db that may already contain data.
func (dts *Diode_TVSDevs) Insert(db *sql.DB) error {
	return insert(db, *dts)
}

func (dts Diode_TVSDevs) Len() int {
	return len(dts)
}

func (dts *Diode_TVSDevs) TableName() string { return "diode_tvs" }

var _ mainDBtbl = (*Diode_TVSDevs)(nil)

type DiodeTVSOutRow struct {
	commonFieldsOut
	diodeTVScommon
}

var _ DefaultRow = (*DiodeTVSOutRow)(nil)

func (dt DiodeTVSOutRow) Strings() []string {
	strs := make([]string, 16)
	dt.commonFieldsOut.Strings(
		[]FldOrder{
			FldID, FldQty, FldNPkg,
			FldNA, FldNA, FldNA, FldNA, FldNA, FldNA, // 3-8 are set below
			FldPkg, FldMtg, FldOrigin, FldLocation,
			FldDatasheet, FldNotes, FldAttrs,
		},
		strs)
	strs[3] = dt.PN
	strs[4] = dt.Category
	strs[5] = util.NullSI(dt.MaxV, "V", 2)
	strs[6] = util.NullSI(dt.MaxA, "A", 2)
	strs[7] = util.NullSI(dt.RecoveryTime, "s", 2)
	strs[8] = strFromNS(dt.Description)
	return strs
}

var diodetvsSelect = `
SELECT t.id AS id,qty,npkg,
pn,category,maxv,maxa,recoverytime,description,
package,mounting,origin,locs.name AS location,datasheet,attrs,notes
FROM diode_tvs AS t
LEFT JOIN locations AS locs ON t.location=locs.id
WHERE ?
ORDER BY pn ASC, qty DESC
`

type DiodeTVSOutput []DiodeTVSOutRow

var _ dbOut = (*DiodeTVSOutput)(nil)

func (dt *DiodeTVSOutput) isDbOut() {}

func (dt *DiodeTVSOutput) Scan(r *sql.Rows) error {
	return sqlx.StructScan(r, dt)
}

func (dt *DiodeTVSOutput) All() iter.Seq[DefaultRow] {
	return func(yield func(r DefaultRow) bool) {
		for _, r := range *dt {
			if !yield(r) {
				return
			}
		}
	}
}
