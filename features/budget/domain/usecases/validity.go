package usecases

import "time"

// saoPauloLocation is the reference timezone for quote validity. Quote validity has
// date-only semantics ("valid until dd/mm/yyyy"), so dates are anchored to the end
// of the day in America/Sao_Paulo. Falls back to UTC if the zoneinfo DB is absent.
var saoPauloLocation = func() *time.Location {
	loc, err := time.LoadLocation("America/Sao_Paulo")
	if err != nil {
		return time.UTC
	}
	return loc
}()

// endOfDaySaoPaulo returns the last instant (23:59:59.999999999) of t's calendar day
// in America/Sao_Paulo. It gives a provided-or-computed validity date consistent
// date-only semantics regardless of the time component sent by the client.
func endOfDaySaoPaulo(t time.Time) time.Time {
	local := t.In(saoPauloLocation)
	y, m, d := local.Date()
	return time.Date(y, m, d, 23, 59, 59, int(time.Second-time.Nanosecond), saoPauloLocation)
}

// computeValidUntil returns the validity instant for a budget sent "now": the end of
// the day, in America/Sao_Paulo, that falls `days` days after now.
func computeValidUntil(now time.Time, days int) time.Time {
	if days <= 0 {
		days = 15
	}
	return endOfDaySaoPaulo(now.In(saoPauloLocation).AddDate(0, 0, days))
}

// normalizeValidUntil snaps a client-provided validity date to the end of its day in
// America/Sao_Paulo (date-only semantics). Nil passes through unchanged.
func normalizeValidUntil(t *time.Time) *time.Time {
	if t == nil {
		return nil
	}
	eod := endOfDaySaoPaulo(*t)
	return &eod
}
