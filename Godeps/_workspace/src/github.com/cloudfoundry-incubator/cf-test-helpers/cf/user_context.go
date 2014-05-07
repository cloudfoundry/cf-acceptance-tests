package cf

type UserContext struct {
	ApiUrl   string
	Username string
	Password string
	Org      string
	Space    string

	SkipSSLValidation bool
}

func NewUserContext(apiUrl, username, password, org, space string, skipSSLValidation bool) UserContext {
	return UserContext{
		ApiUrl:            apiUrl,
		Username:          username,
		Password:          password,
		Org:               org,
		Space:             space,
		SkipSSLValidation: skipSSLValidation,
	}
}
