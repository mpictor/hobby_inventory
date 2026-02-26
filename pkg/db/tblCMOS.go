package db

import (
	"database/sql"
)

// for ttlCreate and cmos, description will be unique to family+function
// put in separate table?
// 4xxx and other wide-voltage-range CMOS
// const cmosCreate = `
// CREATE TABLE cmos(
// 	id integer,
// 	mpfx bpchar,
// 	-- ord,
// 	series bpchar, -- really not int?
// 	func bpchar, -- really not int?
// 	sfx bpchar,
// 	-- PN,
// 	category bpchar,
// 	qty integer,
// 	description bpchar,
// 	location integer,
// 	interesting bpchar,
// 	Motorola_1978 bpchar,
// 	comments bpchar
// );
// `

type CMOS struct {
	commonFields
	Mpfx        sql.NullString
	Series      sql.NullString
	Func        sql.NullString
	Sfx         sql.NullString
	Category    sql.NullString
	Description sql.NullInt64 `db:"description,FK:cmosDescription:id"` // foreign key
	Interesting sql.NullString
	Moto1978    sql.NullString
}

func (*CMOS) isDbTbl() {}
func (cmos *CMOS) ImportCSV(db DB, csv []byte) error {
	// TODO
	panic("unimplemented")
}
func (cmos *CMOS) Store(db DB) error {
	// TODO check that existing table is empty
	// TODO compose INSERT statements
	panic("unimplemented")
}
func (cmos *CMOS) ColumnHeaders() ([]string, error) { panic("unimplemented") }
func (cmos *CMOS) Insert(DB) error                  { panic("unimplemented") }
func (cmos *CMOS) Render()                          { panic("unimplemented") }
func (cmos *CMOS) Update(DB) error                  { panic("unimplemented") }

// must implement
var _ mainDBtbl = (*CMOS)(nil)

type cmosDesc struct {
	ID   int64
	Desc string
}
type CMOSdescriptions []cmosDesc

// must implement
var _ dbTbl = (CMOSdescriptions)(nil)

func (CMOSdescriptions) isDbTbl() {}
