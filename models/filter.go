package models

import (
	"time"
)

type Timestamp int64

func (t Timestamp) Time() time.Time {
	return time.Unix(int64(t), 0)
}

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

/*func (ef Filter) Clone() Filter {
	clone := Filter{
		IDs:     slices.Clone(ef.IDs),
		Authors: slices.Clone(ef.Authors),
		Kinds:   slices.Clone(ef.Kinds),
		Limit:   ef.Limit,
		Search:  ef.Search,
	}

	if ef.Tags != nil {
		clone.Tags = make(TagMap, len(ef.Tags))
		for k, v := range ef.Tags {
			clone.Tags[k] = slices.Clone(v)
		}
	}

	if ef.Since != nil {
		since := *ef.Since
		clone.Since = &since
	}

	if ef.Until != nil {
		until := *ef.Until
		clone.Until = &until
	}

	return clone
}*/

type Filters []Filter

type Subscription struct {
	ID      string
	Filters []Filter
}
