package db

import (
	"bytes"
	"database/sql"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"iter"
	"strconv"
	"strings"
	"unicode"
)

type TTLRow struct {
	commonFields
	Prefix      sql.NullString // SN, CD, etc
	Series      sql.NullString // 54, 74, etc
	Family      sql.NullString // F, LS, ACT, etc
	Func        string         // 00 (quad NAND) etc
	Sfx         sql.NullString // suffix if any
	Category    sql.NullString // buffer, flipflop, etc TODO enum or separate table??
	Description sql.NullInt64  `db:"description,FK:ttlDescriptions:id"` // foreign key
}

func (r TTLRow) insert() (string, []any, error) {
	cols, vals := r.commonFields.insert()
	insertNullStr(&r.Mpfx, "mpfx", &cols, &vals)
	insertNullStr(&r.Series, "series", &cols, &vals)
	insertNullStr(&r.Family, "family", &cols, &vals)
	cols = append(cols, "func")
	vals = append(vals, r.Func)
	insertNullStr(&r.Sfx, "sfx", &cols, &vals)
	insertNullStr(&r.Category, "category", &cols, &vals)
	if r.Description.Valid {
		cols = append(cols, "description")
		vals = append(vals, r.Description.Int64)
	}
	if len(cols) == 0 {
		return "", nil, fmt.Errorf("nothing to insert for %s", r)
	}
	ph := "?"
	ph += strings.Repeat(",?", len(vals)-1)
	s := fmt.Sprintf("INSERT INTO ttl (%s) VALUES(%s);", strings.Join(cols, ","), ph)
	return s, vals, nil
}

// TODO make pretty
func (r TTLRow) String() string {
	j, err := json.Marshal(r)
	if err != nil {
		panic(err)
	}
	return string(j)
}

func (r TTLRow) Strings() []string {
	strs := make([]string, 0, 16)
	strs = r.commonFields.Strings(strs)

	// TODO assumes NullString.String always empty when !Valid
	strs = append(strs,
		r.Mpfx.String, r.Series.String, r.Family.String, r.Func, r.Sfx.String,
		r.Category.String, strconv.FormatInt(r.Description.Int64, 10))

	return strs
}

func (r *TTLRow) parsePN(pn string) error {
	remain := strings.ToUpper(pn)
	notDigit := func(r rune) bool { return !unicode.IsDigit(r) }
	// manufacturer prefix
	idx := strings.IndexFunc(remain, unicode.IsDigit)
	if idx < 0 {
		// no digits, put it all in Func??
		r.Func = remain
		return nil
	}
	if idx > 0 {
		r.Mpfx = sql.NullString{Valid: true, String: remain[:idx]}
		remain = remain[idx:]
	}
	// suffix
	idx = strings.LastIndexFunc(remain, unicode.IsDigit)
	if idx > -1 && len(remain) > idx+1 {
		idx++
		r.Sfx = sql.NullString{Valid: true, String: remain[idx:]}
		remain = remain[:idx]
	}
	// series/family/function
	idx = strings.IndexFunc(remain, notDigit)
	if idx > -1 {
		// has family
		ser := remain[:idx]
		remain = remain[idx:]
		idx = strings.IndexFunc(remain, unicode.IsDigit)
		if idx == -1 {
			return fmt.Errorf("failed to parse pn=%s", pn)
		}
		// has function
		fam := remain[:idx]
		fun := remain[idx:]
		if len(ser) > 0 && len(fun) > 0 {
			r.Series = sql.NullString{Valid: true, String: ser}
			r.Family = sql.NullString{Valid: true, String: fam}
			r.Func = fun
			return nil
		}
	}
	// no family, separate series and function
	// known families are all 2 digit
	if len(remain) <= 2 {
		r.Func = remain
		return nil
	}
	switch remain[:2] {
	case "30", "54", "64", "74", "75":
		// recognized
		r.Series = sql.NullString{Valid: true, String: remain[:2]}
		r.Func = remain[2:]
	default:
		// not recognized, stuff it all in func
		r.Func = remain
	}

	return nil
}

// func (r TTLRow) renderWidths() []int {
// 	panic("unimplemented")
// }
// func (r TTLRow) render(widths []int) error {
// 	panic("unimplemented")
// }

type TTL []TTLRow

func (*TTL) isDbTbl() {}

func (ttl *TTL) ImportCSV(db *sql.DB, in []byte) error {
	r := csv.NewReader(bytes.NewReader(in))
	// skip header
	if _, err := r.Read(); err != nil {
		return err
	}
	for {
		record, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
			// log.Fatal(err)
		}

		// fmt.Println(record)
		row, err := ttlRow(db, record)
		if err != nil {
			return err
		}
		*ttl = append(*ttl, row)
	}
	return nil
}

func ttlRow(db *sql.DB, rec []string) (TTLRow, error) {
	var tr, zero TTLRow
	if len(rec) == 0 {
		return zero, fmt.Errorf("no data")
	}
	// did I actually find a use for the for-case paradigm?!
	// https://thedailywtf.com/articles/The_FOR-CASE_paradigm
	for i, v := range rec {
		if len(v) == 0 {
			continue
		}
		switch i {
		case 0:
			tr.Mpfx = sql.NullString{String: v, Valid: true}
		case 1:
			tr.Series = sql.NullString{String: v, Valid: true}
		case 2:
			tr.Family = sql.NullString{String: v, Valid: true}
		case 3:
			tr.Func = v
		case 4:
			tr.Sfx = sql.NullString{String: v, Valid: true}
		// case 5: PN (discard)
		case 6:
			tr.Category = sql.NullString{String: v, Valid: true}
		case 7:
			q, err := ParseQty(v)
			if err != nil {
				return zero, err
			}
			tr.Qty = q
		case 8:
			tr.Package = sql.NullString{String: v, Valid: true}
		case 9:
			desc, err := auxTblVal(db, tblTTLdescriptions, "desc", v)
			if err != nil {
				return zero, err
				// TODO log error?
			} else {
				tr.Description = sql.NullInt64{Int64: desc, Valid: true}
			}
		case 10:
			tr.Origin = sql.NullString{String: v, Valid: true}
		case 11:
			loc, err := auxTblVal(db, tblLocations, "name", v)
			if err != nil {
				return zero, err
			}
			tr.Location = loc
		case 12:
			// concatenate any additional cells (TODO improve?)
			v = strings.Join(rec[12:], ";")
			tr.Notes = sql.NullString{String: v, Valid: true}
		}
	}
	return tr, nil
}

func (ttl *TTL) Store(db *sql.DB) error {
	// TODO check that existing table is empty
	// TODO transaction?
	return ttl.Insert(db)
}
func (ttl *TTL) ColumnHeaders() ([]string, error) { panic("unimplemented") }
func (ttl *TTL) Insert(db *sql.DB) error {
	// var stmts []string
	rows := 0
	for _, r := range *ttl {
		stmt, vals, err := r.insert()
		if err != nil {
			return err
		}

		// stmts = append(stmts, ins)
		res, err := db.Exec(stmt, vals...)
		if err != nil {
			return err
		}
		ra, err := res.RowsAffected()
		if err != nil {
			return err
		}
		rows += int(ra)
	}
	if rows != len(*ttl) {
		return fmt.Errorf("expect %d rows affected, got %d", len(*ttl), rows)
	}
	return nil
}

// // TODO https://github.com/charmbracelet/bubbletea/blob/main/examples/table/main.go
// func (ttl *TTL) Render() error {
// 	hdrs, err := ttl.ColumnHeaders()
// 	if err != nil {
// 		return err
// 	}
// 	// colWidths := make([]int, len(hdrs))
// 	// for _, r := range *ttl {
// 	// 	rw := r.renderWidths()
// 	// 	for i, w := range rw {
// 	// 		colWidths[i] = max(colWidths[i], w)
// 	// 	}
// 	// }
// 	// for _, r := range *ttl {
// 	// 	if err := r.render(colWidths); err != nil {
// 	// 		return err
// 	// 	}
// 	// }
// 	return nil
// }

func (ttl TTL) All() iter.Seq[[]string] {
	return func(yield func([]string) bool) {
		for _, r := range ttl {
			if !yield(r.Strings()) {
				return
			}
		}
	}
}
func (ttl TTL) Len() int { return len(ttl) }

func (ttl *TTL) Update(*sql.DB) error { panic("unimplemented") }

func (ttl *TTL) SetRow(db *sql.DB, kvs []string) error {
	params := toParamMap(kvs)

	row := TTLRow{}
	params, err := row.commonFields.setParams(db, params)
	if err != nil {
		return err
	}
	if v, ok := params["pn"]; ok {
		for _, f := range []string{"prefix", "series", "family", "function", "suffix"} {
			if _, ok := params[f]; ok {
				return fmt.Errorf("pn/partnumber key is mutually exclusive with part number sub-fields such as %q", f)
			}
		}
		if err := row.parsePN(v); err != nil {
			return err
		}
		delete(params, "pn")
	}

	for k, v := range params {
		switch k {
		case "prefix":
			row.Mpfx = sql.NullString{Valid: true, String: strings.ToUpper(v)}
		case "series":
			row.Series = sql.NullString{Valid: true, String: strings.ToUpper(v)}
		case "family":
			row.Family = sql.NullString{Valid: true, String: strings.ToUpper(v)}
		case "function":
			row.Func = v
		case "suffix":
			row.Sfx = sql.NullString{Valid: true, String: strings.ToUpper(v)}
		case "category":
			row.Category = sql.NullString{Valid: true, String: strings.ToUpper(v)}
		case "description":
			n, err := auxTblVal(db, tblTTLdescriptions, "desc", v)
			if err != nil {
				return fmt.Errorf("setting description: %w", err)
			}
			row.Description = sql.NullInt64{Valid: true, Int64: n}
		// TODO lookup/insert in table
		default:
			return fmt.Errorf("unhandled key %s", k)
		}
	}

	*ttl = append(*ttl, row)
	return nil
}

// must implement
var _ mainDBtbl = (*TTL)(nil)

type ttlDesc struct {
	ID   int64
	Desc string
}
type TTLdescriptions []ttlDesc

// must implement
var _ dbTbl = (TTLdescriptions)(nil)

func (TTLdescriptions) isDbTbl() {}
