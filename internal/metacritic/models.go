package metacritic

import (
	"fmt"
	"log/slog"
	"slices"
	"time"
)

type Medium string

type Action string

const (
	TV Medium = "tv"
	//Game  Medium = "game"
	// PC and Switch are Games but specified uniquely
	// in the URL since platforms need to be set.
	PC     Medium = "pc"
	Switch Medium = "switch"
	Movie  Medium = "movie"
	SAVED  Action = "saved"
)

var AvailableMediums = []Medium{
	TV, PC, Switch, Movie,
}

func (m Medium) ToPath() string {
	if m == PC {
		return "game/pc"
	} else if m == Switch {
		return "game/nintendo-switch"
	}
	return string(m)
}

type Options struct {
	Medium  Medium `form:"medium" binding:"required"`
	MinYear int    `form:"min_year" binding:"required"`
	MaxYear int    `form:"max_year" binding:"required"`
	Page    int    `form:"page"`
}

func (opts Options) RangeAsEpoch() (int64, int64) {
	l, u := opts.RangeAsDate()
	return l.Unix(), u.Unix()
}

func (opts Options) RangeAsDate() (time.Time, time.Time) {
	firstMoment := func(year int) time.Time {
		return time.Date(year, 1, 1, 0, 0, 0, 0, time.UTC)
	}
	l := firstMoment(opts.MinYear)
	u := firstMoment(opts.MaxYear + 1).Add(-time.Second)
	return l, u
}

func (opts Options) IsValid() bool {
	if !slices.Contains(AvailableMediums, opts.Medium) {
		return false
	} else if opts.MinYear < 1900 || opts.MaxYear < 1900 {
		return false
	} else if opts.MinYear > opts.MaxYear {
		return false
	}

	return true
}

func (opts Options) Group() slog.Attr {
	return slog.Group("options",
		slog.String("medium", string(opts.Medium)),
		slog.Int("min_year", opts.MinYear),
		slog.Int("max_year", opts.MaxYear))
}

type ProductCard struct {
	Title       string    `json:"title"`
	Href        string    `json:"href"`
	Score       int       `json:"score"`
	Description string    `json:"description"`
	ReleaseDate time.Time `json:"release_date"`
}

type Post struct {
	ProductCard
	ID          int64     `json:"id"`
	Medium      Medium    `json:"-"`
	RequestedAt time.Time `json:"-"`
}

func (p Post) String() string {
	s := fmt.Sprintf(`Title:       %v
ReleaseYear: %v
Score:       %v`,
		p.Title, p.ReleaseDate.Year(), p.Score)
	return s
}

type ShortPost struct {
	Title  string `json:"title"`
	Medium string `json:"medium"`
}
