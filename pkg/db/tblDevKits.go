package db

import (
	"database/sql"
	"fmt"
	"iter"
	"strings"

	"github.com/jmoiron/sqlx"
)

// dev kit p/n,p/n,qty,conn/skt,desc,mfg,type,origin,location,notes
type DevkitRow struct {
	commonFieldsIn
	devkitCommon
}
type devkitCommon struct {
	KitPN                    string
	DevicePN                 string
	ConnSkt, Desc, Mfg, Type string
	// TODO
}

func dkRow(db *sql.DB, rec []string) (DevkitRow, error) {
	var dk DevkitRow
	if err := dk.commonFieldsIn.read(db, rec, []FldOrder{
		FldNA, FldNA, // 0, 1
		FldQty,
		FldNA, FldNA, FldNA, FldNA, // 3-6
		FldOrigin, FldLocation, FldNotes,
	}); err != nil {
		return DevkitRow{}, err
	}
	dk.KitPN = rec[0]
	dk.DevicePN = rec[1]
	dk.ConnSkt = rec[3]
	dk.Desc = rec[4]
	dk.Mfg = rec[5]
	dk.Type = rec[6]
	return dk, nil
}
func (dk DevkitRow) insert() (string, []any, error) {
	cols, vals := dk.commonFieldsIn.insert()
	cols = append(cols, "kitpn", "devicepn", "connSkt", "desc", "mfg", "type")
	vals = append(vals, dk.KitPN, dk.DevicePN, dk.ConnSkt, dk.Desc, dk.Mfg, dk.Type)
	ph := "?"
	ph += strings.Repeat(",?", len(vals)-1)
	s := fmt.Sprintf("INSERT INTO dev_kits (%s) VALUES(%s);", strings.Join(cols, ","), ph)
	return s, vals, nil

}

type Dev_kits []DevkitRow

func (dks *Dev_kits) ImportCSV(db *sql.DB, tbl string, in []byte) error {
	qq, err := importCSV(db, in, 1, *dks, dkRow)
	*dks = qq
	return err
}

// store in db; db must have extant but empty table
func (dks *Dev_kits) Store(db *sql.DB) error {
	// TODO check that existing table is empty
	// TODO transaction?
	return dks.Insert(db)
}

// map values to columns and set up a row with the data. must subsequently call Insert or Update.
func (dks *Dev_kits) SetRow(db *sql.DB, kv []string) error {
	panic("unimplemented")
}

// like Store, but adds data to db that may already contain data.
func (dks *Dev_kits) Insert(db *sql.DB) error {
	return insert(db, *dks)
}

// update an existing row. TBD: how to specify the exact row
// func (qs *Devkits) Update(*sql.DB) error {}

func (dks Dev_kits) Len() int {
	return len(dks)
}

func (dks *Dev_kits) TableName() string { return "dev_kits" }

var _ mainDBtbl = (*Dev_kits)(nil)

type DevkitOutRow struct {
	commonFieldsOut
	devkitCommon
}

func (dk DevkitOutRow) Strings() []string {
	strs := make([]string, 16)
	/*	FldID
		FldQty
		FldNPkg
		FldPkg
		FldMtg
		FldOrigin
		FldLocation
		FldDatasheet
		FldNotes
		FldAttrs*/
	dk.commonFieldsOut.Strings(
		[]FldOrder{
			FldID, FldQty, FldNPkg,
			FldNA, FldNA, FldNA, FldNA, FldNA, FldNA, // 3-8 are set below
			FldPkg, FldMtg, FldOrigin, FldLocation,
			FldDatasheet, FldNotes, FldAttrs,
		},
		strs)
	strs[3] = dk.KitPN
	strs[4] = dk.DevicePN
	strs[5] = dk.ConnSkt
	strs[6] = dk.Desc
	strs[7] = dk.Mfg
	strs[8] = dk.Type
	return strs
}

var devkitSelect = `
SELECT t.id AS id,qty,npkg,
kitpn,devicepn,connskt,desc,mfg,type,
package,mounting,origin,locs.name AS location,datasheet,attrs,notes
FROM dev_kits AS t
LEFT JOIN locations AS locs ON t.location=locs.id
WHERE ?
ORDER BY pn ASC, qty DESC
`
var _ DefaultRow = (*DevkitOutRow)(nil)

type DevkitOutput []DevkitOutRow

var _ dbOut = (*DevkitOutput)(nil)

func (do *DevkitOutput) isDbOut() {}

func (do *DevkitOutput) Scan(r *sql.Rows) error {
	return sqlx.StructScan(r, do)
}

func (do *DevkitOutput) All() iter.Seq[DefaultRow] {
	return func(yield func(r DefaultRow) bool) {
		for _, r := range *do {
			if !yield(r) {
				return
			}
		}
	}
}
