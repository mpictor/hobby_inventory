package db

import (
	"database/sql"
	"fmt"
	"log"
	"strconv"
	"strings"
	"unicode"
)

// fields common to all:
type commonFields struct {
	ID        int64          // row id in db
	Qty       Qty            // quantity
	NPkg      int64          // number per package, typically 1
	Package   sql.NullString // TO-220, DIP-14, SOIC-8, etc
	Mounting  Mounting       // through hole, smt, etc
	Origin    sql.NullString // freeform, where I got it from - digikey, estate, etc
	Location  int64          `db:"location,FK:locations:id"` // maps to db table locations
	Datasheet sql.NullString // local file name, or databook and page
	Attrs     sqliteBlob     // additional attrs, json-encoded
	Notes     sql.NullString
	// Description - not in commonFields, as multiple TTL or CMOS will share one description
}

func (c commonFields) insert() (cols []string, vals []any) {
	// always skip the id?
	if c.Qty != QtyUnknown {
		cols = append(cols, "qty")
		vals = append(vals, int64(c.Qty))
	}
	cols = append(cols, "npkg", "mounting", "location")
	vals = append(vals, c.NPkg, int64(c.Mounting), c.Location)
	insertNullStr(&c.Package, "package", &cols, &vals)
	insertNullStr(&c.Origin, "origin", &cols, &vals)
	insertNullStr(&c.Datasheet, "datasheet", &cols, &vals)
	if c.Attrs != nil {
		panic("attrs unimplemented")
	}
	insertNullStr(&c.Notes, "notes", &cols, &vals)

	return cols, vals
}

// set common fields, returning remaining fields
// attempt to standardize keys and values since they are human input
func (c *commonFields) setParams(db *sql.DB, m paramMap) (paramMap, error) {
	remain := make(paramMap)
	for k, v := range m {
		switch strings.ToLower(k) {
		case "qty":
			q, err := ParseQty(v)
			if err != nil {
				return nil, err
			}
			c.Qty = q
		case "npkg":
			n, err := strconv.ParseInt(v, 10, 64)
			if err != nil {
				return nil, err
			}
			c.NPkg = n
		case "package":
			if err := c.setPkg(v); err != nil {
				return nil, err
			}
		case "mounting":
			if err := c.Mounting.Parse(v); err != nil {
				return nil, err
			}
		case "origin":
			c.Origin = sql.NullString{Valid: true, String: v}
		case "location":
			n, err := auxTblVal(db, tblLocations, "name", v)
			if err != nil {
				return nil, fmt.Errorf("setting location: %w", err)
			}
			c.Location = n
		case "datasheet":
			c.Datasheet = sql.NullString{Valid: true, String: v}
		case "notes":
			c.Notes = sql.NullString{Valid: true, String: v}
		default:
			remain[k] = v
		}
	}
	return remain, nil
}

// takes a null string and returns a string
// may be unnecessary - not sure if Valid can ever be false while the string is non-empty
func strFromNS(s sql.NullString) string {
	if !s.Valid {
		return ""
	}
	return s.String
}

// takes a string and returns a NullString
func nsFromStr(s string) sql.NullString {
	if len(s) > 0 {
		return sql.NullString{Valid: true, String: s}
	}
	return sql.NullString{}
}

func (c commonFields) Strings(ord []int, ret []string) []string {
	if ret == nil {
		ret = make([]string, 10)
	}
	npkg := ""
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
			ret[i] = strconv.FormatInt(c.Location, 10) // TODO table lookup
		case 7:
			ret[i] = strFromNS(c.Datasheet)
		case 8:
			ret[i] = strFromNS(c.Notes)
		case 9:
			ret[i] = c.Attrs.String()
		}
	}
	// return append(ret,
	// 	strconv.FormatInt(c.ID, 10),
	// 	strconv.FormatInt(int64(c.Qty), 10),
	// 	npkg,
	// 	c.Package.String, // FIXME assumes the string will be empty whenever Valid is false
	// 	c.Mounting.String(),
	// 	c.Origin.String,
	// 	strconv.FormatInt(c.Location, 10), // TODO table lookup
	// 	c.Datasheet.String,
	// 	c.Notes.String,
	// )
	return ret
}

func (c commonFields) ColumnHeaders(ord []int, ret []string) []string {
	if ret == nil {
		ret = make([]string, len(ord))
	}
	for i, o := range ord {
		switch o {
		case 0:
			ret[i] = "id"
		case 1:
			ret[i] = "qty"
		case 2:
			ret[i] = "n/pkg"
		case 3:
			ret[i] = "pkg"
		case 4:
			ret[i] = "mounting"
		case 5:
			ret[i] = "origin"
		case 6:
			ret[i] = "loc"
		case 7:
			ret[i] = "ds"
		case 8:
			ret[i] = "notes"
		case 9:
			ret[i] = "attrs"
		}
	}

	return ret
}

// insert if valid
func insertNullStr(s *sql.NullString, fld string, cols *[]string, vals *[]any) {
	if !s.Valid {
		return
	}
	*cols = append(*cols, fld)
	*vals = append(*vals, s.String)
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
	case "?", "unknown", "":
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

func (c *commonFields) setPkg(pkg string) error {
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
		return fmt.Errorf("unknown mounting %s", s)
	}
	return nil
}

type Locations struct {
	ID   int64
	Name string
}

var _ dbTbl = (*Locations)(nil)

func (Locations) isDbTbl() {}

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

// returns go type corresponding to the given db table, or nil
func GetTbl(name string) mainDBtbl {
	switch strings.ToLower(name) {
	case "ttl", "cmos", "logic":
		return &Logic{}
		// TODO
	}
	return nil
}

var TblOrder = map[string][]int{
	"logic": []int{0, 1, 2, 9, 10, 11, 12, 13, 14, 15, 3, 4, 5, 6, 7, 8, 16, 17},
}
var TblDefaultSelect = map[string]string{
	"logic": `
SELECT logic.id AS id,qty,npkg,vrange,prefix AS pfx,series,family AS fam,
	func,sfx,category,logicd.desc AS desc,package AS pkg,mounting,
	origin,locs.name AS loc,datasheet AS ds,attrs,notes
FROM logic 
LEFT JOIN locations AS locs ON logic.location=locs.id
LEFT JOIN logicDescriptions AS logicd ON logic.description=logicd.id
WHERE ?
ORDER BY logic.series ASC,logic.func ASC,logic.family ASC,logic.qty DESC
`,
}

type DefaultRow interface{ Strings() []string }

type sqliteBlob map[string]string

func (s *sqliteBlob) String() string {
	out := make([]string, 0, len(*s))
	for k, v := range *s {
		out = append(out, fmt.Sprintf("%s=%s", k, v))
	}
	return strings.Join(out, ", ")
}

func (s *sqliteBlob) Scan(a any) error {
	if a == nil {
		return nil
	}
	log.Fatalf("unimplemented. a=%#v", a)
	// TODO
	return nil
}
