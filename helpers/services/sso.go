package services

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/url"
	"os"
	"regexp"
	"strings"

	"github.com/cloudfoundry/cf-acceptance-tests/helpers/random_name"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gexec"

	"github.com/cloudfoundry-incubator/cf-test-helpers/helpers"
	. "github.com/cloudfoundry/cf-acceptance-tests/cats_suite_helpers"
	cats_config "github.com/cloudfoundry/cf-acceptance-tests/helpers/config"
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

func SetOauthEndpoints(apiEndpoint string, oAuthConfig *OAuthConfig, config cats_config.CatsConfig) {
	args := []string{}
	if config.GetSkipSSLValidation() {
		args = append(args, "--insecure")
	}
	args = append(args, fmt.Sprintf("%v/info", apiEndpoint))
	curl := helpers.Curl(Config, args...).Wait(Config.DefaultTimeoutDuration())
	Expect(curl).To(Exit(0))
	apiResponse := curl.Out.Contents()
	jsonResult := ParseJsonResponse(apiResponse)

	oAuthConfig.TokenEndpoint = fmt.Sprintf("%v", jsonResult[`token_endpoint`])
	oAuthConfig.AuthorizationEndpoint = fmt.Sprintf("%v", jsonResult[`authorization_endpoint`])
	return
}

func AuthenticateUser(authorizationEndpoint string, username string, password string) (cookie string) {
	loginCsrfUri := fmt.Sprintf("%v/login", authorizationEndpoint)

	cookieFile, err := ioutil.TempFile("", random_name.CATSRandomName("CATS-CSRF-COOKIE"))
	Expect(err).ToNot(HaveOccurred())
	cookiePath := cookieFile.Name()
	defer func() {
		cookieFile.Close()
		os.Remove(cookiePath)
	}()

	curl := helpers.Curl(Config, loginCsrfUri, `--insecure`, `-i`, `-v`, `-c`, cookiePath).Wait(Config.DefaultTimeoutDuration())
	apiResponse := string(curl.Out.Contents())
	csrfRegEx, _ := regexp.Compile(`name="X-Uaa-Csrf" value="(.*)"`)
	csrfToken := csrfRegEx.FindStringSubmatch(apiResponse)[1]

	loginUri := fmt.Sprintf("%v/login.do", authorizationEndpoint)
	usernameEncoded := url.QueryEscape(username)
	passwordEncoded := url.QueryEscape(password)
	csrfTokenEncoded := url.QueryEscape(csrfToken)
	loginCredentials := fmt.Sprintf("username=%v&password=%v&X-Uaa-Csrf=%v", usernameEncoded, passwordEncoded, csrfTokenEncoded)

	curl = helpers.Curl(Config, loginUri, `--data`, loginCredentials, `--insecure`, `-i`, `-v`, `-b`, cookiePath).Wait(Config.DefaultTimeoutDuration())
	Expect(curl).To(Exit(0))
	apiResponse = string(curl.Out.Contents())

	jsessionRegEx, _ := regexp.Compile(`JSESSIONID([^;]*)`)
	vcapidRegEx, _ := regexp.Compile(`__VCAP_ID__([^;]*)`)
	sessionId := jsessionRegEx.FindString(apiResponse)
	vcapId := vcapidRegEx.FindString(apiResponse)
	cookie = fmt.Sprintf("%v;%v", sessionId, vcapId)
	return
}

func RequestScopes(cookie string, config OAuthConfig) (authCode string, httpCode string) {
	authCode = `initialized`

	requestScopesUri := fmt.Sprintf("%v/oauth/authorize?client_id=%v&response_type=code&redirect_uri=%v&scope=%v",
		config.AuthorizationEndpoint,
		url.QueryEscape(config.ClientId),
		config.RedirectUri,
		config.RequestedScopes)

	curl := helpers.Curl(Config, requestScopesUri, `-L`, `--cookie`, cookie, `--insecure`, `-w`, `:TestReponseCode:%{http_code}`, `-v`).Wait(Config.DefaultTimeoutDuration())
	Expect(curl).To(Exit(0))
	apiResponse := string(curl.Out.Contents())
	resultMap := strings.Split(apiResponse, `:TestReponseCode:`)

	httpCode = resultMap[1]

	if httpCode == `200` {
		authCode = AuthorizeScopes(cookie, config)
	}

	return
}

func AuthorizeScopes(cookie string, config OAuthConfig) (authCode string) {
	authorizedScopes := `scope.0=scope.openid&scope.1=scope.cloud_controller.read&scope.2=scope.cloud_controller.write&user_oauth_approval=true&X-Uaa-Csrf=123456`
	authorizeScopesUri := fmt.Sprintf("%v/oauth/authorize", config.AuthorizationEndpoint)

	curl := helpers.Curl(Config, authorizeScopesUri, `-i`, `--data`, authorizedScopes, `--cookie`, cookie+`;X-Uaa-Csrf=123456`, `--insecure`, `-v`).Wait(Config.DefaultTimeoutDuration())
	Expect(curl).To(Exit(0))
	apiResponse := string(curl.Out.Contents())

	pattern := fmt.Sprintf(`%v\?code=([a-zA-Z0-9]+)`, regexp.QuoteMeta(config.RedirectUri))
	regEx, _ := regexp.Compile(pattern)
	authCode = regEx.FindStringSubmatch(apiResponse)[1]

	return
}

func GetAccessToken(authCode string, config OAuthConfig) (accessToken string) {
	clientCredentials := []byte(fmt.Sprintf("%v:%v", config.ClientId, config.ClientSecret))
	encodedClientCredentials := base64.StdEncoding.EncodeToString(clientCredentials)
	authHeader := fmt.Sprintf("Authorization: Basic %v", encodedClientCredentials)
	requestTokenUri := fmt.Sprintf("%v/oauth/token", config.TokenEndpoint)
	requestTokenData := fmt.Sprintf("scope=%v&code=%v&grant_type=authorization_code&redirect_uri=%v", config.RequestedScopes, authCode, config.RedirectUri)

	curl := helpers.Curl(Config, requestTokenUri, `-H`, authHeader, `--data`, requestTokenData, `--insecure`, `-v`).Wait(Config.DefaultTimeoutDuration())
	Expect(curl).To(Exit(0))
	apiResponse := curl.Out.Contents()
	jsonResult := ParseJsonResponse(apiResponse)

	accessToken = fmt.Sprintf("%v", jsonResult[`access_token`])
	return
}

func QueryServiceInstancePermissionEndpoint(apiEndpoint string, accessToken string, serviceInstanceGuid string) (canManage string, httpCode string) {
	canManage = `not populated`
	authHeader := fmt.Sprintf("Authorization: bearer %v", accessToken)
	permissionsUri := fmt.Sprintf("%v/v2/service_instances/%v/permissions", apiEndpoint, serviceInstanceGuid)

	curl := helpers.Curl(Config, permissionsUri, `-H`, authHeader, `-w`, `:TestReponseCode:%{http_code}`, `--insecure`, `-v`).Wait(Config.DefaultTimeoutDuration())
	Expect(curl).To(Exit(0))
	apiResponse := string(curl.Out.Contents())
	resultMap := strings.Split(apiResponse, `:TestReponseCode:`)

	resultText := resultMap[0]
	httpCode = resultMap[1]

	if httpCode == `200` {
		jsonResult := ParseJsonResponse([]byte(resultText))
		canManage = fmt.Sprintf("%v", jsonResult[`manage`])
	}

	return
}
