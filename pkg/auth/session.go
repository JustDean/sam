package auth

import "time"

type Session struct {
	Id           string    `db:"id"`
	ValidThrough time.Time `db:"valid_through"`
	Username     string    `db:"username"`
}
