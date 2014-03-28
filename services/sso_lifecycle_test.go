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

	StartListeningForAuthCallback(config.RedirectUriPort)

	BeforeEach(func() {
		LoginAsAdmin()
		broker = NewServiceBroker(generator.RandomName(), NewAssets().ServiceBroker)
		broker.Push()
		broker.Configure()

		config.RedirectUriPort = `5551`
		config.ClientId        = broker.Service.DashboardClient.ID
		config.ClientSecret    = broker.Service.DashboardClient.Secret
		config.RedirectUri     = fmt.Sprintf("http://localhost:%v", config.RedirectUriPort)
		config.RequestedScopes = `openid,cloud_controller.read,cloud_controller.write`

		apiEndpoint = LoadConfig().ApiEndpoint
		username    = RegularUserContext.Username
		password    = RegularUserContext.Password
	})

	AfterEach(func() {
		Require(Cf("delete", broker.Name, "-f")).To(ExitWithTimeout(0, 20*time.Second))
		LoginAsUser()
	})

	Context("When a service broker is created", func() {
		It("can perform an operation on a user's behalf using sso", func() {
			defer Recover() // Catches panic thrown by Require expectations

			Require(Cf("create-service-broker", broker.Name, "username", "password", AppUri(broker.Name, "", LoadConfig().AppsDomain))).To(ExitWithTimeout(0, 20*time.Second))
			Expect(Cf("service-brokers")).To(Say(broker.Name))

			// perform the OAuth lifecycle to obtain an access token
			tokenEndpoint              := GetTokenEndpoint(apiEndpoint)
			tokenEndpointSessionCookie := LogIntoTokenEndpoint(tokenEndpoint, username, password)
			authCode, _                := RequestScopes(tokenEndpoint, tokenEndpointSessionCookie, config)
			accessToken                := GetAccessToken(tokenEndpoint, authCode, config)

			// use the access token to perform an operation on the user's behalf
			canManage := QueryServiceInstancePermissionEndpoint(apiEndpoint, accessToken, `made-up-guid`)

			// we don't really care about true or false, either result means we were able to communicate with the endpoint
			Expect(canManage).To(Equal(false))
		})
	})

	Context("When a service broker is updated", func() {
		It("can perform an operation on a user's behalf using sso", func() {
			defer Recover() // Catches panic thrown by Require expectations

			Require(Cf("create-service-broker", broker.Name, "username", "password", AppUri(broker.Name, "", LoadConfig().AppsDomain))).To(ExitWithTimeout(0, 20*time.Second))
			Expect(Cf("service-brokers")).To(Say(broker.Name))

			config.ClientId = `new-client-id`
			broker.Configure()

			Require(Cf("update-service-broker", broker.Name, "username", "password", AppUri(broker.Name, "", LoadConfig().AppsDomain))).To(ExitWithTimeout(0, 20*time.Second))

			// perform the OAuth lifecycle to obtain an access token
			tokenEndpoint              := GetTokenEndpoint(apiEndpoint)
			tokenEndpointSessionCookie := LogIntoTokenEndpoint(tokenEndpoint, username, password)
			authCode, _                := RequestScopes(tokenEndpoint, tokenEndpointSessionCookie, config)
			accessToken                := GetAccessToken(tokenEndpoint, authCode, config)

			// use the access token to perform an operation on the user's behalf
			canManage := QueryServiceInstancePermissionEndpoint(apiEndpoint, accessToken, `made-up-guid`)

			// we don't really care about true or false, either result means we were able to communicate with the endpoint
			Expect(canManage).To(Equal(false))
		})
	})

	Context("When a service broker is updated", func() {
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

			Expect(httpCode).To(Equal(`401`))
		})
	})
})
