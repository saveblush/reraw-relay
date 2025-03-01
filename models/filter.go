package models

type Timestamp int64

type TagMap map[string][]string

type Filter struct {
	IDs     []string   `json:"ids"`
	Kinds   []int      `json:"kinds"`
	Authors []string   `json:"authors"`
	Tags    TagMap     `json:"-"`
	Since   *Timestamp `json:"since"`
	Until   *Timestamp `json:"until"`
	Limit   int        `json:"limit"`
	Search  string     `json:"search"`
}

type Filters []Filter

type Subscription struct {
	ID      string
	Filters []Filter
}
