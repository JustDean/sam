package auth

type User struct {
	Id       string `json:"id"`
	Username string `json:"username"`
	password string
}
