package codecoogsauth

import "os"

// TODO: make this more sophisticated
func Authorize(token string) bool {
	secret := os.Getenv("AUTH_SECRET")

	return token == secret
}
