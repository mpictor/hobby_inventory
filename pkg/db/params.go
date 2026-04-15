package db

import (
	"database/sql"
	"fmt"
	"strings"
	"unicode"
)

func ParamsToRow(db *sql.DB, ttyp CompTbl, args []string) (mainDBtbl, error) {
	tbl := GetTbl(ttyp)
	if tbl == nil {
		return nil, fmt.Errorf("%s is not a valid table name", ttyp)
	}
	err := tbl.SetRow(db, args)
	if err != nil {
		return nil, err
	}
	return tbl, nil
}

// type paramMap map[string]string

type Op int

const (
	Undef    Op = iota
	EQ          // == or =
	LT          // <  or ,
	LE          // <= or ,=
	GT          // >  or .
	GE          // >= or .=
	NE          // !=
	CONTAINS    // :
)

var op2str = map[Op]string{
	EQ:       "=",
	LT:       "<",
	LE:       "<=",
	GT:       ">",
	GE:       ">=",
	NE:       "!=", // Note: special case
	CONTAINS: "LIKE",
}

func (o Op) String() string {
	if s, ok := op2str[o]; ok {
		return s
	}
	return fmt.Sprintf("unknown op %d", o)
}

// like strings.Cut
func cutOp(in string) (k, v string, op Op) {
	idx := strings.IndexAny(in, "!<>,.:=")
	if idx < 0 {
		return in, "", Undef
	}
	k = in[:idx]
	switch {
	case strings.HasPrefix(in[idx:], "<="), strings.HasPrefix(in[idx:], ",="):
		return k, in[idx+2:], LE
	case strings.HasPrefix(in[idx:], ">="), strings.HasPrefix(in[idx:], ".="):
		return k, in[idx+2:], GE
	case strings.HasPrefix(in[idx:], "=="):
		return k, in[idx+2:], EQ
	case strings.HasPrefix(in[idx:], "!="):
		return k, in[idx+2:], NE
	}
	v = in[idx+1:]
	switch in[idx] {
	case '<', ',':
		op = LT
	case '>', '.':
		op = GT
	case ':':
		op = CONTAINS
	case '=':
		op = EQ
	}
	return k, v, op
}

type paramVal struct {
	Op  Op
	Val string
}
type paramMap map[string]paramVal

// turns slice of key=value into map[key]value, and performs
// substitutions in keyAliases if requested
//
// if quote is true:
//   - if op==CONTAINS, wraps value in percent signs
//   - wraps non-numeric values in single quotes
func ToParamMap(kvs []string, substitute bool, quote bool) paramMap {
	m := make(paramMap)
	for _, kv := range kvs {
		// k, v, ok := strings.Cut(kv, "=")
		k, v, op := cutOp(kv) // TODO
		if op == Undef {
			// TODO - error
		}
		// if !ok || len(k) == 0 || len(v) == 0 {
		// continue
		// }
		k = strings.ToLower(k)
		if nk, ok := paramAliases[k]; ok && substitute {
			k = nk
		}
		if quote {
			if op == CONTAINS {
				v = wrap(v, '%')
			}
			if strings.IndexFunc(v, notDigit) > -1 {
				v = wrap(v, '\'')
				//v = "'" + v + "'"
			}
		}
		m[k] = paramVal{Val: v, Op: op}
	}
	return m
}

// TODO generic for all?
var paramAliases = map[string]string{
	"part":       "pn",
	"partnumber": "pn",

	"cat":         "category",
	"ds":          "datasheet",
	"description": "desc",
	"fam":         "family",
	"fn":          "func",
	"function":    "func",
	"loc":         "location",
	"mount":       "mounting",
	"mtg":         "mounting",
	"n/pkg":       "npkg",
	"mpfx":        "prefix",
	"pfx":         "prefix",
	"pkg":         "package",
	"sfx":         "suffix",
}

func notDigit(r rune) bool { return !unicode.IsDigit(r) }
func wrap(in string, r rune) string {
	if !strings.HasPrefix(in, string(r)) {
		in = string(r) + in
	}
	if !strings.HasSuffix(in, string(r)) {
		in = in + string(r)
	}
	return in
}

// turns parameter statements into the WHERE clause of a query
func toQueryWhere(kvs []string) (string, error) {
	var where []string
	pm := ToParamMap(kvs, true, true)
	for k, v := range pm {
		switch v.Op {
		case NE:
			where = append(where, fmt.Sprintf("NOT %s = %s", k, v.Val))
		default:
			where = append(where, fmt.Sprintf("%s %s %s", k, op2str[v.Op], v.Val))
		}
	}
	return strings.Join(where, " AND "), nil
}
func oldtoQueryWhere(kvs []string) (string, error) {
	var where []string
	validOperators := []string{">=", "<=", "==", ">", "<", "="}
nextKW:
	for _, kv := range kvs {
		for _, op := range validOperators {
			if idx := strings.Index(kv, op); idx > -1 {
				k := strings.ToLower(kv[:idx])
				v := kv[idx+len(op):]
				if subst, isAlias := paramAliases[k]; isAlias {
					k = subst
				}
				if op == "==" {
					op = "="
				}
				if op == "=" && strings.Contains(v, "%") {
					op = "LIKE"
				}
				if strings.IndexFunc(v, notDigit) > -1 {
					v = "'" + v + "'"
				}
				where = append(where, fmt.Sprintf("%s %s %s", k, op, v))
				continue nextKW
			}
		}
		return "", fmt.Errorf("parse error in %q", kv)
	}
	return strings.Join(where, " AND "), nil
}
