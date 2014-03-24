package cf

type UserContext struct {
	ApiUrl   string
	Username string
	Password string
	Org      string
	Space    string
}

var NewUserContext = func(apiUrl, username, password, org, space string) UserContext {
	uc := UserContext{}
	uc.ApiUrl = apiUrl
	uc.Username = username
	uc.Password = password
	uc.Org = org
	uc.Space = space
	return uc
}
