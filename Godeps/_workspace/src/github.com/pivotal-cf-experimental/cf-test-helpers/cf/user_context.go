package cf

type UserContext struct {
	ApiUrl     string
	Username   string
	Password   string
	Org        string
	Space      string

	LoginFlags string
}

func NewUserContext(apiUrl, username, password, org, space, loginFlags string) UserContext {
	return UserContext{
		ApiUrl: apiUrl,
		Username: username,
		Password: password,
		Org: org,
		Space: space,
		LoginFlags: loginFlags,
	}
}
