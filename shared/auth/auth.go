package auth

type Auth int

const (
	NO_AUTH Auth = iota
	USER_PASS
)

type Credentials struct {
	Auth       Auth
	Username   string
	Password   string
}

