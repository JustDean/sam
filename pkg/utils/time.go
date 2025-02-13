package utils

import "time"

func GetNowTz() time.Time {
	loc, _ := time.LoadLocation("Local")
	return time.Now().In(loc)
}
