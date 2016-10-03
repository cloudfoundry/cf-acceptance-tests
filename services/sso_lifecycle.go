package services_test

import (
	"github.com/cloudfoundry-incubator/cf-test-helpers/cf"
	. "github.com/cloudfoundry/cf-acceptance-tests/cats_suite_helpers"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"

	"github.com/cloudfoundry/cf-acceptance-tests/helpers/app_helpers"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/assets"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/random_name"
	. "github.com/cloudfoundry/cf-acceptance-tests/helpers/services"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/skip_messages"
)

var _ = ServicesDescribe("SSO Lifecycle", func() {
	var broker ServiceBroker
	var oauthConfig OAuthConfig
	var apiEndpoint string

	redirectUri := `http://example.com`

	BeforeEach(func() {
		if !Config.GetIncludeSSO() {
			Skip(skip_messages.SkipSSOMessage)
		}
		broker = NewServiceBroker(
			random_name.CATSRandomName("BRKR"),
			assets.NewAssets().ServiceBroker,
			TestSetup,
		)
		broker.Push(Config)
		broker.Service.DashboardClient.RedirectUri = redirectUri
		broker.Configure()

		oauthConfig = OAuthConfig{}
		oauthConfig.ClientId = broker.Service.DashboardClient.ID
		oauthConfig.ClientSecret = broker.Service.DashboardClient.Secret
		oauthConfig.RedirectUri = redirectUri
		oauthConfig.RequestedScopes = `openid,cloud_controller_service_permissions.read`

		apiEndpoint = Config.GetApiEndpoint()
		SetOauthEndpoints(apiEndpoint, &oauthConfig, Config)

		broker.Create()
	})

	AfterEach(func() {
		app_helpers.AppReport(broker.Name, Config.DefaultTimeoutDuration())

		broker.Destroy()
	})

	Context("When a service broker is created with a dashboard client", func() {
		var instanceName string

		BeforeEach(func() {
			instanceName = random_name.CATSRandomName("SVC")
		})

		AfterEach(func() {
			Expect(cf.Cf("delete-service", instanceName, "-f").Wait(Config.DefaultTimeoutDuration())).To(gexec.Exit(0))
		})

		It("can perform an operation on a user's behalf using sso", func() {
			//create a service instance
			broker.PublicizePlans()
			serviceInstanceGuid := broker.CreateServiceInstance(instanceName)

			// perform the OAuth lifecycle to obtain an access token
			userSessionCookie := AuthenticateUser(oauthConfig.AuthorizationEndpoint, TestSetup.RegularUserContext().Username, TestSetup.RegularUserContext().Password)

			authCode, _ := RequestScopes(userSessionCookie, oauthConfig)
			Expect(authCode).ToNot(BeNil(), `Failed to request and authorize scopes.`)

			accessToken := GetAccessToken(authCode, oauthConfig)
			Expect(accessToken).ToNot(BeNil(), `Failed to obtain an access token.`)

			// use the access token to perform an operation on the user's behalf
			canManage, httpCode := QueryServiceInstancePermissionEndpoint(apiEndpoint, accessToken, serviceInstanceGuid)

			Expect(httpCode).To(Equal(`200`), `The provided access token was not valid.`)
			Expect(canManage).To(Equal(`true`))
		})
	})

	Context("When a service broker is updated with a new dashboard client", func() {
		var instanceName string

		BeforeEach(func() {
			instanceName = random_name.CATSRandomName("SVC")
		})

		AfterEach(func() {
			Expect(cf.Cf("delete-service", instanceName, "-f").Wait(Config.DefaultTimeoutDuration())).To(gexec.Exit(0))
		})

		It("can perform an operation on a user's behalf using sso", func() {
			oauthConfig.ClientId = random_name.CATSRandomName("CLIENT-ID")
			broker.Service.DashboardClient.ID = oauthConfig.ClientId
			broker.Configure()

			broker.Update()

			//create a service instance
			broker.PublicizePlans()
			serviceInstanceGuid := broker.CreateServiceInstance(instanceName)

			// perform the OAuth lifecycle to obtain an access token
			userSessionCookie := AuthenticateUser(oauthConfig.AuthorizationEndpoint, TestSetup.RegularUserContext().Username, TestSetup.RegularUserContext().Password)

			authCode, _ := RequestScopes(userSessionCookie, oauthConfig)
			Expect(authCode).ToNot(BeNil(), `Failed to request and authorize scopes.`)

			accessToken := GetAccessToken(authCode, oauthConfig)
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
			userSessionCookie := AuthenticateUser(oauthConfig.AuthorizationEndpoint, TestSetup.RegularUserContext().Username, TestSetup.RegularUserContext().Password)

			_, httpCode := RequestScopes(userSessionCookie, oauthConfig)

			// there should not be a client in uaa anymore, so the request for scopes should return an unauthorized
			Expect(httpCode).To(Equal(`401`))
		})
	})
})
