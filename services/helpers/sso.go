package helpers

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/url"
	"regexp"
	"strings"

	. "github.com/pivotal-cf-experimental/cf-test-helpers/runner"
)

type OAuthConfig struct {
	ClientId              string
	ClientSecret          string
	RedirectUri           string
	RequestedScopes       string
	AuthorizationEndpoint string
	TokenEndpoint         string
}

func ParseJsonResponse(response []byte) (resultMap map[string]interface{}) {
	var jsonValue interface{}

	json.Unmarshal(response, &jsonValue)
	resultMap = jsonValue.(map[string]interface{})
	return
}

func SetOauthEndpoints(apiEndpoint string, config *OAuthConfig) {
	result := Curl(fmt.Sprintf("%v/info", apiEndpoint)).FullOutput()
	jsonResult := ParseJsonResponse(result)

	config.TokenEndpoint = fmt.Sprintf("%v", jsonResult[`token_endpoint`])
	config.AuthorizationEndpoint = fmt.Sprintf("%v", jsonResult[`authorization_endpoint`])
	return
}

func AuthenticateUser(authorizationEndpoint string, username string, password string) (cookie string) {
	loginUri := fmt.Sprintf("%v/login.do", authorizationEndpoint)
	usernameEncoded := url.QueryEscape(username)
	passwordEncoded := url.QueryEscape(password)
	loginCredentials := fmt.Sprintf("username=%v&password=%v", usernameEncoded, passwordEncoded)

	result := Curl(loginUri, `--data`, loginCredentials, `--insecure`, `-i`, `-v`)
	stringResult := string(result.FullOutput())

	jsessionRegEx, _ := regexp.Compile(`JSESSIONID([^;]*)`)
	vcapidRegEx, _ := regexp.Compile(`__VCAP_ID__([^;]*)`)
	sessionId := jsessionRegEx.FindString(stringResult)
	vcapId := vcapidRegEx.FindString(stringResult)
	cookie = fmt.Sprintf("\"%v;%v\"", sessionId,vcapId)
	return
}

func RequestScopes(cookie string, config OAuthConfig) (authCode string, httpCode string) {
	authCode = `initialized`

	requestScopesUri := fmt.Sprintf("%v/oauth/authorize?client_id=%v&response_type=code+id_token&redirect_uri=%v&scope=%v",
		config.AuthorizationEndpoint,
		url.QueryEscape(config.ClientId),
		config.RedirectUri,
		config.RequestedScopes)

	result := Curl(requestScopesUri, `-L`, `--cookie`, cookie, `--insecure`, `-w`, `:TestReponseCode:%{http_code}`, `-v`)
	resultString := string(result.FullOutput())
	resultMap := strings.Split(resultString, `:TestReponseCode:`)

	httpCode = resultMap[1]

	if httpCode == `200` {
		authCode = AuthorizeScopes(cookie, config)
	}

	return
}

func AuthorizeScopes(cookie string, config OAuthConfig) (authCode string) {
	authorizedScopes := `scope.0=scope.openid&scope.1=scope.cloud_controller.read&scope.2=scope.cloud_controller.write&user_oauth_approval=true`
	authorizeScopesUri := fmt.Sprintf("%v/oauth/authorize", config.AuthorizationEndpoint)

	result := Curl(authorizeScopesUri, `-i`, `--data`, authorizedScopes, `--cookie`, cookie, `--insecure`, `-v`)
	stringResult := string(result.FullOutput())

	pattern := fmt.Sprintf(`%v\?code=([a-zA-Z0-9]+)`, regexp.QuoteMeta(config.RedirectUri))
	regEx, _ := regexp.Compile(pattern)
	authCode = regEx.FindStringSubmatch(stringResult)[1]

	return
}

func GetAccessToken(authCode string, config OAuthConfig) (accessToken string) {
	clientCredentials := []byte(fmt.Sprintf("%v:%v", config.ClientId, config.ClientSecret))
	encodedClientCredentials := base64.StdEncoding.EncodeToString(clientCredentials)
	authHeader := fmt.Sprintf("Authorization: Basic %v", encodedClientCredentials)
	requestTokenUri := fmt.Sprintf("%v/oauth/token", config.TokenEndpoint)
	requestTokenData := fmt.Sprintf("scope=%v&code=%v&grant_type=authorization_code&redirect_uri=%v", config.RequestedScopes, authCode, config.RedirectUri)

	result := Curl(requestTokenUri, `-H`, authHeader, `--data`, requestTokenData, `--insecure`, `-v`)
	jsonResult := ParseJsonResponse(result.FullOutput())

	accessToken = fmt.Sprintf("%v", jsonResult[`access_token`])
	return
}

func QueryServiceInstancePermissionEndpoint(apiEndpoint string, accessToken string, serviceInstanceGuid string) (canManage string, httpCode string) {
	canManage = `not populated`
	authHeader := fmt.Sprintf("Authorization: bearer %v", accessToken)
	permissionsUri := fmt.Sprintf("%v/v2/service_instances/%v/permissions", apiEndpoint, serviceInstanceGuid)

	result := Curl(permissionsUri, `-H`, authHeader, `-w`, `:TestReponseCode:%{http_code}`, `--insecure`, `-v`)
	resultBody := string(result.FullOutput())
	resultMap := strings.Split(resultBody, `:TestReponseCode:`)

	resultText := resultMap[0]
	httpCode = resultMap[1]

	if httpCode == `200` {
		jsonResult := ParseJsonResponse([]byte(resultText))
		canManage = fmt.Sprintf("%v", jsonResult[`manage`])
	}

	return
}
