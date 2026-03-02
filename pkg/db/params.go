package db

import (
	"database/sql"
	"fmt"
	"strings"
)

func ParamsToRow(db *sql.DB, tblName string, args []string) (mainDBtbl, error) {
	// inv add ttl pn=F74AHCTLS999UB qty=55 fmax=1.21JigoHertz pkg=SO-8765 loc=nowhere
	// fields, err := fieldsForTable(inst, tbl)
	// if err != nil {
	// 	return err
	// }
	tbl := GetTbl(tblName)
	if tbl == nil {
		return nil, fmt.Errorf("%s is not a valid table name", tblName)
	}
	err := tbl.SetRow(db, args)
	if err != nil {
		return nil, err
	}
	return tbl, nil
}

type paramMap map[string]string

// turns slice of key=value into map[key]value, and performs
// substitutions in keyAliases
func toParamMap(kvs []string) paramMap {
	m := make(paramMap)
	for _, kv := range kvs {
		k, v, ok := strings.Cut(kv, "=")
		if !ok || len(k) == 0 || len(v) == 0 {
			continue
		}
		if nk, ok := paramAliases[k]; ok {
			k = nk
		}
		m[k] = v
	}
	return m
}

// TODO generic for all?
var paramAliases = map[string]string{
	"part":       "pn",
	"partnumber": "pn",

	"cat":   "category",
	"ds":    "datasheet",
	"desc":  "description",
	"fam":   "family",
	"fn":    "function",
	"func":  "function",
	"loc":   "location",
	"mount": "mounting",
	"mtg":   "mounting",
	"n/pkg": "npkg",
	"mpfx":  "prefix",
	"pfx":   "prefix",
	"pkg":   "package",
	"sfx":   "suffix",
}

// turns parameter statements into the WHERE clause of a query
func toQueryWhere(kvs []string) (string, error) {
	var where []string
	validOperators := []string{">=", "<=", "==", ">", "<", "="}
nextKW:
	for _, kv := range kvs {
		for _, op := range validOperators {
			if idx := strings.Index(kv, op); idx > -1 {
				k := kv[:idx]
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
				if strings.Contains(v, " ") /*|| strings.Contains(v, "%")*/ {
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
