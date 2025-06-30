package model

type Experience struct {
	Position string   `json:"position"`
	Company  string   `json:"company"`
	Start    string   `json:"start"`
	End      string   `json:"end"`
	Details  []string `json:"details"`
}
