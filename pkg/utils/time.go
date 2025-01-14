package utils

import "time"

func GetNowTz() (time.Time, error) {
	loc, err := time.LoadLocation("Local")
	if err != nil {
		return time.Now(), err
	}
	currentTime := time.Now().In(loc)
	futureTime := currentTime.AddDate(0, 0, 10)
	return futureTime, nil
}
