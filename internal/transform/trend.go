package transform

// TrendData holds trend and direction information
type TrendData struct {
	Trend     int
	Direction string
}

// CareLinkTrendMap maps CareLink trend strings to Nightscout trend values
var CareLinkTrendMap = map[string]TrendData{
	"NONE":            {Trend: 0, Direction: "NONE"},
	"UP_TRIPLE":       {Trend: 1, Direction: "TripleUp"},
	"UP_DOUBLE":       {Trend: 1, Direction: "DoubleUp"},
	"UP":              {Trend: 2, Direction: "SingleUp"},
	"DOWN":            {Trend: 6, Direction: "SingleDown"},
	"DOWN_DOUBLE":     {Trend: 7, Direction: "DoubleDown"},
	"DOWN_TRIPLE":     {Trend: 7, Direction: "TripleDown"},
	"FORTY_FIVE_UP":   {Trend: 3, Direction: "FortyFiveUp"},
	"FORTY_FIVE_DOWN": {Trend: 5, Direction: "FortyFiveDown"},
	"FLAT":            {Trend: 4, Direction: "Flat"},
}
