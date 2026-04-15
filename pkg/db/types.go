package db

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"strconv"
	"strings"
	"unicode"

	"github.com/dustin/go-humanize"
)

// fields common to all
type commonFieldsIn struct {
	commonCommon
	Location int64 `db:"location,FK:locations:id"` // maps to db table locations
}

// everything common for input and output - location changes type due to table lookup
type commonCommon struct {
	ID        int64          // row id in db
	Qty       Qty            // quantity
	NPkg      int64          // number per package, typically 1
	Package   sql.NullString // TO-220, DIP-14, SOIC-8, etc
	Mounting  Mounting       // through hole, smt, etc
	Origin    sql.NullString // freeform, where I got it from - digikey, estate, etc
	Datasheet sql.NullString // local file name, or databook and page
	Attrs     sqliteBlob     // additional attrs, json-encoded
	Notes     sql.NullString
	// TODO Description, Fmax
	// Description - not in common, as multiple TTL or CMOS will share one description
}

// same as commonFieldsIn except for location's type
type commonFieldsOut struct {
	commonCommon
	Location string // from db table locations
}

func (c commonFieldsIn) insert() (cols []string, vals []any) {
	// always skip the id?
	cols = append(cols, "qty")
	vals = append(vals, int64(c.Qty))
	cols = append(cols, "npkg", "mounting", "location")
	vals = append(vals, c.NPkg, int64(c.Mounting), c.Location)
	insertNullStr(c.Package, "package", &cols, &vals)
	insertNullStr(c.Origin, "origin", &cols, &vals)
	insertNullStr(c.Datasheet, "datasheet", &cols, &vals)
	if c.Attrs != nil {
		cols = append(cols, "attrs")
		vals = append(vals, c.Attrs.String())
	}
	insertNullStr(c.Notes, "notes", &cols, &vals)

	return cols, vals
}

// set commonFieldsIn from csv data, using fields specified in ord
func (c *commonFieldsIn) read(db *sql.DB, rec []string, ord []FldOrder) error {
	reqLen := len(ord)
	if ord[reqLen-1] == FldNotes {
		// optional
		reqLen--
	}
	if reqLen > len(rec) {
		return fmt.Errorf("\nord size %d (%v) exceeds\nrec size %d (%v)", len(ord), ord, len(rec), rec)
	}
	fe := func(idx int, fld FldOrder, err error) error {
		return fmt.Errorf("idx %d (%s) in rec %v: %w", idx, fld, rec, err)
	}
	for i, o := range ord {
		if i == len(rec) {
			break
		}
		switch o {
		case FldID:
			return fmt.Errorf("not valid to read ID field in: pos %d in %v", i, ord)
		case FldQty:
			q, err := ParseQty(rec[i])
			if err != nil {
				return fe(i, o, err)
			}
			c.Qty = q
		case FldNPkg:
			n, err := strconv.ParseInt(rec[i], 10, 64)
			if err != nil {
				return fe(i, o, err)
			}
			c.NPkg = n
		case FldPkg:
			c.Package = nsFromStr(rec[i])
		case FldMtg:
			if err := c.Mounting.Parse(rec[i]); err != nil {
				return fe(i, o, err)
			}
		case FldOrigin:
			c.Origin = nsFromStr(rec[i])
		case FldLocation:
			loc, err := auxTblVal(db, tblLocations, "name", rec[i])
			if err != nil {
				return fe(i, o, err)
			}
			c.Location = loc
		case FldDatasheet:
			c.Datasheet = nsFromStr(rec[i])
		case FldNotes:
			c.Notes = nsJoin(c.Notes, ";", rec[i])
		case FldAttrs:
			if err := c.Attrs.Scan(rec[i]); err != nil {
				return fe(i, o, err)
			}
		case FldNA:
			// ignore
		case FldDescription:
			c.descToNpkg(rec[i])
		default:
			return fmt.Errorf("field ord[%d]: unknown enum value %d", i, o)
		}
	}
	return nil
}

// set common fields, returning remaining fields
// attempt to standardize keys and values since they are human input
func (c *commonFieldsIn) setParams(db *sql.DB, m paramMap) (paramMap, error) {
	remain := make(paramMap)
	for k, v := range m {
		if v.Op != EQ {
			return nil, fmt.Errorf("expected operator EQ, got %s", v.Op)
		}
		switch k {
		case "qty":
			q, err := ParseQty(v.Val)
			if err != nil {
				return nil, err
			}
			c.Qty = q
		case "npkg":
			n, err := strconv.ParseInt(v.Val, 10, 64)
			if err != nil {
				return nil, err
			}
			c.NPkg = n
		case "package":
			if err := c.setPkg(v.Val); err != nil {
				return nil, err
			}
		case "mounting":
			if err := c.Mounting.Parse(v.Val); err != nil {
				return nil, err
			}
		case "origin":
			c.Origin = sql.NullString{Valid: true, String: v.Val}
		case "location":
			n, err := auxTblVal(db, tblLocations, "name", v.Val)
			if err != nil {
				return nil, fmt.Errorf("setting location: %w", err)
			}
			c.Location = n
		case "datasheet":
			c.Datasheet = sql.NullString{Valid: true, String: v.Val}
		case "notes":
			c.Notes = sql.NullString{Valid: true, String: v.Val}
		default:
			remain[k] = v
		}
	}
	return remain, nil
}

// dual, quad, hex, octal, etc -> n/pkg
func (cf *commonFieldsIn) descToNpkg(desc string) {
	dl := strings.ToLower(desc)
	nums := []string{"zero_placeholder", "single", "dual", "triple", "quad", "penta", "hex", "septa", "octal"}
	for i, n := range nums {
		if i == 0 {
			// placeholder value to keep indices aligned and eliminate confusion
			continue
		}
		if strings.Contains(dl, n) {
			if cf.NPkg > 0 && cf.NPkg != int64(i) {
				log.Printf("WARN: npkg prev=%d new=%d from desc %q", cf.NPkg, i, desc)
			}
			cf.NPkg = int64(i)
		}
	}
	if cf.NPkg == 0 {
		cf.NPkg = 1
	}
}

// takes a null string and returns a string
// may be unnecessary - not sure if Valid can ever be false while the string is non-empty
func strFromNS(s sql.NullString) string {
	if !s.Valid {
		return ""
	}
	return s.String
}

func strFromNI64(i sql.NullInt64) string {
	if !i.Valid {
		return ""
	}
	return strconv.FormatInt(int64(i.Int64), 10)
}
func strFromNB(b sql.NullBool) string {
	if !b.Valid {
		return ""
	}
	if b.Bool {
		return "true"
	}
	return "false"
}

// takes a string and returns a NullString
func nsFromStr(s string) sql.NullString {
	if len(s) > 0 {
		return sql.NullString{Valid: true, String: s}
	}
	return sql.NullString{}
}

// like strings.Join but adds strings to sql.NullString. Note order of separator.
func nsJoin(ns sql.NullString, sep string, s ...string) sql.NullString {
	// get rid of empty strings
	for i := 0; i < len(s); {
		if len(strings.TrimSpace(s[i])) > 0 {
			i++
			continue
		}
		s = append(s[:i], s[i+1:]...)
	}
	if len(s) == 0 {
		return ns
	}
	var strs []string
	if ns.Valid && len(ns.String) > 0 {
		strs = append(strs, ns.String)
	}
	if len(strs) > 0 {
		strs = append(strs, s...)
	} else {
		strs = s
	}

	ns.String = strings.Join(strs, sep)
	ns.Valid = true
	return ns
}

type FldOrder int

const (
	FldNA FldOrder = iota - 1
	FldID
	FldQty
	FldNPkg
	FldPkg
	FldMtg
	FldOrigin
	FldLocation
	FldDatasheet
	FldNotes
	FldAttrs
	FldDescription // not currently set in common, but can be read for NPkg
)

var fldOrderStrs = map[FldOrder]string{
	FldNA:        "N/A",
	FldID:        "ID",
	FldQty:       "qty",
	FldNPkg:      "n/pkg",
	FldPkg:       "pkg",
	FldMtg:       "mounting",
	FldOrigin:    "origin",
	FldLocation:  "loc",
	FldDatasheet: "ds",
	FldNotes:     "notes",
	FldAttrs:     "attrs",
}

func (f FldOrder) String() string { return fldOrderStrs[f] }

func (c commonFieldsOut) Strings(ord []FldOrder, ret []string) []string {
	if ret == nil {
		ret = make([]string, len(ord))
	}
	npkg := "1"
	if c.NPkg > 0 {
		npkg = strconv.FormatInt(c.NPkg, 10)
	}
	for i, o := range ord {
		switch o {
		case 0:
			ret[i] = strconv.FormatInt(c.ID, 10)
		case 1:
			ret[i] = strconv.FormatInt(int64(c.Qty), 10)
		case 2:
			ret[i] = npkg
		case 3:
			ret[i] = strFromNS(c.Package) // FIXME assumes the string will be empty whenever Valid is false
		case 4:
			ret[i] = c.Mounting.String()
		case 5:
			ret[i] = strFromNS(c.Origin)
		case 6:
			ret[i] = c.Location
		case 7:
			ret[i] = strFromNS(c.Datasheet)
		case 8:
			ret[i] = strFromNS(c.Notes)
		case 9:
			ret[i] = c.Attrs.String()
		}
	}
	return ret
}

// func (c commonFields) columnHeaders(ord []int, ret []string) []string {
// 	if ret == nil {
// 		ret = make([]string, len(ord))
// 	}
// 	for i, o := range ord {
// 		switch o {
// 		case 0:
// 			ret[i] = "id"
// 		case 1:
// 			ret[i] = "qty"
// 		case 2:
// 			ret[i] = "n/pkg"
// 		case 3:
// 			ret[i] = "pkg"
// 		case 4:
// 			ret[i] = "mounting"
// 		case 5:
// 			ret[i] = "origin"
// 		case 6:
// 			ret[i] = "loc"
// 		case 7:
// 			ret[i] = "ds"
// 		case 8:
// 			ret[i] = "notes"
// 		case 9:
// 			ret[i] = "attrs"
// 		}
// 	}

// 	return ret
// }

// parse rec as float, updating field if input is valid
func parseF(fld *float64, in string) error {
	if len(in) == 0 {
		return nil
	}
	val, _, err := humanize.ParseSI(in)
	if err != nil {
		return err
	}
	*fld = val
	return nil
}

// parseF, but for NullFloat64's
func parseNF(fld *sql.NullFloat64, in string, mult ...rune) error {
	if len(in) == 0 {
		return nil
	}
	if len(mult) > 1 {
		return fmt.Errorf("mult can be at most 1 rune")
	}
	if len(mult) == 1 {
		in += string(mult[0])
	}
	if idx := strings.LastIndexAny(in, "=<>"); idx > -1 {
		// remove >=, <=, =, <, >, etc
		in = in[idx+1:]
	}
	val, _, err := humanize.ParseSI(in)
	if err != nil {
		return err
	}
	fld.Valid = true
	fld.Float64 = val
	return nil
}
func parseNI64(fld *sql.NullInt64, in string) error {
	if len(in) == 0 {
		return nil
	}
	val, err := strconv.ParseInt(in, 10, 64)
	if err != nil {
		return err
	}
	fld.Valid = true
	fld.Int64 = val
	return nil
}

func insertNullFloat(f sql.NullFloat64, fld string, cols *[]string, vals *[]any) {
	if !f.Valid {
		return
	}
	*cols = append(*cols, fld)
	*vals = append(*vals, f.Float64)
}

func insertNullInt64(i sql.NullInt64, fld string, cols *[]string, vals *[]any) {
	if !i.Valid {
		return
	}
	*cols = append(*cols, fld)
	*vals = append(*vals, i.Int64)
}

// insert if valid
func insertNullStr(s sql.NullString, fld string, cols *[]string, vals *[]any) {
	if !s.Valid {
		return
	}
	*cols = append(*cols, fld)
	*vals = append(*vals, s.String)
}

func insertNullBool(b sql.NullBool, fld string, cols *[]string, vals *[]any) {
	if !b.Valid {
		return
	}
	*cols = append(*cols, fld)
	*vals = append(*vals, b.Bool)
}

type Qty int64

// qty
const (
	QtyMany    Qty = -99
	QtyUnknown Qty = -999
)

func ParseQty(s string) (Qty, error) {
	s = strings.TrimSpace(s)
	s = strings.ToLower(s)
	switch s {
	case "many":
		return QtyMany, nil
	case "?", "0?", "unknown", "":
		return QtyUnknown, nil
	}
	q, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return QtyUnknown, err
	}
	return Qty(q), nil
}
func (q Qty) String() string {
	switch q {
	case QtyMany:
		return "many"
	case QtyUnknown:
		return "?"
	default:
		return strconv.FormatInt(int64(q), 10)
	}
}

func (c *commonFieldsIn) setPkg(pkg string) error {
	pkg = strings.ToUpper(pkg)
	if strings.Contains(pkg, "-") {
		c.Package = sql.NullString{String: pkg, Valid: true}
		return nil
	}
	i := strings.IndexFunc(pkg, unicode.IsDigit)
	if i < 0 {
		// no digits
		c.Package = sql.NullString{String: pkg, Valid: true}
	}
	if i == 0 { // first is digit - not valid?
		return fmt.Errorf("invalid package, must not begin with digit")
	}
	// letters, then digits - insert a dash
	c.Package = sql.NullString{String: pkg[:i] + "-" + pkg[i:], Valid: true}
	return nil
}

// TODO can we have array fields? array of comments would be useful

type Mounting int

const (
	MtgUnspecified Mounting = iota
	MtgUnknown
	MtgSMT
	MtgTH
	MtgPanel
	MtgChassis
	MtgOther
)

var mtgStrs = map[Mounting]string{
	// MtgUnspecified: "-",
	MtgUnknown: "unknown",
	MtgSMT:     "SMT",
	MtgTH:      "TH",
	MtgPanel:   "panel",
	MtgChassis: "chassis",
	MtgOther:   "other",
}

func (m Mounting) String() string { return mtgStrs[m] }

func (m *Mounting) Parse(s string) error {
	if len(s) == 0 {
		*m = MtgUnknown
		return nil
	}
	switch strings.ToLower(s) {
	case "unsp", "unspec", "unspecified":
		*m = MtgUnspecified
	case "unk", "unknown":
		*m = MtgUnknown
	case "smt", "smd":
		*m = MtgSMT
	case "th", "tht", "through", "throughhole":
		*m = MtgTH
	case "pnl", "panel":
		*m = MtgPanel
	case "chas", "chassis":
		*m = MtgPanel
	case "other":
		*m = MtgOther
	default:
		return fmt.Errorf("unknown mounting %q", s)
	}
	return nil
}

type Locations struct {
	ID   int64
	Name string
}

var _ dbTbl = (*Locations)(nil)

func (Locations) TableName() string { return "locations" }

type auxTbl int

const (
	tblLogicDescriptions auxTbl = iota
	tblLocations
)

var auxTblNames = map[auxTbl]string{
	tblLogicDescriptions: "logicDescriptions",
	tblLocations:         "locations",
}

// retrieve a value from an auxiliary table, e.g. location or logicDescriptions
func auxTblVal(db *sql.DB, tbl auxTbl, f, v string) (int64, error) {
	tname, ok := auxTblNames[tbl]
	if !ok {
		return 0, fmt.Errorf("unknown aux table %d", tbl)
	}
	q := fmt.Sprintf(`SELECT id FROM %s WHERE %s='%s'`, tname, f, v)
	rows, err := db.Query(q)
	if err != nil {
		return 0, err
	}
	defer rows.Close()

	var id, n int64
	for rows.Next() {
		n++
		if n > 1 {
			return 0, fmt.Errorf("too many results for query %q", q)
		}
		if err := rows.Scan(&id); err != nil {
			return 0, err
		}
	}
	if rows.Err() != nil {
		return 0, rows.Err()
	}
	if n == 0 {
		// no result, add it
		q = fmt.Sprintf("INSERT INTO %s (%s) VALUES('%s')", tname, f, v)
		res, err := db.Exec(q)
		if err != nil {
			return 0, err
		}
		id, err = res.LastInsertId()
		if err != nil {
			return 0, err
		}
	}
	return id, nil
}

var TblDefaultSelect = map[CompTbl]string{
	CTLogic: `
SELECT logic.id AS id,qty,npkg,vrange,prefix AS pfx,series,family AS fam,
	func,sfx,category,logicd.desc AS desc,package AS pkg,mounting,
	origin,locs.name AS loc,datasheet AS ds,attrs,notes
FROM logic 
LEFT JOIN locations AS locs ON logic.location=locs.id
LEFT JOIN logicDescriptions AS logicd ON logic.description=logicd.id
WHERE ?
ORDER BY logic.series ASC,logic.func ASC,logic.family ASC,logic.qty DESC
`,
	CTTransistors: `
SELECT q.id AS id,qty,npkg,pn,type,doping,maxv,maxa,maxf,gain,package,mounting,
	origin,locs.name AS location,datasheet,attrs,notes,altpn
FROM transistors AS q 
LEFT JOIN locations AS locs ON q.location=locs.id
WHERE ?
ORDER BY type ASC,doping ASC,pn ASC,q.qty DESC
`,
	CTDevKits:   devkitSelect,
	CTDiodeTVS:  diodetvsSelect,
	CTLineDrv:   linedrvSelect,
	CTOpamp:     opampSelect,
	CTOpto:      optoSelect,
	CTOther:     otherSelect,
	CTPassive:   passiveSelect,
	CTPower:     powerSelect,
	CTTmrOscPll: tmrOscPllSelect,
}

func OutputRowsType(tbl CompTbl) dbOut {
	switch tbl {
	case CTLogic:
		return &logicOutput{}
	case CTTransistors:
		return &transistorOutput{}
	case CTDevKits:
		return &DevkitOutput{}
	case CTDiodeTVS:
		return &DiodeTVSOutput{}
	case CTLineDrv:
		return &LineDrvrOutput{}
	case CTOpamp:
		return &OpampOutput{}
	case CTOpto:
		return &OptoOutput{}
	case CTOther:
		return &OtherOutput{}
	case CTPassive:
		return &PassiveOutput{}
	case CTPower:
		return &PowerOutput{}
	case CTTmrOscPll:
		return &TmrOscPllOutput{}
	default:
		return nil
	}
}

type DefaultRow interface{ Strings() []string }
type DefaultRows []DefaultRow

type sqliteBlob map[string]string

func (s *sqliteBlob) String() string {
	if s == nil || len(*s) == 0 {
		return ""
	}
	// out := make([]string, 0, len(*s))
	// for k, v := range *s {
	// 	out = append(out, fmt.Sprintf("%s=%s", k, v))
	// }
	// return strings.Join(out, ", ")
	out, err := json.Marshal(*s)
	if err != nil {
		log.Fatalf("sqliteBlob.String(): %s", err)
	}
	return string(out)
}

func (s *sqliteBlob) Scan(a any) error {
	if a == nil {
		return nil
	}
	// log.Printf("Scan(a), a=%#v", a)
	buf := a.([]byte)
	return json.Unmarshal(buf, s)
}
