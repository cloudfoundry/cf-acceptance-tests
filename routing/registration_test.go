package routing

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gbytes"

	"github.com/cloudfoundry-incubator/cf-test-helpers/generator"
)

var _ = Describe("Registration", func() {
	var (
		systemDomain       string
		oauthPassword      string
		oauthUrl           string
		routingApiEndpoint string

		route     string
		routeJSON string
	)

	BeforeEach(func() {
		systemDomain = config.SystemDomain
		oauthPassword = config.ClientSecret
		oauthUrl = config.Protocol() + "uaa." + systemDomain
		routingApiEndpoint = config.Protocol() + "api." + systemDomain

		route = generator.RandomName()
		routeJSON = `[{"route":"` + route + `","port":65340,"ip":"1.2.3.4","ttl":60}]`
	})

	Describe("the routing API is enabled", func() {
		It("can register, list and unregister routes", func() {
			args := []string{"register", routeJSON, "--api", routingApiEndpoint, "--client-id", "gorouter", "--client-secret", oauthPassword, "--oauth-url", oauthUrl}
			session := Rtr(args...)
			Eventually(session.Out).Should(Say("Successfully registered routes"))

			args = []string{"list", "--api", routingApiEndpoint, "--client-id", "gorouter", "--client-secret", oauthPassword, "--oauth-url", oauthUrl}
			session = Rtr(args...)

			Eventually(session.Out).Should(Say(route))

			args = []string{"unregister", routeJSON, "--api", routingApiEndpoint, "--client-id", "gorouter", "--client-secret", oauthPassword, "--oauth-url", oauthUrl}
			session = Rtr(args...)

			Eventually(session.Out).Should(Say("Successfully unregistered routes"))

			args = []string{"list", "--api", routingApiEndpoint, "--client-id", "gorouter", "--client-secret", oauthPassword, "--oauth-url", oauthUrl}
			session = Rtr(args...)

			Eventually(session.Out).ShouldNot(Say(route))
		})
	})
})
