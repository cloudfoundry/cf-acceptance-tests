package windows

import (
	"fmt"
	"os"
	"path/filepath"

	. "github.com/cloudfoundry/cf-acceptance-tests/cats_suite_helpers"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/app_helpers"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/assets"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/random_name"
	"github.com/cloudfoundry/cf-test-helpers/v2/cf"
	"github.com/cloudfoundry/cf-test-helpers/v2/workflowhelpers"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
	. "github.com/onsi/gomega/gexec"
)

const DefaultRouterGroupName = "default-tcp"

var _ = WindowsTCPRoutingDescribe("Windows TCP Routing", func() {
	var domainName string
	var compiledApp string

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

		originalDir, err := os.Getwd()
		Expect(err).NotTo(HaveOccurred())
		err = os.Chdir(assets.NewAssets().TCPListener)
		Expect(err).NotTo(HaveOccurred())
		compiledApp, err = gexec.BuildWithEnvironment(".", []string{"GOOS=windows"})
		Expect(err).NotTo(HaveOccurred())
		err = os.Chdir(originalDir)
		Expect(err).NotTo(HaveOccurred())
	})

	Context("external ports", func() {
		var (
			appName            string
			tcpDropletReceiver = assets.NewAssets().TCPListener
			serverId1          = "server1"
			externalPort1      string
		)

		BeforeEach(func() {
			appName = random_name.CATSRandomName("APP")
			cmd := fmt.Sprintf("tcp-listener --serverId=%s", serverId1)

			Expect(cf.Cf("push",
				"--no-route",
				"--no-start",
				appName,
				"-p", compiledApp,
				"-b", Config.GetBinaryBuildpackName(),
				"-s", Config.GetWindowsStack(),
				"-m", DEFAULT_MEMORY_LIMIT,
				"-f", filepath.Join(tcpDropletReceiver, "manifest.yml"),
				"-c", cmd,
			).Wait()).To(Exit(0))
			externalPort1 = MapTCPRoute(appName, domainName)
			Expect(cf.Cf("start", appName).Wait(Config.CfPushTimeoutDuration())).To(Exit(0))
		})

		AfterEach(func() {
			app_helpers.AppReport(appName)
			Eventually(cf.Cf("delete", appName, "-f", "-r")).Should(Exit(0))
		})

		It("maps a single external port to an application's container port", func() {
			resp, err := SendAndReceive(domainName, externalPort1)
			Expect(err).ToNot(HaveOccurred())
			Expect(resp).To(ContainSubstring(serverId1))
		})

		Context("with two different apps", func() {
			var (
				secondAppName string
				serverId2     = "server2"
			)

			BeforeEach(func() {
				secondAppName = random_name.CATSRandomName("APP")
				cmd := fmt.Sprintf("tcp-listener --serverId=%s", serverId2)

				Expect(cf.Cf("push",
					"--no-route",
					"--no-start",
					secondAppName,
					"-p", compiledApp,
					"-s", Config.GetWindowsStack(),
					"-b", Config.GetBinaryBuildpackName(),
					"-m", DEFAULT_MEMORY_LIMIT,
					"-f", filepath.Join(tcpDropletReceiver, "manifest.yml"),
					"-c", cmd,
				).Wait()).To(Exit(0))

				Expect(cf.Cf("map-route",
					secondAppName, domainName, "--port", externalPort1,
				).Wait()).To(Exit(0))
				Expect(cf.Cf("start", secondAppName).Wait(Config.CfPushTimeoutDuration())).To(Exit(0))
			})

			AfterEach(func() {
				app_helpers.AppReport(secondAppName)
				Eventually(cf.Cf("delete-route", domainName, "--port", externalPort1, "-f")).Should(Exit(0))
				Eventually(cf.Cf("delete", appName, "-f", "-r")).Should(Exit(0))
				Eventually(cf.Cf("delete", secondAppName, "-f", "-r")).Should(Exit(0))
			})

			It("maps single external port to both applications", func() {
				serverResponses, err := GetNServerResponses(10, domainName, externalPort1)
				Expect(err).ToNot(HaveOccurred())
				Expect(serverResponses).To(ContainElement(ContainSubstring(serverId1)))
				Expect(serverResponses).To(ContainElement(ContainSubstring(serverId2)))
			})
		})

		Context("with a second external port", func() {
			var externalPort2 string

			BeforeEach(func() {
				externalPort2 = MapTCPRoute(appName, domainName)
			})

			It("maps both ports to the same application", func() {
				resp1, err := SendAndReceive(domainName, externalPort1)
				Expect(err).ToNot(HaveOccurred())
				Expect(resp1).To(ContainSubstring(serverId1))

				resp2, err := SendAndReceive(domainName, externalPort2)
				Expect(err).ToNot(HaveOccurred())
				Expect(resp2).To(ContainSubstring(serverId1))
			})
		})
	})
})
