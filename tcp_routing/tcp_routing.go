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
	"path/filepath"
)

const DefaultRouterGroupName = "default-tcp"

var _ = TCPRoutingDescribe("TCP Routing", func() {
	var domainName string

	BeforeEach(func() {
		domainName = fmt.Sprintf("tcp.%s", Config.GetAppsDomain())
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
				"-p", tcpDropletReceiver,
				"-b", Config.GetGoBuildpackName(),
				"-m", DEFAULT_MEMORY_LIMIT,
				"-f", filepath.Join(tcpDropletReceiver, "manifest.yml"),
				"-c", cmd,
			).Wait()).To(Exit(0))
			externalPort1 = mapTCPRoute(appName, domainName)
			Expect(cf.Cf("start", appName).Wait(Config.CfPushTimeoutDuration())).To(Exit(0))
		})

		AfterEach(func() {
			app_helpers.AppReport(appName)
			Eventually(cf.Cf("delete", appName, "-f", "-r")).Should(Exit(0))
		})

		It("maps a single external port to an application's container port", func() {
			resp, err := sendAndReceive(domainName, externalPort1)
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
					"-p", tcpDropletReceiver,
					"-b", Config.GetGoBuildpackName(),
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
				Eventually(cf.Cf("delete", appName, "-f", "-r")).Should(Exit(0))
			})

			It("maps single external port to both applications", func() {
				serverResponses, err := getNServerResponses(10, domainName, externalPort1)
				Expect(err).ToNot(HaveOccurred())
				Expect(serverResponses).To(ContainElement(ContainSubstring(serverId1)))
				Expect(serverResponses).To(ContainElement(ContainSubstring(serverId2)))
			})
		})

		Context("with a second external port", func() {
			var externalPort2 string

			BeforeEach(func() {
				externalPort2 = mapTCPRoute(appName, domainName)
			})

			It("maps both ports to the same application", func() {
				resp1, err := sendAndReceive(domainName, externalPort1)
				Expect(err).ToNot(HaveOccurred())
				Expect(resp1).To(ContainSubstring(serverId1))

				resp2, err := sendAndReceive(domainName, externalPort2)
				Expect(err).ToNot(HaveOccurred())
				Expect(resp2).To(ContainSubstring(serverId1))
			})
		})
	})
})

func getNServerResponses(n int, domainName, externalPort1 string) ([]string, error) {
	var responses []string

	for i := 0; i < n; i++ {
		resp, err := sendAndReceive(domainName, externalPort1)
		if err != nil {
			return nil, err
		}

		responses = append(responses, resp)
	}

	return responses, nil
}

func mapTCPRoute(appName, domainName string) string {
	createRouteSession := cf.Cf("map-route", appName, domainName, "--random-port").Wait()
	Expect(createRouteSession).To(Exit(0))

	r := regexp.MustCompile(fmt.Sprintf(`.+%s:(\d+).+`, domainName))
	return r.FindStringSubmatch(string(createRouteSession.Out.Contents()))[1]
}

func sendAndReceive(addr string, externalPort string) (string, error) {
	address := fmt.Sprintf("%s:%s", addr, externalPort)

	conn, err := net.Dial("tcp", address)
	if err != nil {
		return "", err
	}
	defer conn.Close()

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
