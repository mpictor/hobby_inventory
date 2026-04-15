package util

import (
	"database/sql"

	"github.com/dustin/go-humanize"
)

// humanize.SI, but without the space after the float
func SI(input float64, unit string, decimals int) string {
	value, prefix := humanize.ComputeSI(input)
	return humanize.FtoaWithDigits(value, decimals) + prefix + unit
}

// like SI, but returns empty string if invalid
func NullSI(input sql.NullFloat64, unit string, decimals int) string {
	if !input.Valid {
		return ""
	}
	value, prefix := humanize.ComputeSI(input.Float64)
	return humanize.FtoaWithDigits(value, decimals) + prefix + unit
}
