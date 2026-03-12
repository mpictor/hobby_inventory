package db

import (
	"bytes"
	"database/sql"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"strconv"
	"strings"
	"unicode"
)

type LogicRow struct {
	commonFields
	VRange      VRange
	Prefix      sql.NullString // SN, CD, etc
	Series      sql.NullString // 54, 74, etc
	Family      sql.NullString // F, LS, ACT, etc
	Func        string         // 00 (quad NAND) etc
	Sfx         sql.NullString // suffix if any
	Category    sql.NullString // buffer, flipflop, etc TODO enum or separate table??
	Description sql.NullInt64  `db:"description,FK:logicDescriptions:id"` // foreign key
}

func (l LogicRow) insert() (string, []any, error) {
	cols, vals := l.commonFields.insert()
	cols = append(cols, "vrange")
	vals = append(vals, l.VRange)
	insertNullStr(&l.Prefix, "prefix", &cols, &vals)
	insertNullStr(&l.Series, "series", &cols, &vals)
	insertNullStr(&l.Family, "family", &cols, &vals)
	cols = append(cols, "func")
	vals = append(vals, l.Func)
	insertNullStr(&l.Sfx, "sfx", &cols, &vals)
	insertNullStr(&l.Category, "category", &cols, &vals)
	if l.Description.Valid {
		cols = append(cols, "description")
		vals = append(vals, l.Description.Int64)
	}
	if len(cols) == 0 {
		return "", nil, fmt.Errorf("nothing to insert for %s", l)
	}
	ph := "?"
	ph += strings.Repeat(",?", len(vals)-1)
	s := fmt.Sprintf("INSERT INTO logic (%s) VALUES(%s);", strings.Join(cols, ","), ph)
	return s, vals, nil
}

// TODO make pretty
func (l LogicRow) String() string {
	j, err := json.Marshal(l)
	if err != nil {
		panic(err)
	}
	return string(j)
}

// func (l LogicRow) Strings(ord []int) []string {
// 	if ord == nil {
// 		ord = []int{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17}
// 	}
// 	strs := make([]string, len(ord))
// 	// 0-9 are in commonFields
// 	const lastCommon = 9
// 	strs = l.commonFields.Strings(ord, strs)
//
// 	for i, o := range ord {
// 		switch o {
// 		case lastCommon + 1:
// 			strs[i] = strFromNS(l.Prefix)
// 		case lastCommon + 2:
// 			strs[i] = strFromNS(l.Series)
// 		case lastCommon + 3:
// 			strs[i] = strFromNS(l.Family)
// 		case lastCommon + 4:
// 			strs[i] = l.Func
// 		case lastCommon + 5:
// 			strs[i] = strFromNS(l.Sfx)
// 		case lastCommon + 6:
// 			strs[i] = strFromNS(l.Category)
// 		case lastCommon + 7:
// 			strs[i] = strconv.FormatInt(l.Description.Int64, 10)
// 		}
// 	}
//
// 	return strs
// }

func (l *LogicRow) parsePN(pn string) error {
	// TODO support cmos
	remain := strings.ToUpper(pn)
	notDigit := func(r rune) bool { return !unicode.IsDigit(r) }
	// manufacturer prefix
	idx := strings.IndexFunc(remain, unicode.IsDigit)
	if idx < 0 {
		// no digits, put it all in Func??
		l.Func = remain
		return nil
	}
	if idx > 0 {
		l.Prefix = sql.NullString{Valid: true, String: remain[:idx]}
		remain = remain[idx:]
	}
	// suffix
	idx = strings.LastIndexFunc(remain, unicode.IsDigit)
	if idx > -1 && len(remain) > idx+1 {
		idx++
		l.Sfx = sql.NullString{Valid: true, String: remain[idx:]}
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
			l.Series = sql.NullString{Valid: true, String: ser}
			l.Family = sql.NullString{Valid: true, String: fam}
			l.Func = fun
			return nil
		}
	}
	// no family, separate series and function
	// known families are all 2 digit
	if len(remain) <= 2 {
		l.Func = remain
		return nil
	}
	switch remain[:2] {
	case "30", "54", "64", "74", "75":
		// recognized
		l.Series = sql.NullString{Valid: true, String: remain[:2]}
		l.Func = remain[2:]
	default:
		// not recognized, stuff it all in func
		l.Func = remain
	}

	return nil
}

type Logic []LogicRow

func (*Logic) isDbTbl() {}

func (lgc *Logic) ImportCSV(db *sql.DB, tbl string, in []byte) error {
	r := csv.NewReader(bytes.NewReader(in))
	xlate := logicTranslation[tbl]
	for range xlate.headerRows {
		// skip header
		if _, err := r.Read(); err != nil {
			return err
		}
	}
	for {
		record, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		if allEmpty(record) {
			continue
		}
		if Verbose {
			log.Println("csv:", record)
		}
		row, err := logicRow(db, tbl, record)
		if err != nil {
			return err
		}
		*lgc = append(*lgc, row)
	}
	return nil
}

func allEmpty(r []string) bool {
	for _, c := range r {
		c = strings.TrimSpace(c)
		if c == "0" || c == "00" {
			continue
		}
		if len(c) > 0 {
			return false
		}
	}
	return true
}

var logicTranslation = map[string]struct {
	cols       map[int]int
	fixup      func(*LogicRow, []string)
	headerRows int
}{
	// 0     1      2      3       4    5   6      7    8   9           10     11       12
	// mpfx,series,family,function,sfx,PN,category,qty,pkg,description,origin,location,notes
	"ttl": {
		cols: map[int]int{0: 0, 1: 1, 2: 2, 3: 3, 4: 4, 5: 5, 6: 6, 7: 7, 8: 8, 9: 9, 10: 10, 11: 11, 12: 12},
		fixup: func(lr *LogicRow, rec []string) {
			lr.VRange = VrTTL
		},
		headerRows: 1,
	},
	// 0     1   2    3   4   5       6 7 8           9           10       11            12
	// mpfx,ord,func,sfx,PN,category,qty,,description,location,interesting,Motorola 1978,comments,,,,,,
	"cmos": {
		cols: map[int]int{0: 0, 1: 1, 2: 3, 3: 4, 4: 5, 5: 6, 6: 7 /*skip 7*/, 8: 9, 9: 11, 10: 10, 11: 13, 12: 12},
		fixup: func(lr *LogicRow, rec []string) {
			// mpfx,ord,func,sfx
			if !lr.Prefix.Valid && !lr.Series.Valid && len(lr.Func) == 0 && !lr.Sfx.Valid && len(rec[4]) > 0 {
				// only func was defined, so use verbatim
				lr.Func = rec[4]
			}
			lr.VRange = VrCMOS
		},
		headerRows: 2,
	},
}

func logicRow(db *sql.DB, tbl string, rec []string) (LogicRow, error) {
	var tr, zero LogicRow
	if len(rec) == 0 {
		return zero, fmt.Errorf("no data")
	}
	xlate, ok := logicTranslation[tbl]
	if !ok {
		return zero, fmt.Errorf("unknown table %s", tbl)
	}

	// did I actually find a use for the for-case paradigm?!
	// https://thedailywtf.com/articles/The_FOR-CASE_paradigm
	for i, v := range rec {
		if len(v) == 0 {
			continue
		}
		if xlate.cols != nil {
			j, ok := xlate.cols[i]
			if ok {
				i = j
			}
			// otherwise just use j
		}

		switch i {
		case 0:
			tr.Prefix = sql.NullString{String: v, Valid: true}
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
			desc, err := auxTblVal(db, tblLogicDescriptions, "desc", v)
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
		case 13:
			tr.Datasheet = nsFromStr("moto78:" + v)
		case 12:
			// concatenate any additional cells (TODO improve?)
			var v []string
			for _, r := range rec[12:] {
				r = strings.TrimSpace(r)
				if len(r) > 0 {
					v = append(v, r)
				}
			}
			if len(v) > 0 {
				tr.Notes = sql.NullString{String: strings.Join(v, ";"), Valid: true}
			}
		}
	}
	if xlate.fixup != nil {
		xlate.fixup(&tr, rec)
	}
	return tr, nil
}

func (lgc *Logic) Store(db *sql.DB) error {
	// TODO check that existing table is empty
	// TODO transaction?
	return lgc.Insert(db)
}
func (lgc *Logic) ColumnHeaders(ord []int) []string {
	strs := make([]string, len(ord))
	strs = commonFields{}.ColumnHeaders(ord, strs)
	for i, o := range ord {
		switch o {
		case 9:
			strs[i] = "pfx"
		case 10:
			strs[i] = "series"
		case 11:
			strs[i] = "fam"
		case 12:
			strs[i] = "func"
		case 13:
			strs[i] = "sfx"
		case 14:
			strs[i] = "category"
		case 15:
			strs[i] = "desc"
		}
	}
	return strs
}

func (lgc *Logic) Insert(db *sql.DB) error {
	rows := 0
	for _, r := range *lgc {
		stmt, vals, err := r.insert()
		if err != nil {
			return err
		}

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
	if rows != len(*lgc) {
		return fmt.Errorf("expect %d rows affected, got %d", len(*lgc), rows)
	}
	return nil
}

//	func (lgc Logic) All(ord []int) iter.Seq[[]string] {
//		return func(yield func([]string) bool) {
//			for _, r := range lgc {
//				if !yield(r.Strings(ord)) {
//					return
//				}
//			}
//		}
//	}
func (lgc Logic) Len() int { return len(lgc) }

func (lgc *Logic) Update(*sql.DB) error { panic("unimplemented") }

func (lgc *Logic) SetRow(db *sql.DB, kvs []string) error {
	params := ToParamMap(kvs, true)

	row := LogicRow{}
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
			row.Prefix = sql.NullString{Valid: true, String: strings.ToUpper(v)}
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
			n, err := auxTblVal(db, tblLogicDescriptions, "desc", v)
			if err != nil {
				return fmt.Errorf("setting description: %w", err)
			}
			row.Description = sql.NullInt64{Valid: true, Int64: n}
		// TODO lookup/insert in table
		default:
			return fmt.Errorf("unhandled key %s", k)
		}
	}

	*lgc = append(*lgc, row)
	return nil
}

// must implement
var _ mainDBtbl = (*Logic)(nil)

type logicDesc struct {
	ID   int64
	Desc string
}
type logicDescriptions []logicDesc

// must implement
var _ dbTbl = (logicDescriptions)(nil)

func (logicDescriptions) isDbTbl() {}

type VRange int

const (
	VrUnknown VRange = iota
	VrTTL
	VrCMOS
	VrLVttl
)

var vrStrings = map[VRange]string{
	VrUnknown: "unknown",
	VrTTL:     "ttl",
	VrCMOS:    "cmos",
	VrLVttl:   "lvttl",
}

func (vr VRange) String() string {
	if s, ok := vrStrings[vr]; ok {
		return s
	}
	return vrStrings[VrUnknown]
}

// logic data, for output
type logicOutRow struct {
	Id              int
	Qty             Qty
	NPkg            int
	VRange          VRange
	Pfx, Series     sql.NullString
	Family          sql.NullString `db:"fam"`
	Func            string
	Sfx, Category   sql.NullString
	Desc, Pkg       sql.NullString
	Mounting        Mounting
	Origin, Loc, DS sql.NullString
	Attrs           sqliteBlob
	Notes           sql.NullString
}

func (l logicOutRow) Strings() []string {
	strs := make([]string, 18)
	strs[0] = strconv.FormatInt(int64(l.Id), 10)
	strs[1] = l.Qty.String()
	if l.NPkg > 0 {
		strs[2] = strconv.FormatInt(int64(l.NPkg), 10)
	}
	strs[3] = l.VRange.String()
	strs[4] = strFromNS(l.Pfx)
	strs[5] = strFromNS(l.Series)
	strs[6] = strFromNS(l.Family)
	strs[7] = l.Func
	strs[8] = strFromNS(l.Sfx)
	strs[9] = strFromNS(l.Category)
	strs[10] = strFromNS(l.Desc)
	strs[11] = strFromNS(l.Pkg)
	strs[12] = l.Mounting.String()
	strs[13] = strFromNS(l.Origin)
	strs[14] = strFromNS(l.Loc)
	strs[15] = strFromNS(l.DS)
	strs[16] = l.Attrs.String()
	strs[17] = strFromNS(l.Notes)
	return strs
}

// func (l *logicOutRow) Scan(a any) error { }
// var _ sql.Scanner = (*logicOutRow)(nil)
