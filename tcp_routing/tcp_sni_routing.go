package tcp_routing

import (
	"fmt"
	"path/filepath"

	. "github.com/cloudfoundry/cf-acceptance-tests/cats_suite_helpers"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/app_helpers"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/assets"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/random_name"
	"github.com/cloudfoundry/cf-test-helpers/v2/cf"
	"github.com/cloudfoundry/cf-test-helpers/v2/workflowhelpers"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gexec"
)

// SNI TCP routing exercises the TNZ-81099 feature: multiple apps share the
// same external TCP port and are differentiated by an SNI hostname on the
// client TLS ClientHello. The TCP router terminates frontend TLS and forwards
// the plaintext stream to the correct backend based on SNI.
var _ = TCPSNIRoutingDescribe("TCP SNI Routing", func() {
	var domainName string

	BeforeEach(func() {
		domainName = Config.GetTCPDomain()
		workflowhelpers.AsUser(TestSetup.AdminUserContext(), Config.DefaultTimeoutDuration(), func() {
			routerGroupOutput := string(cf.Cf("router-groups").Wait().Out.Contents())
			Expect(routerGroupOutput).To(
				MatchRegexp(fmt.Sprintf("%s\\s+tcp", DefaultRouterGroupName)),
				fmt.Sprintf("Router group %s of type tcp doesn't exist", DefaultRouterGroupName),
			)

			Expect(cf.Cf("create-shared-domain",
				domainName,
				"--router-group", DefaultRouterGroupName,
			).Wait()).To(Exit())
		})
	})

	Context("two apps sharing a port via SNI", func() {
		var (
			appA, appB         string
			hostA              = "app-a"
			hostB              = "app-b"
			serverIdA          = "server-a"
			serverIdB          = "server-b"
			externalPort       int
			tcpDropletReceiver = assets.NewAssets().TCPListener
		)

		BeforeEach(func() {
			appA = random_name.CATSRandomName("APP-A")
			appB = random_name.CATSRandomName("APP-B")

			pushArgs := func(appName, serverId string) []string {
				return []string{
					"push",
					"--no-route",
					"--no-start",
					appName,
					"-p", tcpDropletReceiver,
					"-b", Config.GetGoBuildpackName(),
					"-m", DEFAULT_MEMORY_LIMIT,
					"-f", filepath.Join(tcpDropletReceiver, "manifest.yml"),
					"-c", fmt.Sprintf("tcp-listener --serverId=%s", serverId),
				}
			}

			Expect(cf.Cf(pushArgs(appA, serverIdA)...).Wait()).To(Exit(0))
			Expect(cf.Cf(pushArgs(appB, serverIdB)...).Wait()).To(Exit(0))

			// Allocate a port for appA by mapping its SNI route without an explicit port.
			externalPort = MapTCPRouteWithHostname(appA, domainName, hostA, 0)

			// Reuse the same port for appB, differentiated only by SNI hostname.
			MapTCPRouteWithHostname(appB, domainName, hostB, externalPort)

			Expect(cf.Cf("start", appA).Wait(Config.CfPushTimeoutDuration())).To(Exit(0))
			Expect(cf.Cf("start", appB).Wait(Config.CfPushTimeoutDuration())).To(Exit(0))
		})

		AfterEach(func() {
			app_helpers.AppReport(appA)
			app_helpers.AppReport(appB)
			Eventually(cf.Cf("delete", appA, "-f", "-r")).Should(Exit(0))
			Eventually(cf.Cf("delete", appB, "-f", "-r")).Should(Exit(0))
		})

		It("routes to the correct backend based on the SNI hostname", func() {
			sniA := fmt.Sprintf("%s.%s", hostA, domainName)
			sniB := fmt.Sprintf("%s.%s", hostB, domainName)

			respA, err := SendAndReceiveTLS(domainName, externalPort, sniA, true)
			Expect(err).ToNot(HaveOccurred())
			Expect(respA).To(ContainSubstring(serverIdA))

			respB, err := SendAndReceiveTLS(domainName, externalPort, sniB, true)
			Expect(err).ToNot(HaveOccurred())
			Expect(respB).To(ContainSubstring(serverIdB))
		})

		It("does not cross-talk: repeated requests for one hostname only hit that app", func() {
			sniA := fmt.Sprintf("%s.%s", hostA, domainName)

			responses, err := GetNTLSResponses(10, domainName, externalPort, sniA, true)
			Expect(err).ToNot(HaveOccurred())
			Expect(responses).To(HaveLen(10))
			for _, r := range responses {
				Expect(r).To(ContainSubstring(serverIdA))
				Expect(r).ToNot(ContainSubstring(serverIdB))
			}
		})
	})

	// Two apps mapped to the same port AND the same SNI hostname. HAProxy places both
	// backends in a single pool and round-robins across them. This is the SNI analogue
	// of the port-only "maps single external port to both applications" test.
	Context("two apps sharing the same port and same SNI hostname", func() {
		var (
			appA, appB         string
			sharedHost         = "shared-sni"
			serverIdA          = "server-shared-a"
			serverIdB          = "server-shared-b"
			externalPort       int
			tcpDropletReceiver = assets.NewAssets().TCPListener
		)

		BeforeEach(func() {
			appA = random_name.CATSRandomName("APP-A")
			appB = random_name.CATSRandomName("APP-B")

			pushArgs := func(appName, serverId string) []string {
				return []string{
					"push",
					"--no-route",
					"--no-start",
					appName,
					"-p", tcpDropletReceiver,
					"-b", Config.GetGoBuildpackName(),
					"-m", DEFAULT_MEMORY_LIMIT,
					"-f", filepath.Join(tcpDropletReceiver, "manifest.yml"),
					"-c", fmt.Sprintf("tcp-listener --serverId=%s", serverId),
				}
			}

			Expect(cf.Cf(pushArgs(appA, serverIdA)...).Wait()).To(Exit(0))
			Expect(cf.Cf(pushArgs(appB, serverIdB)...).Wait()).To(Exit(0))

			externalPort = MapTCPRouteWithHostname(appA, domainName, sharedHost, 0)
			MapTCPRouteWithHostname(appB, domainName, sharedHost, externalPort)

			Expect(cf.Cf("start", appA).Wait(Config.CfPushTimeoutDuration())).To(Exit(0))
			Expect(cf.Cf("start", appB).Wait(Config.CfPushTimeoutDuration())).To(Exit(0))
		})

		AfterEach(func() {
			app_helpers.AppReport(appA)
			app_helpers.AppReport(appB)
			Eventually(cf.Cf("delete-route", domainName,
				"--port", fmt.Sprintf("%d", externalPort),
				"--hostname", sharedHost, "-f",
			)).Should(Exit(0))
			Eventually(cf.Cf("delete", appA, "-f")).Should(Exit(0))
			Eventually(cf.Cf("delete", appB, "-f")).Should(Exit(0))
		})

		It("load-balances TLS connections across both apps on the shared SNI hostname", func() {
			sharedSNI := fmt.Sprintf("%s.%s", sharedHost, domainName)

			responses, err := GetNTLSResponses(10, domainName, externalPort, sharedSNI, true)
			Expect(err).ToNot(HaveOccurred())
			Expect(responses).To(ContainElement(ContainSubstring(serverIdA)))
			Expect(responses).To(ContainElement(ContainSubstring(serverIdB)))
		})
	})

	// One app reachable via two different SNI hostnames on the same port. HAProxy creates
	// two independent use_backend rules that both resolve to the same app's containers.
	// This is the SNI analogue of the port-only "maps both ports to the same application" test.
	Context("one app reachable via two different SNI hostnames on the same port", func() {
		var (
			appA               string
			hostOne            = "sni-host-one"
			hostTwo            = "sni-host-two"
			serverIdA          = "server-multi-sni"
			externalPort       int
			tcpDropletReceiver = assets.NewAssets().TCPListener
		)

		BeforeEach(func() {
			appA = random_name.CATSRandomName("APP-A")

			Expect(cf.Cf(
				"push",
				"--no-route",
				"--no-start",
				appA,
				"-p", tcpDropletReceiver,
				"-b", Config.GetGoBuildpackName(),
				"-m", DEFAULT_MEMORY_LIMIT,
				"-f", filepath.Join(tcpDropletReceiver, "manifest.yml"),
				"-c", fmt.Sprintf("tcp-listener --serverId=%s", serverIdA),
			).Wait()).To(Exit(0))

			externalPort = MapTCPRouteWithHostname(appA, domainName, hostOne, 0)
			MapTCPRouteWithHostname(appA, domainName, hostTwo, externalPort)

			Expect(cf.Cf("start", appA).Wait(Config.CfPushTimeoutDuration())).To(Exit(0))
		})

		AfterEach(func() {
			app_helpers.AppReport(appA)
			Eventually(cf.Cf("delete", appA, "-f", "-r")).Should(Exit(0))
		})

		It("reaches the same app via both SNI hostnames", func() {
			sniOne := fmt.Sprintf("%s.%s", hostOne, domainName)
			sniTwo := fmt.Sprintf("%s.%s", hostTwo, domainName)

			respOne, err := SendAndReceiveTLS(domainName, externalPort, sniOne, true)
			Expect(err).ToNot(HaveOccurred())
			Expect(respOne).To(ContainSubstring(serverIdA))

			respTwo, err := SendAndReceiveTLS(domainName, externalPort, sniTwo, true)
			Expect(err).ToNot(HaveOccurred())
			Expect(respTwo).To(ContainSubstring(serverIdA))
		})
	})
})
