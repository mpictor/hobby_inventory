package db

import (
	"database/sql"
	"fmt"
	"iter"
	"strings"

	"github.com/jmoiron/sqlx"
	"github.com/mpictor/hobby_inventory/pkg/util"
)

// TODO one table for all, or bipolar/fet/etc separate?

type TransistorRow struct {
	commonFieldsIn
	transistorCommon
}
type transistorCommon struct {
	PN           string
	Type, Doping string // TODO enums?
	MaxV, MaxA   sql.NullFloat64
	// fT and hFE -> attrs? FIXME
	// for now, try generic:
	MaxF, Gain sql.NullFloat64

	AltPN sql.NullString
}

func fieldErr[T any](k, v string, err error) (T, error) {
	var zero T
	return zero, fmt.Errorf("field %s(%q): %w", k, v, err)
}

func qRow(db *sql.DB, rec []string) (TransistorRow, error) {
	fe := fieldErr[TransistorRow]
	var qr TransistorRow
	qr.PN = rec[0]
	q, err := ParseQty(rec[1])
	if err != nil {
		return fe("qty", rec[1], err)
	}
	qr.Qty = q
	qr.Type = rec[2]
	qr.Doping = rec[3]
	if err := parseNF(&qr.MaxV, rec[4]); err != nil {
		return fe("maxv", rec[4], err)
	}
	if err := parseNF(&qr.MaxA, rec[5]); err != nil {
		return fe("maxa", rec[5], err)
	}
	// unit is MHz in CSV
	if err := parseNF(&qr.MaxF, rec[6], 'M'); err != nil {
		return fe("maxf", rec[6], err)
	}
	if err := parseNF(&qr.Gain, rec[7]); err != nil {
		return fe("gain", rec[7], err)
	}
	qr.Package = nsFromStr(rec[8])
	qr.Notes = nsFromStr(rec[9])
	qr.descToNpkg(rec[9])
	qr.Origin = nsFromStr(rec[10])
	loc, err := auxTblVal(db, tblLocations, "name", rec[11])
	if err != nil {
		return fe("location", rec[11], err)
	}
	qr.Location = loc
	qr.AltPN = nsFromStr(rec[12])
	return qr, nil
}

func (tr TransistorRow) insert() (string, []any, error) {
	cols, vals := tr.commonFieldsIn.insert()
	cols = append(cols, "pn", "type", "doping")
	vals = append(vals, tr.PN, tr.Type, tr.Doping)
	insertNullFloat(tr.MaxV, "maxv", &cols, &vals)
	insertNullFloat(tr.MaxA, "maxa", &cols, &vals)
	insertNullFloat(tr.MaxF, "maxf", &cols, &vals)
	insertNullFloat(tr.Gain, "gain", &cols, &vals)
	insertNullStr(tr.AltPN, "altpn", &cols, &vals)
	if len(cols) == 0 {
		return "", nil, fmt.Errorf("nothing to insert for %v", tr)
	}
	ph := "?"
	ph += strings.Repeat(",?", len(vals)-1)
	s := fmt.Sprintf("INSERT INTO transistors (%s) VALUES(%s);", strings.Join(cols, ","), ph)
	return s, vals, nil
}

type Transistors []TransistorRow

func (qs *Transistors) ImportCSV(db *sql.DB, tbl string, in []byte) error {
	qq, err := importCSV(db, in, 1, *qs, qRow)
	*qs = qq
	return err

	// var qq []TransistorRow = *qs
	// r := csv.NewReader(bytes.NewReader(in))
	// // skip header
	// if _, err := r.Read(); err != nil {
	// 	return err
	// }
	// for {
	// 	record, err := r.Read()
	// 	if err == io.EOF {
	// 		break
	// 	}
	// 	if err != nil {
	// 		return err
	// 	}
	// 	if allEmpty(record) {
	// 		continue
	// 	}
	// 	if Verbose {
	// 		log.Println("csv:", record)
	// 	}
	// 	row, err := qRow(db, record)
	// 	if err != nil {
	// 		return fmt.Errorf("parsing record %v: %w", record, err)
	// 	}
	// 	*qs = append(*qs, row)
	// }
	// return nil
}

// func (qs *Transistors) ColumnHeaders(ord []int) []string { }

// store in db; db must have extant but empty table
func (qs *Transistors) Store(db *sql.DB) error {
	// TODO check that existing table is empty
	// TODO transaction?
	return qs.Insert(db)
}

// map values to columns and set up a row with the data. must subsequently call Insert or Update.
func (qs *Transistors) SetRow(db *sql.DB, kv []string) error {
	panic("unimplemented")
}

// like Store, but adds data to db that may already contain data.
func (qs *Transistors) Insert(db *sql.DB) error {
	return insert(db, *qs)
}

// update an existing row. TBD: how to specify the exact row
// func (qs *Transistors) Update(*sql.DB) error {}

func (qs Transistors) Len() int {
	return len(qs)
}

func (qs *Transistors) TableName() string { return "transistors" }

var _ mainDBtbl = (*Transistors)(nil)

type transistorOutRow struct {
	commonFieldsOut
	transistorCommon
	// TODO
	// Id   int
	// Qty  Qty
	// NPkg int

	// these currently match a subset of TransistorRow
	// PN           string
	// Type, Doping string // TODO enums?
	// MaxV, MaxA   sql.NullFloat64
	// // fT and hFE -> attrs? FIXME
	// // for now, try generic:
	// MaxF, Gain sql.NullFloat64
	// AltPN      sql.NullString

	// Desc,
	// Pkg             sql.NullString
	// Mounting        Mounting
	// Origin, Loc, DS sql.NullString
	// Attrs           sqliteBlob
	// Notes           sql.NullString
	// csv part number,qty,type,N P,V,A,"fT, MHz",hFE,pkg,note,origin,location,alt PN
}

// type transistorOutRow TransistorRow

var _ DefaultRow = (*transistorOutRow)(nil)

func (tr transistorOutRow) Strings() []string {
	strs := make([]string, 18)
	tr.commonFieldsOut.Strings(
		[]FldOrder{
			FldID, FldQty, FldNPkg,
			FldNA, FldNA, FldNA, FldNA, FldNA, FldNA, FldNA, // 3-9 are set below
			FldPkg, FldMtg, FldOrigin, FldLocation,
			FldDatasheet, FldNotes, FldAttrs,
			FldNA, // 17 is set below
		},
		strs)
	strs[3] = tr.PN
	strs[4] = tr.Type
	strs[5] = tr.Doping
	strs[6] = util.NullSI(tr.MaxV, "V", 2)
	strs[7] = util.NullSI(tr.MaxA, "A", 2)
	strs[8] = util.NullSI(tr.MaxF, "Hz", 2)
	strs[9] = util.NullSI(tr.Gain, "", 2)
	strs[17] = strFromNS(tr.AltPN)
	return strs
}

type transistorOutput []transistorOutRow

var _ dbOut = (*transistorOutput)(nil)

func (to *transistorOutput) isDbOut() {}

func (to *transistorOutput) Scan(r *sql.Rows) error {
	return sqlx.StructScan(r, to)
}

func (to *transistorOutput) All() iter.Seq[DefaultRow] {
	return func(yield func(r DefaultRow) bool) {
		for _, r := range *to {
			if !yield(r) {
				return
			}
		}
	}
}
