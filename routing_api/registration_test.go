package routing_api

import (
	"os/exec"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gbytes"
	. "github.com/onsi/gomega/gexec"

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
		oauthPassword = config.OauthPassword
		oauthUrl = "http://uaa." + systemDomain
		routingApiEndpoint = "http://routing-api." + systemDomain

		route = generator.RandomName()
		routeJSON = `[{"route":"` + route + `","port":65340,"ip":"1.2.3.4","ttl":60}]`
	})

	Describe("the routing API is enabled", func() {
		It("can register, list and unregister routes", func() {
			args := []string{"register", routeJSON, "--api", routingApiEndpoint, "--oauth-name", "gorouter", "--oauth-password", oauthPassword, "--oauth-url", oauthUrl}
			session := Rtr(args...)

			Eventually(session.Out).Should(Say("Successfully registered routes"))

			args = []string{"list", "--api", routingApiEndpoint, "--oauth-name", "gorouter", "--oauth-password", oauthPassword, "--oauth-url", oauthUrl}
			session = Rtr(args...)

			Eventually(session.Out).Should(Say(route))

			args = []string{"unregister", routeJSON, "--api", routingApiEndpoint, "--oauth-name", "gorouter", "--oauth-password", oauthPassword, "--oauth-url", oauthUrl}
			session = Rtr(args...)

			Eventually(session.Out).Should(Say("Successfully unregistered routes"))

			args = []string{"list", "--api", routingApiEndpoint, "--oauth-name", "gorouter", "--oauth-password", oauthPassword, "--oauth-url", oauthUrl}
			session = Rtr(args...)

			Eventually(session.Out).ShouldNot(Say(route))
		})
	})
})

func Rtr(args ...string) *Session {
	session, err := Start(exec.Command("rtr", args...), GinkgoWriter, GinkgoWriter)
	Expect(err).NotTo(HaveOccurred())

	return session
}
