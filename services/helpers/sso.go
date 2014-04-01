package helpers

import (
	"fmt"
	"encoding/base64"
	"encoding/json"
	"regexp"
	"net/http"
	"net/url"
	"strings"

	. "github.com/pivotal-cf-experimental/cf-test-helpers/runner"
)

type OAuthConfig struct {
	RedirectUriPort string
	ClientId 				string
	ClientSecret 		string
	RedirectUri 		string
	RequestedScopes string
}

func ParseJsonResponse(response []byte) (resultMap map[string]interface{}) {
	var jsonValue interface{}

	json.Unmarshal(response, &jsonValue)
	resultMap = jsonValue.(map[string]interface{})
	return
}

func RegisterAuthCallbackHandler() {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		code := r.FormValue("code")
		w.Write([]byte(code))
	})
}

func StartListeningForAuthCallback(port int) {
	go http.ListenAndServe(fmt.Sprintf(":%d", port), nil)
}

func GetTokenEndpoint(apiEndpoint string) (tokenEndpoint string) {
	result        := Curl(fmt.Sprintf("%v/info", apiEndpoint)).FullOutput()
	jsonResult    := ParseJsonResponse(result)
	tokenEndpoint = fmt.Sprintf("%v/oauth", jsonResult[`token_endpoint`])
	return
}

func LogIntoTokenEndpoint(tokenEndpoint string, username string, password string) (cookie string) {
	loginUri         := fmt.Sprintf("%v/authorize/login.do", tokenEndpoint)
	usernameEncoded  := url.QueryEscape(username)
	passwordEncoded  := url.QueryEscape(password)
	loginCredentials := fmt.Sprintf("username=%v&password=%v", usernameEncoded, passwordEncoded)

	result       := Curl(loginUri, `--data`, loginCredentials, `--insecure`, `-i`).FullOutput()
	stringResult := string(result)

	regEx, _  := regexp.Compile(`JSESSIONID([^;]*)`)
	sessionId := regEx.FindString(stringResult)
	cookie    = fmt.Sprintf("\"%v\"", sessionId)
	return
}

func RequestScopes(tokenEndpoint string, cookie string, config OAuthConfig) (authCode string, httpCode string) {
	requestScopesUri := fmt.Sprintf("%v/authorize?client_id=%v&response_type=code+id_token&redirect_uri=%v&scope=%v",
		tokenEndpoint,
		url.QueryEscape(config.ClientId),
		config.RedirectUri,
		config.RequestedScopes)

	result     := Curl(requestScopesUri, `-L`, `--cookie`, cookie, `--insecure`, `-w`, `:TestReponseCode:%{http_code}`).FullOutput()
	resultBody := string(result)
	resultMap  := strings.Split(resultBody, `:TestReponseCode:`)

	resultText := resultMap[0]
	httpCode   = resultMap[1]

	if (httpCode == `200`) {
		if (strings.Contains(resultText, `authorize`)) {
			authCode = AuthorizeScopes(tokenEndpoint, cookie)
		} else {
			authCode = resultText
		}
	}

	return
}

func AuthorizeScopes(tokenEndpoint string, cookie string) (authCode string){
	authorizedScopes   := `scope.0=scope.openid&scope.1=scope.cloud_controller.read&scope.2=scope.cloud_controller.write&user_oauth_approval=true`
	authorizeScopesUri := fmt.Sprintf("%v/authorize", tokenEndpoint)
	result := Curl(authorizeScopesUri, `-L`, `--data`, authorizedScopes, `--cookie`, cookie, `--insecure`)

	authCode = string(result.FullOutput())
	return
}

func GetAccessToken(tokenEndpoint string, authCode string, config OAuthConfig) (accessToken string) {
	clientCredentials        := []byte(fmt.Sprintf("%v:%v", config.ClientId, config.ClientSecret))
	encodedClientCredentials := base64.StdEncoding.EncodeToString(clientCredentials)
	authHeader               := fmt.Sprintf("Authorization: Basic %v", encodedClientCredentials)
	requestTokenUri          := fmt.Sprintf("%v/token", tokenEndpoint)
	requestTokenData         := fmt.Sprintf("scope=%v&code=%v&grant_type=authorization_code&redirect_uri=%v", config.RequestedScopes, authCode, config.RedirectUri)

	result     := Curl(requestTokenUri, `-H`, authHeader, `--data`, requestTokenData, `--insecure`).FullOutput()
	jsonResult := ParseJsonResponse(result)

	accessToken = fmt.Sprintf("%v", jsonResult[`access_token`])
	return
}

func QueryServiceInstancePermissionEndpoint(apiEndpoint string, accessToken string, serviceInstanceGuid string) (canManage string, httpCode string) {
	canManage      = `not populated`
	authHeader     := fmt.Sprintf("Authorization: bearer %v", accessToken)
	permissionsUri := fmt.Sprintf("%v/v2/service_instances/%v/permissions", apiEndpoint, serviceInstanceGuid)

	result     := Curl(permissionsUri, `-H`, authHeader, `-w`, `:TestReponseCode:%{http_code}`, `--insecure`).FullOutput()
	resultBody := string(result)
	resultMap  := strings.Split(resultBody, `:TestReponseCode:`)

	resultText := resultMap[0]
	httpCode   = resultMap[1]

	if (httpCode == `200`) {
		jsonResult := ParseJsonResponse([]byte(resultText))
		canManage	= fmt.Sprintf("%v", jsonResult[`manage`])
	}

	return
}
