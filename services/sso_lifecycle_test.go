package services

import (
	"time"
	"fmt"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/vito/cmdtest/matchers"
	. "github.com/cloudfoundry/cf-acceptance-tests/helpers"
	. "github.com/cloudfoundry/cf-acceptance-tests/services/helpers"
	. "github.com/pivotal-cf-experimental/cf-test-helpers/cf"
		"github.com/pivotal-cf-experimental/cf-test-helpers/generator"
)

var _ = Describe("SSO Lifecycle", func() {
	var broker ServiceBroker
	var config OAuthConfig

	var (
		apiEndpoint,
		username,
		password string
	)

	var redirectUriPort int

	RegisterAuthCallbackHandler()

	BeforeEach(func() {
		LoginAsAdmin()

		apiEndpoint = LoadConfig().ApiEndpoint
		username    = RegularUserContext.Username
		password    = RegularUserContext.Password
	})

	JustBeforeEach(func() {
		StartListeningForAuthCallback(redirectUriPort)

		redirectUriPortString := fmt.Sprintf("%d", redirectUriPort)
		redirectUri           := fmt.Sprintf("http://localhost:%v", redirectUriPortString)

		broker = NewServiceBroker(generator.RandomName(), NewAssets().ServiceBroker)
		broker.Push()
		broker.Service.DashboardClient.RedirectUri = redirectUri
		broker.Configure()

		config = OAuthConfig{}
		config.RedirectUriPort = redirectUriPortString
		config.ClientId        = broker.Service.DashboardClient.ID
		config.ClientSecret    = broker.Service.DashboardClient.Secret
		config.RedirectUri     = redirectUri
		config.RequestedScopes = `openid,cloud_controller.read,cloud_controller.write`
	})

	AfterEach(func() {
		broker.Destroy()
		LoginAsUser()
	})

	Context("When a service broker is created", func() {
		BeforeEach(func() {
			redirectUriPort = 5551
		})

		It("can perform an operation on a user's behalf using sso", func() {
			defer Recover() // Catches panic thrown by Require expectations

			Require(Cf("create-service-broker", broker.Name, "username", "password", AppUri(broker.Name, "", LoadConfig().AppsDomain))).To(ExitWithTimeout(0, 20*time.Second))
			Expect(Cf("service-brokers")).To(Say(broker.Name))

			//create a service instance
			broker.PublicizePlans()
			serviceInstanceGuid := broker.CreateServiceInstance(generator.RandomName())

			// perform the OAuth lifecycle to obtain an access token
			tokenEndpoint              := GetTokenEndpoint(apiEndpoint)
			tokenEndpointSessionCookie := LogIntoTokenEndpoint(tokenEndpoint, username, password)

			authCode, _ := RequestScopes(tokenEndpoint, tokenEndpointSessionCookie, config)
			Expect(authCode).ToNot(BeNil(), `Failed to request and authorize scopes.`)

			accessToken := GetAccessToken(tokenEndpoint, authCode, config)
			Expect(accessToken).ToNot(BeNil(), `Failed to obtain an access token.`)

			// use the access token to perform an operation on the user's behalf
			canManage, httpCode := QueryServiceInstancePermissionEndpoint(apiEndpoint, accessToken, serviceInstanceGuid)

			Expect(httpCode).To(Equal(`200`), `The provided access token was not valid.`)
			Expect(canManage).To(Equal(`true`))
		})
	})

	Context("When a service broker is updated", func() {
		BeforeEach(func() {
			redirectUriPort = 5552
		})

		It("can perform an operation on a user's behalf using sso", func() {
			defer Recover() // Catches panic thrown by Require expectations

			Require(Cf("create-service-broker", broker.Name, "username", "password", AppUri(broker.Name, "", LoadConfig().AppsDomain))).To(ExitWithTimeout(0, 20*time.Second))
			Expect(Cf("service-brokers")).To(Say(broker.Name))

			config.ClientId = `new-client-id`
			broker.Service.DashboardClient.ID = config.ClientId
			broker.Configure()

			Require(Cf("update-service-broker", broker.Name, "username", "password", AppUri(broker.Name, "", LoadConfig().AppsDomain))).To(ExitWithTimeout(0, 20*time.Second))

			//create a service instance
			broker.PublicizePlans()
			serviceInstanceGuid := broker.CreateServiceInstance(generator.RandomName())

			// perform the OAuth lifecycle to obtain an access token
			tokenEndpoint              := GetTokenEndpoint(apiEndpoint)
			tokenEndpointSessionCookie := LogIntoTokenEndpoint(tokenEndpoint, username, password)

			authCode, _ := RequestScopes(tokenEndpoint, tokenEndpointSessionCookie, config)
			Expect(authCode).ToNot(BeNil(), `Failed to request and authorize scopes.`)

			accessToken := GetAccessToken(tokenEndpoint, authCode, config)
			Expect(accessToken).ToNot(BeNil(), `Failed to obtain an access token.`)

			// use the access token to perform an operation on the user's behalf
			canManage, httpCode := QueryServiceInstancePermissionEndpoint(apiEndpoint, accessToken, serviceInstanceGuid)

			Expect(httpCode).To(Equal(`200`), `The provided access token was not valid.`)
			Expect(canManage).To(Equal(`true`))
		})
	})

	Context("When a service broker is deleted", func() {
		BeforeEach(func() {
			redirectUriPort = 5553
		})

		It("can no longer perform an operation on a user's behalf using sso", func() {
			defer Recover() // Catches panic thrown by Require expectations

			Require(Cf("create-service-broker", broker.Name, "username", "password", AppUri(broker.Name, "", LoadConfig().AppsDomain))).To(ExitWithTimeout(0, 20*time.Second))
			Expect(Cf("service-brokers")).To(Say(broker.Name))

			Require(Cf("delete-service-broker", broker.Name, "-f")).To(ExitWithTimeout(0, 20*time.Second))
			Expect(Cf("service-brokers")).ToNot(Say(broker.Name))

			// perform the OAuth lifecycle to obtain an access token
			tokenEndpoint              := GetTokenEndpoint(apiEndpoint)
			tokenEndpointSessionCookie := LogIntoTokenEndpoint(tokenEndpoint, username, password)
			_, httpCode                := RequestScopes(tokenEndpoint, tokenEndpointSessionCookie, config)

			// there should not be a client in uaa anymore, so the request for scopes should return an unauthorized
			Expect(httpCode).To(Equal(`401`))
		})
	})
})
