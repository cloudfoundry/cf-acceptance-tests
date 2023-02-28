package util

func TokenIsPresent(token string) bool {
	return token != "" && token != "revoked"
}
