package tcp_routing

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gexec"
	. "github.com/cloudfoundry/cf-acceptance-tests/cats_suite_helpers"
	"time"
	"fmt"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/assets"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/random_name"
	"github.com/cloudfoundry-incubator/cf-test-helpers/workflowhelpers"
	"github.com/cloudfoundry-incubator/cf-test-helpers/cf"
	"regexp"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/app_helpers"
	"net"
)

const DefaultRouterGroupName = "default-tcp"

var _ = TCPRoutingDescribe("TCP Routing", func() {
	var (
		domainName string
	)

	BeforeEach(func() {
		//helpers.UpdateOrgQuota(adminContext)
		//workflowhelpers.AsUser(TestSetup.AdminUserContext(), Config.DefaultTimeoutDuration(), func() {
		//
		//})
		domainName = fmt.Sprintf("%s.%s", random_name.CATSRandomName("TCP-DOMAIN"), Config.GetAppsDomain())

		workflowhelpers.AsUser(TestSetup.AdminUserContext(), Config.DefaultTimeoutDuration(), func() {
			routerGroupOutput := string(cf.Cf("router-groups").Wait(Config.DefaultTimeoutDuration()).Out.Contents())
			Expect(routerGroupOutput).To(MatchRegexp(fmt.Sprintf("%s\\s+tcp", DefaultRouterGroupName)), fmt.Sprintf("Router group %s of type tcp doesn't exist", DefaultRouterGroupName))

			Expect(cf.Cf("create-shared-domain", domainName, "--router-group", DefaultRouterGroupName).Wait(Config.DefaultTimeoutDuration())).To(Exit(0))
		})
	})

	Context("single app port", func() {
		var (
			appName            string
			tcpDropletReceiver = assets.NewAssets().TCPDropletReceiver
			serverId1          string
			externalPort1      string
			spaceName          string
		)

		BeforeEach(func() {
			appName = random_name.CATSRandomName("APP")
			serverId1 = "server1"
			cmd := fmt.Sprintf("tcp-droplet-receiver --serverId=%s", serverId1)
			spaceName = TestSetup.TestSpace.SpaceName()

			// Uses --no-route flag so there is no HTTP route
			Expect(cf.Cf("push",
				"--no-route",
				"--no-start",
				appName,
				"-p", tcpDropletReceiver,
				"-b", Config.GetGoBuildpackName(),
				"-m", DEFAULT_MEMORY_LIMIT,
				"-c", cmd,
			).Wait(Config.DefaultTimeoutDuration())).To(Exit(0))
			//routing_helpers.UpdatePorts(appName, []uint16{3333}, DEFAULT_TIMEOUT)
			externalPort1 = mapTCPRoute(appName, domainName)
			Expect(cf.Cf("start", appName).Wait(Config.CfPushTimeoutDuration())).To(Exit(0))
		})

		AfterEach(func() {
			app_helpers.AppReport(appName, Config.DefaultTimeoutDuration())
			Eventually(cf.Cf("delete", appName, "-f", "-r"), Config.DefaultTimeoutDuration()).Should(Exit(0))
		})

		It("maps a single external port to an application's container port", func() {
				Eventually(func() error {
					_, err := sendAndReceive(domainName, externalPort1)
					return err
				}).ShouldNot(HaveOccurred())

				resp, err := sendAndReceive(domainName, externalPort1)
				Expect(err).ToNot(HaveOccurred())
				Expect(resp).To(ContainSubstring(serverId1))
		})
	})
})

func mapTCPRoute(appName, domainName string) string {
	createRouteSession := cf.Cf("map-route", appName, domainName, "--random-port").Wait(Config.DefaultTimeoutDuration())
	Expect(createRouteSession).To(Exit(0))

	r := regexp.MustCompile(fmt.Sprintf(`.+%s:(\d+).+`, domainName))
	return r.FindStringSubmatch(string(createRouteSession.Out.Contents()))[1]
}

func sendAndReceive(addr string, externalPort string) (string, error) {
	address := fmt.Sprintf("%s:%s", addr, externalPort)

	conn, err := net.Dial("tcp", address)
	defer conn.Close()
	if err != nil {
		return "", err
	}

	message := []byte(fmt.Sprintf("Time is %d", time.Now().Nanosecond()))

	_, err = conn.Write(message)
	if err != nil {
		if ne, ok := err.(*net.OpError); ok {
			if ne.Temporary() {
				return sendAndReceive(addr, externalPort)
			}
		}

		return "", err
	}

	buff := make([]byte, 1024)
	_, err = conn.Read(buff)
	if err != nil {
		if ne, ok := err.(*net.OpError); ok {
			if ne.Temporary() {
				return sendAndReceive(addr, externalPort)
			}
		}

		return "", err
	}

	return string(buff), nil
}