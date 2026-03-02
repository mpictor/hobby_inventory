package db

import (
	"database/sql"
	"iter"
)

type CMOSRow struct {
	commonFields
	Prefix      sql.NullString
	Series      sql.NullString
	Func        sql.NullString
	Sfx         sql.NullString
	Category    sql.NullString
	Description sql.NullInt64 `db:"description,FK:cmosDescriptions:id"` // foreign key
	Interesting sql.NullString
	Moto1978    sql.NullString
}
type CMOS []CMOSRow

func (*CMOS) isDbTbl() {}

func (cmos *CMOS) ImportCSV(db *sql.DB, csv []byte) error {
	// TODO
	panic("unimplemented")
}
func (cmos *CMOS) Store(db *sql.DB) error {
	// TODO check that existing table is empty
	// TODO compose INSERT statements
	panic("unimplemented")
}
func (cmos *CMOS) ColumnHeaders() []string { panic("unimplemented") }
func (cmos *CMOS) Insert(*sql.DB) error    { panic("unimplemented") }

// func (cmos *CMOS) Render()                              { panic("unimplemented") }
func (cmos *CMOS) Update(*sql.DB) error                 { panic("unimplemented") }
func (cmos *CMOS) SetRow(db *sql.DB, kv []string) error { panic("unimplemented") }
func (cmos CMOS) All() iter.Seq[[]string]               { panic("unimplemented") }
func (cmos CMOS) Len() int                              { return len(cmos) }

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
