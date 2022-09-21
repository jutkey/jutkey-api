package sql

import "time"

func MsToSeconds(millisecond int64) int64 {
	return time.UnixMilli(millisecond).Unix()
}

func GetFirstDateOfMonth(z int, d time.Time) time.Time {
	secondsEastOfUTC := int((time.Duration(z) * time.Hour).Seconds())
	zone := time.FixedZone("zone Time", secondsEastOfUTC)
	ts := d.In(zone)
	ts = ts.AddDate(0, 0, -ts.Day()+1)
	return GetZeroTime(ts)
}

func GetZeroTime(d time.Time) time.Time {
	return time.Date(d.Year(), d.Month(), d.Day(), 0, 0, 0, 0, d.Location())
}

func GetLastDateOfMonthToInt64(z int, t int64) int64 {
	secondsEastOfUTC := int((time.Duration(z) * time.Hour).Seconds())
	zone := time.FixedZone("zone Time", secondsEastOfUTC)
	d := time.Unix(t, 0)
	ts := d.In(zone)
	ts = ts.AddDate(0, 1, 0)
	return ts.Unix()
}

func GetZoneTimes() time.Time {
	d := time.Now()
	ts := d.In(time.UTC)
	return GetZeroTime(ts)
}
