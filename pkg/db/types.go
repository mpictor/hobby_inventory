package db

import (
	"database/sql"
	"fmt"
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
func (c commonFields) Strings(ret []string) []string {
	if ret == nil {
		ret = make([]string, 0, 9)
	}
	ret[0] = strconv.FormatInt(c.ID, 10)
	ret[1] = strconv.FormatInt(int64(c.Qty), 10)
	ret[2] = strconv.FormatInt(c.NPkg, 10)
	ret[3] = c.Package.String // FIXME assumes the string will be empty whenever Valid is false
	ret[4] = c.Mounting.String()
	ret[5] = c.Origin.String
	ret[6] = strconv.FormatInt(c.Location, 10) // TODO table lookup
	ret[7] = c.Datasheet.String
	ret[8] = c.Notes.String
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
	MtgUnspecified: "unspecified",
	MtgUnknown:     "unknown",
	MtgSMT:         "SMT",
	MtgTH:          "TH",
	MtgPanel:       "panel",
	MtgChassis:     "chassis",
	MtgOther:       "other",
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
	tblTTLdescriptions auxTbl = iota
	tblLocations
)

var auxTblNames = map[auxTbl]string{
	tblTTLdescriptions: "ttlDescriptions",
	tblLocations:       "locations",
}

// retrieve a value from an auxiliary table, e.g. location or ttlDescription
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
	case "ttl":
		return &TTL{}
	case "cmos":
		return &CMOS{}
		// TODO
	}
	return nil
}
