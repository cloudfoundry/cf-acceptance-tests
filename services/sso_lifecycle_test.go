package services_test

import (
	"strings"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gexec"

	"github.com/cloudfoundry-incubator/cf-test-helpers/cf"
	"github.com/cloudfoundry-incubator/cf-test-helpers/generator"
	"github.com/cloudfoundry-incubator/cf-test-helpers/helpers"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/assets"
	. "github.com/cloudfoundry/cf-acceptance-tests/helpers/services"
)

var _ = Describe("SSO Lifecycle", func() {
	var broker ServiceBroker
	var config OAuthConfig
	var redirectUri string
	var apiEndpoint = helpers.LoadConfig().ApiEndpoint

	Describe("For service clients (provided in catalog)", func() {

		BeforeEach(func() {
			redirectUri = `http://www.purple.com`

			broker = NewServiceBroker(generator.RandomName(), assets.NewAssets().ServiceBroker, context, true)
			broker.Push()
			broker.Service.DashboardClient.RedirectUri = redirectUri
			broker.Configure()

			config = OAuthConfig{}
			config.ClientId = broker.Service.DashboardClient.ID
			config.ClientSecret = broker.Service.DashboardClient.Secret
			config.RedirectUri = redirectUri
			config.RequestedScopes = `openid,cloud_controller_service_permissions.read`

			SetOauthEndpoints(apiEndpoint, &config)

			broker.Create()
		})

		AfterEach(func() {
			broker.Destroy()
		})

		Context("When a service broker is created with a service dashboard client", func() {
			It("can perform an operation on a user's behalf using sso", func() {
				//create a service instance
				broker.PublicizePlans()
				serviceInstanceGuid := broker.CreateServiceInstance(generator.RandomName())

				// perform the OAuth lifecycle to obtain an access token
				userSessionCookie := AuthenticateUser(config.AuthorizationEndpoint, context.RegularUserContext().Username, context.RegularUserContext().Password)

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

		Context("When a service broker is updated with a new service dashboard client", func() {
			It("can perform an operation on a user's behalf using sso", func() {
				config.ClientId = generator.RandomName()
				broker.Service.DashboardClient.ID = config.ClientId
				broker.Configure()

				broker.Update()

				//create a service instance
				broker.PublicizePlans()
				serviceInstanceGuid := broker.CreateServiceInstance(generator.RandomName())

				// perform the OAuth lifecycle to obtain an access token
				userSessionCookie := AuthenticateUser(config.AuthorizationEndpoint, context.RegularUserContext().Username, context.RegularUserContext().Password)

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
				broker.Delete()

				// perform the OAuth lifecycle to obtain an access token
				userSessionCookie := AuthenticateUser(config.AuthorizationEndpoint, context.RegularUserContext().Username, context.RegularUserContext().Password)

				_, httpCode := RequestScopes(userSessionCookie, config)

				// there should not be a client in uaa anymore, so the request for scopes should return an unauthorized
				Expect(httpCode).To(Equal(`401`))
			})
		})
	})

	XDescribe("For service instance dashboard clients", func() {
		BeforeEach(func() {
			redirectUri := `http://www.purple.com`

			broker = NewServiceBroker(generator.RandomName(), assets.NewAssets().ServiceBroker, context, true)
			broker.Push()
			broker.Service.DashboardClient.RedirectUri = redirectUri
			broker.Configure()

			config = OAuthConfig{}
			config.ClientId = broker.SsoPlans[0].DashboardClient.ID
			config.ClientSecret = broker.SsoPlans[0].DashboardClient.Secret
			config.RedirectUri = redirectUri
			config.RequestedScopes = `openid,cloud_controller_service_permissions.read`

			SetOauthEndpoints(apiEndpoint, &config)

			broker.Create()
			broker.EnableServiceAccess()
		})

		AfterEach(func() {
			broker.Destroy()
		})

		Context("When a service instance is created with a dashboard client", func() {

			It("can perform an operation on a user's behalf using sso", func() {
				//create a service instance
				instanceName := generator.RandomName()
				createService := cf.Cf("create-service", broker.Service.Name, broker.SsoPlans[0].Name, instanceName).Wait(DEFAULT_TIMEOUT)
				Expect(createService).To(Exit(0), "failed creating service")

				//get the service instance guid
				serviceInstance := cf.Cf("service", instanceName, "--guid").Wait(DEFAULT_TIMEOUT)
				Expect(serviceInstance).To(Exit(0), "failed creating service")
				serviceInstanceGuid := strings.Trim(string(serviceInstance.Out.Contents()), "\n")

				// perform the OAuth lifecycle to obtain an access token
				userSessionCookie := AuthenticateUser(config.AuthorizationEndpoint, context.RegularUserContext().Username, context.RegularUserContext().Password)

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
	})
})
