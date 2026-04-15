package db

import (
	"fmt"
	"maps"
	"strings"

	// "github.com/xmonader/abbrev"
	"github.com/mpictor/hobby_inventory/pkg/abbrev"
)

type CompTbl int

const (
	CTUnknown CompTbl = iota
	CTLogic
	CTTransistors
	CTDevKits
	CTDiodeTVS
	CTLineDrv
	CTOpamp
	CTOpto
	CTOther
	CTPassive
	CTPower
	CTTmrOscPll
	CT_LAST // must be last value
)

var (
	ctAliases = map[string]CompTbl{
		"ttl":     CTLogic,
		"cmos":    CTLogic,
		"q":       CTTransistors,
		"bipolar": CTTransistors,
		"fet":     CTTransistors,
		"mosfet":  CTTransistors,
		"dk":      CTDevKits,
		"devkit":  CTDevKits,
		"d":       CTDiodeTVS,
		// "diode":   CTDiodeTVS,
		"tvs": CTDiodeTVS,
		// "line": CTLineDrv,
		// "opamp":   CTOpamp,
		// "other":   CTOther,
		"pwr": CTPower,
		// "tmr": CTTmrOscPll,
		"osc": CTTmrOscPll,
		"pll": CTTmrOscPll,
	}
	ct2str = map[CompTbl]string{
		CTLogic:       "logic",
		CTTransistors: "transistors",
		CTDevKits:     "dev_kits",
		CTDiodeTVS:    "diode_tvs",
		CTLineDrv:     "line_drv",
		CTOpamp:       "opamps",
		CTOpto:        "opto",
		CTOther:       "others",
		CTPassive:     "passive",
		CTPower:       "power",
		CTTmrOscPll:   "tmr_osc_pll",
	}
	str2ct = map[string]CompTbl{
		"logic":       CTLogic,
		"transistors": CTTransistors,
		"dev_kits":    CTDevKits,
		"diode_tvs":   CTDiodeTVS,
		"line_drv":    CTLineDrv,
		"opamps":      CTOpamp,
		"opto":        CTOpto,
		"others":      CTOther,
		"passive":     CTPassive,
		"power":       CTPower,
		"tmr_osc_pll": CTTmrOscPll,
	}
	ctAbbrev map[string]CompTbl
)

// TODO also consider something like fuzzy matching? e.g. Sixeight/go-fuzzaldrin
func init() {
	words := make(map[string]CompTbl, len(str2ct)+len(ctAliases))
	maps.Copy(words, str2ct)
	maps.Copy(words, ctAliases)

	// find all unique abbreviations
	ctAbbrev = abbrev.AbbrevMap(words)
}

// returns input slice, minus any occurrences of x
func exclude[T comparable](in []T, x T) []T {
	for i := 0; i < len(in); {
		if in[i] == x {
			in = append(in[:i], in[i+1:]...)
			continue
		}
		i++
	}
	return in
}

func compTblAbbrevs() []string {
	cta := make(map[CompTbl][]string)
	for k, v := range ctAbbrev {
		cta[v] = append(cta[v], k)
	}
	tblMax := 0
	for k := range cta {
		tblMax = max(tblMax, len(k.String()))
	}
	strs := make([]string, 2, len(cta)+2)
	strs[0] = fmt.Sprintf("%-*s | %s", tblMax, "table", "aliases & abbreviations")
	strs[1] = strings.Repeat("-", tblMax+1) + "|" + strings.Repeat("-", len(strs[0])-tblMax-2)

	for k, v := range cta {
		strs = append(strs, fmt.Sprintf("%-*s | %s", tblMax, k.String(), strings.Join(exclude(v, k.String()), ", ")))
	}
	return strs
}

// TODO sort owlphabetically
func CompTblsAndAliases(v bool) []string {
	if v {
		return compTblAbbrevs()
	}
	var tbls = make([]string, int(CT_LAST))
	var aliases = make([]string, int(CT_LAST))
	tblMax := 0
	for i := range CT_LAST {
		tbls[i] = CompTbl(i).String()
		tblMax = max(tblMax, len(tbls[i]))
	}
	// tblMax++
	for k, v := range ctAliases {
		if len(aliases[v]) == 0 {
			aliases[v] = k
			continue
		}
		aliases[v] = aliases[v] + ", " + k
	}
	for i, v := range aliases {
		aliases[i] = fmt.Sprintf("%-*s | %s", tblMax, tbls[i], v)
	}
	legend := fmt.Sprintf("%-*s | %s", tblMax, "table", "aliases")
	div := strings.Repeat("-", tblMax+1) + "|" + strings.Repeat("-", len(legend)-tblMax+3)

	// legend := fmt.Sprintf("%-*s:  %s", tblMax, "table", "aliases")
	// div := strings.Repeat("-", len(legend))
	// aliases[0] (CTUnknown) is not a table, so exclude
	return append([]string{legend, div}, aliases[1:]...)
}

func ParseCompTbl(s string) (CompTbl, error) {
	ct, ok := ctAbbrev[s]
	if ok {
		return ct, nil
	}
	return CTUnknown, fmt.Errorf("unknown component table %q", s)
}

func (c CompTbl) String() string {
	if s, ok := ct2str[c]; ok {
		return s
	}
	return "unknown"
}

// returns go type corresponding to the given db table, or nil
func GetTbl(tbl CompTbl) mainDBtbl {
	switch tbl {
	case CTLogic:
		return &Logic{}
	case CTTransistors:
		return &Transistors{}
	case CTDevKits:
		return &Dev_kits{}
	case CTDiodeTVS:
		return &Diode_TVSDevs{}
	case CTLineDrv:
		return &Line_Drv{}
	case CTOpamp:
		return &Opamps{}
	case CTOpto:
		return &OptoDevs{}
	case CTOther:
		return &Others{}
	case CTPassive:
		return &PassiveDevs{}
	case CTPower:
		return &PowerDevs{}
	case CTTmrOscPll:
		return &Tmr_Osc_PllDevs{}
	}
	return nil
}
