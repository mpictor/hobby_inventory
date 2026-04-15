package db

import (
	"database/sql"
	"fmt"
	"iter"
	"strings"

	"github.com/jmoiron/sqlx"
	"github.com/mpictor/hobby_inventory/pkg/util"
)

// PN,category,qty,V,A,pkg,description,location,
type PowerDevice struct {
	commonFieldsIn
	PN          string
	Category    sql.NullString
	MaxV, MaxA  sql.NullFloat64
	Description sql.NullString
}

func powerRow(db *sql.DB, rec []string) (PowerDevice, error) {
	fe := fieldErr[PowerDevice]
	var pr, zero PowerDevice
	pr.PN = rec[0]
	pr.Category = nsFromStr(rec[1])
	q, err := ParseQty(rec[2])
	if err != nil {
		return zero, err
	}
	pr.Qty = q
	if err := parseNF(&pr.MaxV, rec[3]); err != nil {
		return fe("maxv", rec[3], err)
	}
	if err := parseNF(&pr.MaxA, rec[4]); err != nil {
		return fe("maxa", rec[4], err)
	}
	pr.Package = nsFromStr(rec[5])
	// TODO NPkg from description
	pr.Description = nsFromStr(rec[6])
	loc, err := auxTblVal(db, tblLocations, "name", rec[7])
	if err != nil {
		return fe("location", rec[7], err)
	}
	pr.Location = loc
	return pr, nil
}

func (pr PowerDevice) insert() (string, []any, error) {
	cols, vals := pr.commonFieldsIn.insert()
	cols = append(cols, "pn")
	vals = append(vals, pr.PN)
	insertNullStr(pr.Category, "category", &cols, &vals)
	insertNullFloat(pr.MaxV, "maxv", &cols, &vals)
	insertNullFloat(pr.MaxA, "maxa", &cols, &vals)
	insertNullStr(pr.Description, "description", &cols, &vals)
	if len(cols) == 0 {
		return "", nil, fmt.Errorf("nothing to insert for %v", pr)
	}
	ph := "?"
	ph += strings.Repeat(",?", len(vals)-1)
	s := fmt.Sprintf("INSERT INTO power (%s) VALUES(%s);", strings.Join(cols, ","), ph)
	return s, vals, nil
}

type PowerDevs []PowerDevice

func (ps *PowerDevs) TableName() string { return "power" }

var _ mainDBtbl = (*PowerDevs)(nil)

func (ps *PowerDevs) ImportCSV(db *sql.DB, tbl string, in []byte) error {
	pp, err := importCSV(db, in, 1, *ps, powerRow)
	*ps = pp
	return err
}

// store in db; db must have extant but empty table
func (ps *PowerDevs) Store(db *sql.DB) error {
	if err := checkEmpty(db, ps); err != nil {
		return err
	}
	return ps.Insert(db)
}

// map values to columns and set up a row with the data. must subsequently call Insert or Update.
func (ps *PowerDevs) SetRow(db *sql.DB, kv []string) error {
	panic("unimplemented")
}

// like Store, but adds data to db that may already contain data
func (ps *PowerDevs) Insert(db *sql.DB) error {
	return insert(db, *ps)
}

func (ps *PowerDevs) Len() int { return len(*ps) }

type PowerRowOut struct {
	commonFieldsOut
	PN          string
	Category    sql.NullString
	MaxV, MaxA  sql.NullFloat64
	Description sql.NullString
}

var _ DefaultRow = (*PowerRowOut)(nil)

func (pr PowerRowOut) Strings() []string {
	strs := make([]string, 15)
	pr.commonFieldsOut.Strings(
		[]FldOrder{
			FldID, FldQty, FldNPkg,
			FldNA, FldNA, FldNA, FldNA, FldNA, // 3-7 are set below
			FldPkg, FldMtg, FldOrigin, FldLocation,
			FldDatasheet, FldAttrs, FldNotes,
		},
		strs)

	strs[3] = pr.PN
	strs[4] = strFromNS(pr.Category)
	strs[5] = util.NullSI(pr.MaxV, "V", 2)
	strs[6] = util.NullSI(pr.MaxA, "A", 2)
	strs[7] = strFromNS(pr.Description)
	return strs
}

var powerSelect = `
SELECT t.id AS id,qty,npkg,
pn,category,maxv,maxa,description,
package,mounting,origin,locs.name AS location,datasheet,attrs,notes
FROM power AS t
LEFT JOIN locations AS locs ON t.location=locs.id
WHERE ?
ORDER BY category,maxv DESC,maxa DESC, qty DESC
`

type PowerOutput []PowerRowOut

var _ dbOut = (*PowerOutput)(nil)

func (po *PowerOutput) isDbOut() {}

func (po *PowerOutput) Scan(r *sql.Rows) error {
	return sqlx.StructScan(r, po)
}

func (po *PowerOutput) All() iter.Seq[DefaultRow] {
	return func(yield func(r DefaultRow) bool) {
		for _, r := range *po {
			if !yield(r) {
				return
			}
		}
	}
}
