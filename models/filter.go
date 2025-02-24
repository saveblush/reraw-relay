package models

import "time"

type Timestamp int64

func (t Timestamp) Time() time.Time {
	return time.Unix(int64(t), 0)
}

type Filter struct {
	IDs     []string   `json:"ids"`
	Kinds   []int      `json:"kinds"`
	Authors []string   `json:"authors"`
	Tags    TagMap     `json:"tags"`
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
