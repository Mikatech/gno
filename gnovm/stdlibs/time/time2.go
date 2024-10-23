package time2

import "errors"

const (
	minWall = wallToInternal // year 1885

	// The year of the zero Time.
	// Assumed by the unixToInternal computation below.
	internalYear = 1

	// Offsets to convert between internal and absolute or Unix times.
	absoluteToInternal int64 = (absoluteZeroYear - internalYear) * 365.2425 * secondsPerDay
	internalToAbsolute       = -absoluteToInternal
	unixToInternal     int64 = (1969*365 + 1969/4 - 1969/100 + 1969/400) * secondsPerDay
	internalToUnix     int64 = -unixToInternal
	wallToInternal     int64 = (1884*365 + 1884/4 - 1884/100 + 1884/400) * secondsPerDay

	// The unsigned zero year for internal calculations.
	// Must be 1 mod 400, and times before it will not compute correctly,
	// but otherwise can be changed at will.
	absoluteZeroYear = -292277022399

	secondsPerMinute = 60
	secondsPerHour   = 60 * secondsPerMinute
	secondsPerDay    = 24 * secondsPerHour
	secondsPerWeek   = 7 * secondsPerDay
	daysPer400Years  = 365*400 + 97
	daysPer100Years  = 365*100 + 24
	daysPer4Years    = 365*4 + 1
)

type Time struct {
	sec  int64
	nsec int32
}

func (t *Time) unixSec() int64 { return t.sec + internalToUnix }

func (t Time) After(u Time) bool {
	return t.sec > u.sec || t.sec == u.sec && t.nsec > u.nsec
}

func (t Time) Before(u Time) bool {
	return t.sec < u.sec || t.sec == u.sec && t.nsec < u.nsec
}

func (t Time) Equal(u Time) bool {
	return t.sec == u.sec && t.nsec == u.nsec
}

type Month int

const (
	January Month = 1 + iota
	February
	March
	April
	May
	June
	July
	August
	September
	October
	November
	December
)

func (m Month) String() string {
	if January <= m && m <= December {
		return longMonthNames[m-1]
	}
	buf := make([]byte, 20)
	n := fmtInt(buf, uint64(m))
	return "%!Month(" + string(buf[n:]) + ")"
}

type Weekday int

const (
	Sunday Weekday = iota
	Monday
	Tuesday
	Wednesday
	Thursday
	Friday
	Saturday
)

func (d Weekday) String() string {
	if Sunday <= d && d <= Saturday {
		return longDayNames[d]
	}
	buf := make([]byte, 20)
	n := fmtInt(buf, uint64(d))
	return "%!Weekday(" + string(buf[n:]) + ")"
}

func (t Time) IsZero() bool {
	return t.sec == 0 && t.nsec == 0
}

// Date returns the year, month, and day in which t occurs.
func (t Time) Date() (year int, month Month, day int) {
	year, month, day, _ = t.date(true)
	return
}

// Year returns the year in which t occurs.
func (t Time) Year() int {
	year, _, _, _ := t.date(false)
	return year
}

// Month returns the month of the year specified by t.
func (t Time) Month() Month {
	_, month, _, _ := t.date(false)
	return month
}

// Day returns the day of the month specified by t.
func (t Time) Day() int {
	_, _, day, _ := t.date(true)
	return day
}

func (t Time) Weekday() Weekday {
	return internalWeekday(t.internal())
}

func internalWeekday(it uint64) Weekday {
	sec := (it + uint64(Monday)*secondsPerDay) % secondsPerWeek
	return Weekday(sec / secondsPerDay)
}

// ISOWeek returns the ISO 8601 year and week number in which t occurs.
// Week ranges from 1 to 53. Jan 01 to Jan 03 of year n might belong to
// week 52 or 53 of year n-1, and Dec 29 to Dec 31 might belong to week 1
// of year n+1.
func (t Time) ISOWeek() (year, week int) {
	// According to the rule that the first calendar week of a calendar year is
	// the week including the first Thursday of that year, and that the last one is
	// the week immediately preceding the first calendar week of the next calendar year.
	// See https://www.iso.org/obp/ui#iso:std:iso:8601:-1:ed-1:v1:en:term:3.1.1.23 for details.

	// weeks start with Monday
	// Monday Tuesday Wednesday Thursday Friday Saturday Sunday
	// 1      2       3         4        5      6        7
	// +3     +2      +1        0        -1     -2       -3
	// the offset to Thursday
	it := t.internal()
	d := Thursday - internalWeekday(it)
	// handle Sunday
	if d == 4 {
		d -= 3
	}
	// find the Thursday of the calendar week
	it += uint64(d) * secondsPerDay
	year, _, _, yday := internalDate(it, false)
	return year, yday/7 + 1
}

// Clock returns the hour, minute, and second within the day specified by t.
func (t Time) Clock() (hour, min, sec int) {
	return internalClock(t.internal())
}

// internalClock is like clock but operates on an internal time.
func internalClock(it uint64) (hour, min, sec int) {
	sec = int(it % secondsPerDay)
	hour = sec / secondsPerHour
	sec -= hour * secondsPerHour
	min = sec / secondsPerMinute
	sec -= min * secondsPerMinute
	return
}

// Hour returns the hour within the day specified by t, in the range [0, 23].
func (t Time) Hour() int {
	return int(t.internal()%secondsPerDay) / secondsPerHour
}

// Minute returns the minute offset within the hour specified by t, in the range [0, 59].
func (t Time) Minute() int {
	return int(t.internal()%secondsPerHour) / secondsPerMinute
}

// Second returns the second offset within the minute specified by t, in the range [0, 59].
func (t Time) Second() int {
	return int(t.internal() % secondsPerMinute)
}

// Nanosecond returns the nanosecond offset within the second specified by t,
// in the range [0, 999999999].
func (t Time) Nanosecond() int {
	return int(t.nsec)
}

// YearDay returns the day of the year specified by t, in the range [1,365] for non-leap years,
// and [1,366] in leap years.
func (t Time) YearDay() int {
	_, _, _, yday := t.date(false)
	return yday + 1
}

// AddDate returns the time corresponding to adding the
// given number of years, months, and days to t.
// For example, AddDate(-1, 2, 3) applied to January 1, 2011
// returns March 4, 2010.
//
// AddDate normalizes its result in the same way that Date does,
// so, for example, adding one month to October 31 yields
// December 1, the normalized form for November 31.
func (t Time) AddDate(years int, months int, days int) Time {
	year, month, day := t.Date()
	hour, min, sec := t.Clock()
	return Date(year+years, month+Month(months), day+days, hour, min, sec, int(t.nsec))
}

// rename of the "abs" method in the original time.go
// since we removed the location & zoneinfo related code
func (t Time) internal() uint64 {
	sec := t.unixSec()
	return uint64(sec + (unixToInternal + internalToAbsolute))
}

func (t Time) date(full bool) (year int, month Month, day int, yday int) {
	return internalDate(t.internal(), full)
}

func internalDate(it uint64, full bool) (year int, month Month, day int, yday int) {
	d := it / secondsPerDay

	// Account for 400 year cycles.
	n := d / daysPer400Years
	y := 400 * n
	d -= daysPer400Years * n

	// Cut off 100-year cycles.
	// The last cycle has one extra leap year, so on the last day
	// of that year, day / daysPer100Years will be 4 instead of 3.
	// Cut it back down to 3 by subtracting n>>2.
	n = d / daysPer100Years
	n -= n >> 2
	y += 100 * n
	d -= daysPer100Years * n

	// Cut off 4-year cycles.
	// The last cycle has a missing leap year, which does not
	// affect the computation.
	n = d / daysPer4Years
	y += 4 * n
	d -= daysPer4Years * n

	// Cut off years within a 4-year cycle.
	// The last year is a leap year, so on the last day of that year,
	// day / 365 will be 4 instead of 3. Cut it back down to 3
	// by subtracting n>>2.
	n = d / 365
	n -= n >> 2
	y += n
	d -= 365 * n

	year = int(int64(y) + absoluteZeroYear)
	yday = int(d)

	if !full {
		return
	}

	day = yday
	if isLeap(year) {
		switch {
		case day > 31+29-1:
			// After leap day; pretend it wasn't there
			day--
		case day == 31+29-1:
			// Leap day
			month = February
			day = 29
			return
		}
	}

	// Estimate month on assumption that every month has 31 days.
	// The estimate may be too low by at most one month, so adjust.
	month = Month(day / 31)
	end := int(daysBefore[month+1])
	var begin int
	if day >= end {
		month++
		begin = end
	} else {
		begin = int(daysBefore[month])
	}

	month++ // because January is 1
	day = day - begin + 1
	return
}

// daysBefore[m] counts the number of days in a non-leap year
// before month m begins. There is an entry for m=12, counting
// the number of days before January of next year (365).
var daysBefore = [...]int32{
	0,
	31,
	31 + 28,
	31 + 28 + 31,
	31 + 28 + 31 + 30,
	31 + 28 + 31 + 30 + 31,
	31 + 28 + 31 + 30 + 31 + 30,
	31 + 28 + 31 + 30 + 31 + 30 + 31,
	31 + 28 + 31 + 30 + 31 + 30 + 31 + 31,
	31 + 28 + 31 + 30 + 31 + 30 + 31 + 31 + 30,
	31 + 28 + 31 + 30 + 31 + 30 + 31 + 31 + 30 + 31,
	31 + 28 + 31 + 30 + 31 + 30 + 31 + 31 + 30 + 31 + 30,
	31 + 28 + 31 + 30 + 31 + 30 + 31 + 31 + 30 + 31 + 30 + 31,
}

// daysSinceEpoch takes a year and returns the number of days from
// the absolute epoch to the start of that year.
// This is basically (year - zeroYear) * 365, but accounting for leap days.
func daysSinceEpoch(year int) uint64 {
	y := uint64(int64(year) - absoluteZeroYear)

	// Add in days from 400-year cycles.
	n := y / 400
	y -= 400 * n
	d := daysPer400Years * n

	// Add in 100-year cycles.
	n = y / 100
	y -= 100 * n
	d += daysPer100Years * n

	// Add in 4-year cycles.
	n = y / 4
	y -= 4 * n
	d += daysPer4Years * n

	// Add in non-leap years.
	n = y
	d += 365 * n

	return d
}

const (
	Nanosecond  Duration = 1
	Microsecond          = 1000 * Nanosecond
	Millisecond          = 1000 * Microsecond
	Second               = 1000 * Millisecond
	Minute               = 60 * Second
	Hour                 = 60 * Minute
)

// A Duration represents the elapsed time between two instants
// as an int64 nanosecond count. The representation limits the
// largest representable duration to approximately 290 years.
type Duration int64

const (
	minDuration Duration = -1 << 63
	maxDuration Duration = 1<<63 - 1
)

// String returns a string representing the duration in the form "72h3m0.5s".
// Leading zero units are omitted. As a special case, durations less than one
// second format use a smaller unit (milli-, micro-, or nanoseconds) to ensure
// that the leading digit is non-zero. The zero duration formats as 0s.
func (d Duration) String() string {
	// Largest time is 2540400h10m10.000000000s
	var buf [32]byte
	w := len(buf)

	u := uint64(d)
	neg := d < 0
	if neg {
		u = -u
	}

	if u < uint64(Second) {
		// Special case: if duration is smaller than a second,
		// use smaller units, like 1.2ms
		var prec int
		w--
		buf[w] = 's'
		w--
		switch {
		case u == 0:
			return "0s"
		case u < uint64(Microsecond):
			// print nanoseconds
			prec = 0
			buf[w] = 'n'
		case u < uint64(Millisecond):
			// print microseconds
			prec = 3
			// U+00B5 'µ' micro sign == 0xC2 0xB5
			w-- // Need room for two bytes.
			copy(buf[w:], "µ")
		default:
			// print milliseconds
			prec = 6
			buf[w] = 'm'
		}
		w, u = fmtFrac(buf[:w], u, prec)
		w = fmtInt(buf[:w], u)
	} else {
		w--
		buf[w] = 's'

		w, u = fmtFrac(buf[:w], u, 9)

		// u is now integer seconds
		w = fmtInt(buf[:w], u%60)
		u /= 60

		// u is now integer minutes
		if u > 0 {
			w--
			buf[w] = 'm'
			w = fmtInt(buf[:w], u%60)
			u /= 60

			// u is now integer hours
			// Stop at hours because days can be different lengths.
			if u > 0 {
				w--
				buf[w] = 'h'
				w = fmtInt(buf[:w], u)
			}
		}
	}

	if neg {
		w--
		buf[w] = '-'
	}

	return string(buf[w:])
}

// Nanoseconds returns the duration as an integer nanosecond count.
func (d Duration) Nanoseconds() int64 { return int64(d) }

// Microseconds returns the duration as an integer microsecond count.
func (d Duration) Microseconds() int64 { return int64(d) / 1e3 }

// Milliseconds returns the duration as an integer millisecond count.
func (d Duration) Milliseconds() int64 { return int64(d) / 1e6 }

// These methods return float64 because the dominant
// use case is for printing a floating point number like 1.5s, and
// a truncation to integer would make them not useful in those cases.
// Splitting the integer and fraction ourselves guarantees that
// converting the returned float64 to an integer rounds the same
// way that a pure integer conversion would have, even in cases
// where, say, float64(d.Nanoseconds())/1e9 would have rounded
// differently.

// Seconds returns the duration as a floating point number of seconds.
func (d Duration) Seconds() float64 {
	sec := d / Second
	nsec := d % Second
	return float64(sec) + float64(nsec)/1e9
}

// Minutes returns the duration as a floating point number of minutes.
func (d Duration) Minutes() float64 {
	min := d / Minute
	nsec := d % Minute
	return float64(min) + float64(nsec)/(60*1e9)
}

// Hours returns the duration as a floating point number of hours.
func (d Duration) Hours() float64 {
	hour := d / Hour
	nsec := d % Hour
	return float64(hour) + float64(nsec)/(60*60*1e9)
}

// Truncate returns the result of rounding d toward zero to a multiple of m.
// If m <= 0, Truncate returns d unchanged.
func (d Duration) Truncate(m Duration) Duration {
	if m <= 0 {
		return d
	}
	return d - d%m
}

// lessThanHalf reports whether x+x < y but avoids overflow,
// assuming x and y are both positive (Duration is signed).
func lessThanHalf(x, y Duration) bool {
	return uint64(x)+uint64(x) < uint64(y)
}

// Round returns the result of rounding d to the nearest multiple of m.
// The rounding behavior for halfway values is to round away from zero.
// If the result exceeds the maximum (or minimum)
// value that can be stored in a Duration,
// Round returns the maximum (or minimum) duration.
// If m <= 0, Round returns d unchanged.
func (d Duration) Round(m Duration) Duration {
	if m <= 0 {
		return d
	}
	r := d % m
	if d < 0 {
		r = -r
		if lessThanHalf(r, m) {
			return d + r
		}
		if d1 := d - m + r; d1 < d {
			return d1
		}
		return minDuration // overflow
	}
	if lessThanHalf(r, m) {
		return d - r
	}
	if d1 := d + m - r; d1 > d {
		return d1
	}
	return maxDuration // overflow
}

// Add returns the time t+d
func (t Time) Add(d Duration) Time {
	dsec := int64(d / 1e9)
	nsec := t.nsec + int32(d%1e9)
	if nsec >= 1e9 {
		dsec++
		nsec -= 1e9
	} else if nsec < 0 {
		dsec--
		nsec += 1e9
	}
	return Time{t.sec + dsec, nsec}
}

// Sub returns the duration t-u. If the result exceeds the maximum (or minimum)
// value that can be stored in a Duration, the maximum (or minimum) duration
// will be returned.
// To compute t-d for a duration d, use t.Add(-d).
func (t Time) Sub(u Time) Duration {
	sec := t.sec - u.sec
	nsec := t.nsec - u.nsec
	if sec > 0 && nsec < 0 {
		sec--
		nsec += 1e9
	} else if sec < 0 && nsec > 0 {
		sec++
		nsec -= 1e9
	}
	if sec > int64(maxDuration) {
		return maxDuration
	}
	if sec < int64(minDuration) {
		return minDuration
	}
	return Duration(sec*1e9 + int64(nsec))
}

// Since returns the time elapsed since t.
// It is shorthand for time.Now().Sub(t).
func Since(t Time) Duration {
	return Now().Sub(t)
}

// Until returns the duration until t.
// It is shorthand for t.Sub(time.Now()).
func Until(t Time) Duration {
	return t.Sub(Now())
}

// Date returns the Time corresponding to
//
//	yyyy-mm-dd hh:mm:ss + nsec nanoseconds
//
// The month, day, hour, min, sec, and nsec values may be outside
// their usual ranges and will be normalized during the conversion.
// For example, October 32 converts to November 1.
func Date(year int, month Month, day, hour, min, sec, nsec int) Time {
	// Normalize month, overflowing into year.
	m := int(month) - 1
	year, m = norm(year, m, 12)
	month = Month(m) + 1

	// Normalize nsec, sec, min, hour, overflowing into day.
	sec, nsec = norm(sec, nsec, 1e9)
	min, sec = norm(min, sec, 60)
	hour, min = norm(hour, min, 60)
	day, hour = norm(day, hour, 24)

	// Compute days since the absolute epoch.
	d := daysSinceEpoch(year)

	// Add in days before this month.
	d += uint64(daysBefore[month-1])
	if isLeap(year) && month >= March {
		d++ // February 29
	}

	// Add in days before today.
	d += uint64(day - 1)

	// Add in time elapsed today.
	abs := d * secondsPerDay
	abs += uint64(hour*secondsPerHour + min*secondsPerMinute + sec)

	unix := int64(abs) + (absoluteToInternal + internalToUnix)
	t := unixTime(unix, int32(nsec))
	return t
}

// Abs returns the absolute value of d.
// As a special case, math.MinInt64 is converted to math.MaxInt64.
func (d Duration) Abs() Duration {
	switch {
	case d >= 0:
		return d
	case d == minDuration:
		return maxDuration
	default:
		return -d
	}
}

func now() (sec int64, nsec int32, mono int64) // injected by runtime

func Now() Time {
	sec, nsec, _ := now()
	sec += unixToInternal - minWall
	return Time{sec, nsec}
}

// Unix returns t as a Unix time, the number of seconds elapsed
// since January 1, 1970 UTC. The result does not depend on the
// location associated with t.
// Unix-like operating systems often record time as a 32-bit
// count of seconds, but since the method here returns a 64-bit
// value it is valid for billions of years into the past or future.
func (t Time) Unix() int64 {
	return t.unixSec()
}

// UnixMilli returns t as a Unix time, the number of milliseconds elapsed since
// January 1, 1970 UTC. The result is undefined if the Unix time in
// milliseconds cannot be represented by an int64 (a date more than 292 million
// years before or after 1970). The result does not depend on the
// location associated with t.
func (t Time) UnixMilli() int64 {
	return t.unixSec()*1e3 + int64(t.nsec)/1e6
}

// UnixMicro returns t as a Unix time, the number of microseconds elapsed since
// January 1, 1970 UTC. The result is undefined if the Unix time in
// microseconds cannot be represented by an int64 (a date before year -290307 or
// after year 294246). The result does not depend on the location associated
// with t.
func (t Time) UnixMicro() int64 {
	return t.unixSec()*1e6 + int64(t.nsec)/1e3
}

// UnixNano returns t as a Unix time, the number of nanoseconds elapsed
// since January 1, 1970 UTC. The result is undefined if the Unix time
// in nanoseconds cannot be represented by an int64 (a date before the year
// 1678 or after 2262). Note that this means the result of calling UnixNano
// on the zero Time is undefined. The result does not depend on the
// location associated with t.
func (t Time) UnixNano() int64 {
	return (t.unixSec())*1e9 + int64(t.nsec)
}

// MarshalBinary implements the encoding.BinaryMarshaler interface.
func (t Time) MarshalBinary() ([]byte, error) {
	sec := t.sec
	nsec := t.nsec
	enc := []byte{
		//encode seconds (int64) / bytes 0 to 7
		byte(sec >> 56),
		byte(sec >> 48),
		byte(sec >> 40),
		byte(sec >> 32),
		byte(sec >> 24),
		byte(sec >> 16),
		byte(sec >> 8),
		byte(sec),
		//encode nanoseconds (int32) / bytes 8 to 11
		byte(nsec >> 24),
		byte(nsec >> 16),
		byte(nsec >> 8),
		byte(nsec),
	}
	return enc, nil
}

// UnmarshalBinary implements the encoding.BinaryUnmarshaler interface.
func (t *Time) UnmarshalBinary(data []byte) error {
	buf := data
	if len(buf) == 0 {
		return errors.New("Time.UnmarshalBinary: no data")
	}
	if len(buf) != 12 { // 8 bytes for sec (int64) + 4 bytes for nsec (int32)
		return errors.New("Time.UnmarshalBinary: invalid length")
	}
	sec := int64(buf[7]) | int64(buf[6])<<8 | int64(buf[5])<<16 | int64(buf[4])<<24 |
		int64(buf[3])<<32 | int64(buf[2])<<40 | int64(buf[1])<<48 | int64(buf[0])<<56

	buf = buf[8:]
	nsec := int32(buf[3]) | int32(buf[2])<<8 | int32(buf[1])<<16 | int32(buf[0])<<24

	*t = Time{}
	t.sec = sec
	t.nsec = nsec

	return nil
}

// MarshalJSON implements the json.Marshaler interface.
// The time is a quoted string in RFC 3339 format, with sub-second precision added if present.
func (t Time) MarshalJSON() ([]byte, error) {
	if y := t.Year(); y < 0 || y >= 10000 {
		// RFC 3339 is clear that years are 4 digits exactly.
		// See golang.org/issue/4556#c15 for more discussion.
		return nil, errors.New("Time.MarshalJSON: year outside of range [0,9999]")
	}

	b := make([]byte, 0, len(RFC3339Nano)+2)
	b = append(b, '"')
	b = t.AppendFormat(b, RFC3339Nano)
	b = append(b, '"')
	return b, nil
}

// UnmarshalJSON implements the json.Unmarshaler interface.
// The time is expected to be a quoted string in RFC 3339 format.
func (t *Time) UnmarshalJSON(data []byte) error {
	// Ignore null, like in the main JSON package.
	if string(data) == "null" {
		return nil
	}
	// Fractional seconds are handled implicitly by Parse.
	var err error
	*t, err = Parse(`"`+RFC3339+`"`, string(data))
	return err
}

// MarshalText implements the encoding.TextMarshaler interface.
// The time is formatted in RFC 3339 format, with sub-second precision added if present.
func (t Time) MarshalText() ([]byte, error) {
	if y := t.Year(); y < 0 || y >= 10000 {
		return nil, errors.New("Time.MarshalText: year outside of range [0,9999]")
	}

	b := make([]byte, 0, len(RFC3339Nano))
	return t.AppendFormat(b, RFC3339Nano), nil
}

// UnmarshalText implements the encoding.TextUnmarshaler interface.
// The time is expected to be in RFC 3339 format.
func (t *Time) UnmarshalText(data []byte) error {
	// Fractional seconds are handled implicitly by Parse.
	var err error
	*t, err = Parse(RFC3339, string(data))
	return err
}

// Unix returns the local Time corresponding to the given Unix time,
// sec seconds and nsec nanoseconds since January 1, 1970 UTC.
// It is valid to pass nsec outside the range [0, 999999999].
// Not all sec values have a corresponding time value. One such
// value is 1<<63-1 (the largest int64 value).
func Unix(sec int64, nsec int64) Time {
	if nsec < 0 || nsec >= 1e9 {
		n := nsec / 1e9
		sec += n
		nsec -= n * 1e9
		if nsec < 0 {
			nsec += 1e9
			sec--
		}
	}
	return unixTime(sec, int32(nsec))
}

// UnixMilli returns the local Time corresponding to the given Unix time,
// msec milliseconds since January 1, 1970 UTC.
func UnixMilli(msec int64) Time {
	return Unix(msec/1e3, (msec%1e3)*1e6)
}

// UnixMicro returns the local Time corresponding to the given Unix time,
// usec microseconds since January 1, 1970 UTC.
func UnixMicro(usec int64) Time {
	return Unix(usec/1e6, (usec%1e6)*1e3)
}

func unixTime(sec int64, nsec int32) Time {
	return Time{sec + unixToInternal, nsec}
}

func isLeap(year int) bool {
	return year%4 == 0 && (year%100 != 0 || year%400 == 0)
}

// fmtFrac formats the fraction of v/10**prec (e.g., ".12345") into the
// tail of buf, omitting trailing zeros. It omits the decimal
// point too when the fraction is 0. It returns the index where the
// output bytes begin and the value v/10**prec.
func fmtFrac(buf []byte, v uint64, prec int) (nw int, nv uint64) {
	// Omit trailing zeros up to and including decimal point.
	w := len(buf)
	isprint := false
	for i := 0; i < prec; i++ {
		digit := v % 10
		isprint = isprint || digit != 0
		if isprint {
			w--
			buf[w] = byte(digit) + '0'
		}
		v /= 10
	}
	if isprint {
		w--
		buf[w] = '.'
	}
	return w, v
}

// add v at the end of buf and return the index where the number starts in buf
func fmtInt(buf []byte, v uint64) int {
	w := len(buf)
	if v == 0 {
		w--
		buf[w] = '0'
	} else {
		for v > 0 {
			w--
			buf[w] = byte(v%10) + '0'
			v /= 10
		}
	}
	return w
}

// norm returns nhi, nlo such that
//
//	hi * base + lo == nhi * base + nlo
//	0 <= nlo < base
func norm(hi, lo, base int) (nhi, nlo int) {
	if lo < 0 {
		n := (-lo-1)/base + 1
		hi -= n
		lo += n * base
	}
	if lo >= base {
		n := lo / base
		hi += n
		lo -= n * base
	}
	return hi, lo
}
