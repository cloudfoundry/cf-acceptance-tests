package services

import (
	. "github.com/cloudfoundry/cf-acceptance-tests/helpers"
	. "github.com/cloudfoundry/cf-acceptance-tests/services/helpers"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/pivotal-cf-experimental/cf-test-helpers/generator"
)

var _ = Describe("SSO Lifecycle", func() {
	var broker ServiceBroker
	var config OAuthConfig
	var apiEndpoint = LoadConfig().ApiEndpoint

	redirectUri := `http://example.com`

	BeforeEach(func() {
		broker = NewServiceBroker(generator.RandomName(), NewAssets().ServiceBroker)
		broker.Push()
		broker.Service.DashboardClient.RedirectUri = redirectUri
		broker.Configure()

		config = OAuthConfig{}
		config.ClientId = broker.Service.DashboardClient.ID
		config.ClientSecret = broker.Service.DashboardClient.Secret
		config.RedirectUri = redirectUri
		config.RequestedScopes = `openid,cloud_controller.read,cloud_controller.write`

		SetOauthEndpoints(apiEndpoint, &config)
	})

	AfterEach(func() {
		broker.Destroy()
	})

	Context("When a service broker is created", func() {
		It("can perform an operation on a user's behalf using sso", func() {
			broker.Create(LoadConfig().AppsDomain)

			//create a service instance
			broker.PublicizePlans()
			serviceInstanceGuid := broker.CreateServiceInstance(generator.RandomName())

			// perform the OAuth lifecycle to obtain an access token
			userSessionCookie := AuthenticateUser(config.AuthorizationEndpoint, RegularUserContext.Username, RegularUserContext.Password)

			authCode, _ := RequestScopes(userSessionCookie, config)
			Expect(authCode).ToNot(BeNil(), `Failed to request and authorize scopes.`)

			accessToken := GetAccessToken(authCode, config)
			Expect(accessToken).ToNot(BeNil(), `Failed to obtain an access token.`)

			// use the access token to perform an operation on the user's behalf
			canManage, httpCode := QueryServiceInstancePermissionEndpoint(apiEndpoint, accessToken, serviceInstanceGuid)

			Expect(httpCode).To(Equal(`200`), `The provided access token was not valid.`)
			Expect(canManage).To(Equal(`true`))
		})
	})

	Context("When a service broker is updated", func() {
		It("can perform an operation on a user's behalf using sso", func() {
			broker.Create(LoadConfig().AppsDomain)

			config.ClientId = generator.RandomName()
			broker.Service.DashboardClient.ID = config.ClientId
			broker.Configure()

			broker.Update(LoadConfig().AppsDomain)

			//create a service instance
			broker.PublicizePlans()
			serviceInstanceGuid := broker.CreateServiceInstance(generator.RandomName())

			// perform the OAuth lifecycle to obtain an access token
			userSessionCookie := AuthenticateUser(config.AuthorizationEndpoint, RegularUserContext.Username, RegularUserContext.Password)

			authCode, _ := RequestScopes(userSessionCookie, config)
			Expect(authCode).ToNot(BeNil(), `Failed to request and authorize scopes.`)

			accessToken := GetAccessToken(authCode, config)
			Expect(accessToken).ToNot(BeNil(), `Failed to obtain an access token.`)

			// use the access token to perform an operation on the user's behalf
			canManage, httpCode := QueryServiceInstancePermissionEndpoint(apiEndpoint, accessToken, serviceInstanceGuid)

			Expect(httpCode).To(Equal(`200`), `The provided access token was not valid.`)
			Expect(canManage).To(Equal(`true`))
		})
	})

	Context("When a service broker is deleted", func() {
		It("can no longer perform an operation on a user's behalf using sso", func() {
			broker.Create(LoadConfig().AppsDomain)

			broker.Delete()

			// perform the OAuth lifecycle to obtain an access token
			userSessionCookie := AuthenticateUser(config.AuthorizationEndpoint, RegularUserContext.Username, RegularUserContext.Password)

			_, httpCode := RequestScopes(userSessionCookie, config)

			// there should not be a client in uaa anymore, so the request for scopes should return an unauthorized
			Expect(httpCode).To(Equal(`401`))
		})
	})
})
