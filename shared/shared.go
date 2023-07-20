package shared

type Game struct {
	Area   []bool `json:"area"`
	Width  int    `json:"width"`
	Height int    `json:"height"`
}

var WorldWidth int = 10
var WorldHeight int = 10
