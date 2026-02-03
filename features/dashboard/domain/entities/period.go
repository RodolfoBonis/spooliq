package entities

import "time"

// Period represents a time period for dashboard filtering.
type Period string

// Period constants
const (
	Period7D  Period = "7d"
	Period30D Period = "30d"
	Period3M  Period = "3m"
	Period6M  Period = "6m"
	Period1Y  Period = "1y"
	PeriodAll Period = "all"
)

// ParsePeriod parses a string into a Period, defaulting to 30d.
func ParsePeriod(s string) Period {
	switch s {
	case "7d":
		return Period7D
	case "30d":
		return Period30D
	case "3m":
		return Period3M
	case "6m":
		return Period6M
	case "1y":
		return Period1Y
	case "all":
		return PeriodAll
	default:
		return Period30D
	}
}

// ToTimeRange returns (start, end) for the current period and (prevStart, prevEnd) for previous period comparison
func (p Period) ToTimeRange() (start, end, prevStart, prevEnd time.Time) {
	now := time.Now()
	end = now

	switch p {
	case Period7D:
		start = now.AddDate(0, 0, -7)
		prevStart = now.AddDate(0, 0, -14)
		prevEnd = start
	case Period30D:
		start = now.AddDate(0, 0, -30)
		prevStart = now.AddDate(0, 0, -60)
		prevEnd = start
	case Period3M:
		start = now.AddDate(0, -3, 0)
		prevStart = now.AddDate(0, -6, 0)
		prevEnd = start
	case Period6M:
		start = now.AddDate(0, -6, 0)
		prevStart = now.AddDate(-1, 0, 0)
		prevEnd = start
	case Period1Y:
		start = now.AddDate(-1, 0, 0)
		prevStart = now.AddDate(-2, 0, 0)
		prevEnd = start
	case PeriodAll:
		start = time.Time{} // zero time
		prevStart = time.Time{}
		prevEnd = time.Time{}
	}
	return
}

// TruncateFunc returns the date_trunc interval for SQL grouping
func (p Period) TruncateFunc() string {
	switch p {
	case Period7D, Period30D:
		return "day"
	default:
		return "month"
	}
}
