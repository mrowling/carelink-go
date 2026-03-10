package transform

import "math"

// MgdlToMmol converts blood glucose from mg/dL to mmol/L
// Formula: mmol/L = mg/dL ÷ 18.018
// Returns value rounded to 1 decimal place
func MgdlToMmol(mgdl int) float64 {
	mmol := float64(mgdl) / 18.018
	return math.Round(mmol*10) / 10
}
