package security_groups_test

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"

	. "github.com/cloudfoundry/cf-acceptance-tests/cats_suite_helpers"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gbytes"
	. "github.com/onsi/gomega/gexec"

	"github.com/cloudfoundry/cf-acceptance-tests/helpers/app_helpers"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/assets"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/random_name"
	"github.com/cloudfoundry/cf-test-helpers/v2/cf"
	"github.com/cloudfoundry/cf-test-helpers/v2/workflowhelpers"

	"github.com/cloudfoundry/cf-acceptance-tests/helpers/skip_messages"

)

var _ = IPv6SecurityGroupsDescribe("Security Group Tests", func() {
	var (
		orgName           string
		spaceName         string
		securityGroupName string
		serverAppName     string
		privateHost       string
		privatePort       int
		clientAppName     string
	)

	BeforeEach(func() {
		orgName = TestSetup.RegularUserContext().Org
		spaceName = TestSetup.RegularUserContext().Space
	})

	Describe("IPv6 Security Group for Internal Cloud Controller Access", func() {
		var appName string

		BeforeEach(func() {
			appName = random_name.CATSRandomName("APP")

			By("pushing a proxy app")
			Expect(cf.Cf(
				"push", appName,
				"-b", Config.GetGoBuildpackName(),
				"-p", assets.NewAssets().ProxyIpv6,
				"-f", assets.NewAssets().ProxyIpv6+"/manifest.yml",
			).Wait(Config.CfPushTimeoutDuration())).To(Exit(0))
		})

		AfterEach(func() {
			Expect(cf.Cf("delete", appName, "-f", "-r").Wait(Config.CfPushTimeoutDuration())).To(Exit(0))
			deleteSecurityGroup(securityGroupName)
		})

		It("manages whether apps can reach certain IPv6 addresses per ASG configuration", func() {
			proxyRequestURL := fmt.Sprintf("%s%s.%s/https_proxy/cloud-controller-ng.service.cf.internal:9024/", Config.Protocol(), appName, Config.GetAppsDomain())

			client := &http.Client{
				Transport: &http.Transport{
					TLSClientConfig: &tls.Config{
						InsecureSkipVerify: Config.GetSkipSSLValidation(),
					},
				},
			}

			By("checking that our app can't initially reach cloud controller over internal address")
			assertAppCannotConnect(client, proxyRequestURL)

			By("binding a new IPv6 security group")
			securityGroupName = bindCCSecurityGroupIpv6(orgName, spaceName)

			if Config.GetDynamicASGsEnabled() {
				By("checking that our app can eventually reach cloud controller over internal address")
				assertEventuallyAppCanConnect(client, proxyRequestURL)
			} else {
				By("because dynamic ASGs are not enabled, validating an app restart is required")
				assertAppRestartRequiredForConnect(client, proxyRequestURL, appName)
			}

			By("unbinding the security group")
			unbindSecurityGroup(securityGroupName, orgName, spaceName)

			if Config.GetDynamicASGsEnabled() {
				By("checking that our app eventually cannot reach cloud controller over internal address")
				assertEventuallyAppCannotConnect(client, proxyRequestURL)
			} else {
				By("because dynamic ASGs are not enabled, validating an app restart is required")
				assertAppRestartRequiredForDisconnect(client, proxyRequestURL, appName)
			}
		})
	})

	Describe("Using container-networking and running security-groups with IPv6", func() {
		BeforeEach(func() {
			if !Config.GetIncludeContainerNetworking() {
				Skip(skip_messages.SkipContainerNetworkingMessage)
			}

			serverAppName, privateHost, privatePort = pushServerApp()
			clientAppName = pushClientApp()
			assertIPv6NetworkingPreconditions(clientAppName, privateHost, privatePort)
		})

		AfterEach(func() {
			app_helpers.AppReport(serverAppName)
			Expect(cf.Cf("delete", serverAppName, "-f", "-r").Wait(Config.CfPushTimeoutDuration())).To(Exit(0))

			app_helpers.AppReport(clientAppName)
			Expect(cf.Cf("delete", clientAppName, "-f", "-r").Wait(Config.CfPushTimeoutDuration())).To(Exit(0))

			deleteSecurityGroup(securityGroupName)
		})

		It("correctly configures ASGs and C2C policy independent of each other in terms of IPv6 env", func() {
			By("creating a wide-open ASG")
			dest := Destination{
				IP:       "4000::/3", // Some random IP that isn't covered by an existing Security Group rule
				Protocol: "all",
			}
			securityGroupName = createSecurityGroup(dest)
			privateAddress := Config.GetUnallocatedIPForSecurityGroup()

			By("binding new security group")
			bindSecurityGroup(securityGroupName, orgName, spaceName)

			Expect(cf.Cf("restart", clientAppName).Wait(Config.CfPushTimeoutDuration())).To(Exit(0))

			By("Testing that client app cannot connect to the server app using the overlay")
			containerIp, containerPort := getAppContainerIpAndPort(serverAppName)
			catnipCurlResponse := testAppConnectivity(clientAppName, containerIp, containerPort)
			Expect(catnipCurlResponse.ReturnCode).NotTo(Equal(0), "No policy configured but client app can talk to server app using overlay")

			By("Testing that external connectivity to a private IP is not refused (but may be unreachable for other reasons)")
			Eventually(func() string {
				resp := testAppConnectivity(clientAppName, privateAddress, 80)
				return resp.Stderr
			}, 3*time.Minute).Should(MatchRegexp("Connection timed out after|No route to host"), "Wide-open ASG configured but app is still refused by private IP")

			By("adding policy")
			workflowhelpers.AsUser(TestSetup.AdminUserContext(), Config.DefaultTimeoutDuration(), func() {
				Expect(cf.Cf("target", "-o", orgName, "-s", spaceName).Wait()).To(Exit(0))
				Expect(string(cf.Cf("network-policies").Wait().Out.Contents())).ToNot(ContainSubstring(serverAppName))
				Expect(cf.Cf("add-network-policy", clientAppName, serverAppName, "--port", fmt.Sprintf("%d", containerPort), "--protocol", "tcp").Wait(Config.CfPushTimeoutDuration())).To(Exit(0))
				Expect(string(cf.Cf("netwozrk-policies").Wait().Out.Contents())).To(ContainSubstring(serverAppName))
			})

			By("Testing that client app can connect to server app using the overlay")
			Eventually(func() int {
				catnipCurlResponse = testAppConnectivity(clientAppName, containerIp, containerPort)
				return catnipCurlResponse.ReturnCode
			}, "5s").Should(Equal(0), "Policy is configured + wide-open ASG but client app cannot talk to server app using overlay")

			By("unbinding the wide-open security group")
			unbindSecurityGroup(securityGroupName, orgName, spaceName)
			Expect(cf.Cf("restart", clientAppName).Wait(Config.CfPushTimeoutDuration())).To(Exit(0))

			By("Testing that client app can still connect to server app using the overlay")
			Eventually(func() int {
				catnipCurlResponse = testAppConnectivity(clientAppName, containerIp, containerPort)
				return catnipCurlResponse.ReturnCode
			}, 3*time.Minute).Should(Equal(0), "Policy is configured, ASGs are not but client app cannot talk to server app using overlay")

			By("Testing that external connectivity to a private IP is refused")
			Eventually(func() string {
				resp := testAppConnectivity(clientAppName, privateAddress, 80)
				return resp.Stderr
			}, 3*time.Minute).Should(MatchRegexp("refused|No route to host| Connection timed out"))

			By("deleting policy")
			workflowhelpers.AsUser(TestSetup.AdminUserContext(), Config.DefaultTimeoutDuration(), func() {
				Expect(cf.Cf("target", "-o", orgName, "-s", spaceName).Wait()).To(Exit(0))
				Expect(string(cf.Cf("network-policies").Wait().Out.Contents())).To(ContainSubstring(serverAppName))
				Expect(cf.Cf("remove-network-policy", clientAppName, serverAppName, "--port", fmt.Sprintf("%d", containerPort), "--protocol", "tcp").Wait(Config.CfPushTimeoutDuration())).To(Exit(0))
				Expect(string(cf.Cf("network-policies").Wait().Out.Contents())).ToNot(ContainSubstring(serverAppName))
			})

			By("Testing the client app cannot connect to the server app using the overlay")
			Eventually(func() int {
				catnipCurlResponse = testAppConnectivity(clientAppName, containerIp, containerPort)
				return catnipCurlResponse.ReturnCode
			}, 3*time.Minute).ShouldNot(Equal(0), "No policy is configured but client app can talk to server app using overlay")
		})
	})

	Describe("Using staging security groups", func() {
		var testAppName, buildpack string

		BeforeEach(func() {
			serverAppName, privateHost, privatePort = pushServerApp()

			By("Asserting default staging security group configuration")
			testAppName = random_name.CATSRandomName("APP")
			buildpack = createDummyBuildpack()
			pushApp(testAppName, buildpack)

			privateUri := fmt.Sprintf("%s:%d", privateHost, privatePort)
			Expect(cf.Cf("set-env", testAppName, "TESTURI", privateUri).Wait()).To(Exit(0))

			Expect(cf.Cf("start", testAppName).Wait(Config.CfPushTimeoutDuration())).To(Exit(1))
			Eventually(getStagingOutput(testAppName), 5).Should(Say("CURL_EXIT=[^0]"), "Expected staging security groups not to allow internal communication between app containers. Configure your staging security groups to not allow traffic on internal networks, or disable this test by setting 'include_security_groups' to 'false' in '"+os.Getenv("CONFIG")+"'.")
		})

		AfterEach(func() {
			app_helpers.AppReport(serverAppName)
			Expect(cf.Cf("delete", serverAppName, "-f", "-r").Wait(Config.CfPushTimeoutDuration())).To(Exit(0))

			app_helpers.AppReport(testAppName)
			Expect(cf.Cf("delete", testAppName, "-f", "-r").Wait(Config.CfPushTimeoutDuration())).To(Exit(0))

			deleteBuildpack(buildpack)
		})

		It("allows external and denies internal traffic during staging based on default IPv6 staging security rules", func() {
			Expect(cf.Cf("set-env", testAppName, "TESTURI", "www.ipv6.google.com").Wait()).To(Exit(0))
			Expect(cf.Cf("start", testAppName).Wait(Config.CfPushTimeoutDuration())).To(Exit(1))
			Eventually(getStagingOutput(testAppName), 5).Should(Say("CURL_EXIT=0"))
		})
	})
})

func bindCCSecurityGroupIpv6(orgName, spaceName string) string {
	destinations := []Destination{{
		IP:       "2001:0db8::/32", // Adjust as necessary for the environment
		Ports:    "9024",           // Adjust as needed for security context
		Protocol: "tcp",
	}}
	securityGroupName := createSecurityGroupIPv6(destinations...)
	bindSecurityGroup(securityGroupName, orgName, spaceName)

	return securityGroupName
}

func createSecurityGroupIPv6(allowedDestinations ...Destination) string {
	file, err := os.CreateTemp(os.TempDir(), "CATS-sg-rules")
	Expect(err).NotTo(HaveOccurred())
	defer os.Remove(file.Name())

	Expect(json.NewEncoder(file).Encode(allowedDestinations)).To(Succeed())

	rulesPath := file.Name()
	securityGroupName := random_name.CATSRandomName("SG")

	workflowhelpers.AsUser(TestSetup.AdminUserContext(), Config.DefaultTimeoutDuration(), func() {
		Expect(cf.Cf("create-security-group", securityGroupName, rulesPath).Wait()).To(Exit(0))
	})

	return securityGroupName
}

func assertIPv6NetworkingPreconditions(clientAppName string, privateHost string, privatePort int) {
	By("Asserting default running security group configuration for traffic between containers")
	catnipCurlResponse := testAppConnectivity(clientAppName, privateHost, privatePort)
	Expect(catnipCurlResponse.ReturnCode).NotTo(Equal(0), "Expected default running security groups not to allow internal communication between app containers. Configure your running security groups to not allow traffic on internal networks, or disable this test by setting 'include_security_groups' to 'false' in '"+os.Getenv("CONFIG")+"'.")

	By("Asserting default running security group configuration from a running container to an external destination")
	catnipCurlResponse = testAppConnectivity(clientAppName, "www.ipv6.google.com", 80)
	Expect(catnipCurlResponse.ReturnCode).To(Equal(0), "Expected default running security groups to allow external traffic from app containers. Configure your running security groups to not allow traffic on internal networks, or disable this test by setting 'include_security_groups' to 'false' in '"+os.Getenv("CONFIG")+"'.")
}