package security_groups_test

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	. "github.com/cloudfoundry/cf-acceptance-tests/cats_suite_helpers"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gbytes"
	. "github.com/onsi/gomega/gexec"

	"github.com/cloudfoundry-incubator/cf-test-helpers/cf"
	"github.com/cloudfoundry-incubator/cf-test-helpers/helpers"
	"github.com/cloudfoundry-incubator/cf-test-helpers/workflowhelpers"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/app_helpers"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/assets"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/logs"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/random_name"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/skip_messages"
)

type AppsResponse struct {
	Resources []struct {
		Metadata struct {
			Url string
		}
	}
}

type StatsResponse map[string]struct {
	Stats struct {
		Host string
		Port int
	}
}

type CatnipCurlResponse struct {
	Stdout     string
	Stderr     string
	ReturnCode int `json:"return_code"`
}

func pushApp(appName, buildpack string) {
	Expect(cf.Cf("push",
		appName,
		"--no-start",
		"-b", buildpack,
		"-m", DEFAULT_MEMORY_LIMIT,
		"-p", assets.NewAssets().Catnip,
		"-c", "./catnip",
		"-d", Config.GetAppsDomain()).Wait()).To(Exit(0))
}

func getAppHostIpAndPort(appName string) (string, int) {
	var appsResponse AppsResponse
	cfResponse := cf.Cf("curl", fmt.Sprintf("/v2/apps?q=name:%s", appName)).Wait().Out.Contents()
	json.Unmarshal(cfResponse, &appsResponse)
	serverAppUrl := appsResponse.Resources[0].Metadata.Url

	var statsResponse StatsResponse
	cfResponse = cf.Cf("curl", fmt.Sprintf("%s/stats", serverAppUrl)).Wait().Out.Contents()
	json.Unmarshal(cfResponse, &statsResponse)

	return statsResponse["0"].Stats.Host, statsResponse["0"].Stats.Port
}

func testAppConnectivity(clientAppName string, privateHost string, privatePort int) CatnipCurlResponse {
	var catnipCurlResponse CatnipCurlResponse
	curlResponse := helpers.CurlApp(Config, clientAppName, fmt.Sprintf("/curl/%s/%d", privateHost, privatePort))
	json.Unmarshal([]byte(curlResponse), &catnipCurlResponse)
	return catnipCurlResponse
}

func getAppContainerIpAndPort(appName string) (string, int) {
	curlResponse := helpers.CurlApp(Config, appName, "/myip")
	containerIp := strings.TrimSpace(curlResponse)

	curlResponse = helpers.CurlApp(Config, appName, "/env/VCAP_APPLICATION")
	var env map[string]interface{}
	err := json.Unmarshal([]byte(curlResponse), &env)
	Expect(err).NotTo(HaveOccurred())
	containerPort := int(env["port"].(float64))

	return containerIp, containerPort
}

type Destination struct {
	IP       string `json:"destination"`
	Port     int    `json:"ports,string,omitempty"`
	Protocol string `json:"protocol"`
}

func createSecurityGroup(allowedDestinations ...Destination) string {
	file, _ := ioutil.TempFile(os.TempDir(), "CATS-sg-rules")
	defer os.Remove(file.Name())
	Expect(json.NewEncoder(file).Encode(allowedDestinations)).To(Succeed())

	rulesPath := file.Name()
	securityGroupName := random_name.CATSRandomName("SG")

	workflowhelpers.AsUser(TestSetup.AdminUserContext(), Config.DefaultTimeoutDuration(), func() {
		Expect(cf.Cf("create-security-group", securityGroupName, rulesPath).Wait()).To(Exit(0))
	})

	return securityGroupName
}

func bindSecurityGroup(securityGroupName, orgName, spaceName string) {
	By("Applying security group")
	workflowhelpers.AsUser(TestSetup.AdminUserContext(), Config.DefaultTimeoutDuration(), func() {
		Expect(cf.Cf("bind-security-group", securityGroupName, orgName, spaceName).Wait()).To(Exit(0))
	})
}

func unbindSecurityGroup(securityGroupName, orgName, spaceName string) {
	By("Unapplying security group")
	workflowhelpers.AsUser(TestSetup.AdminUserContext(), Config.DefaultTimeoutDuration(), func() {
		Expect(cf.Cf("unbind-security-group", securityGroupName, orgName, spaceName).Wait()).To(Exit(0))
	})
}

func deleteSecurityGroup(securityGroupName string) {
	workflowhelpers.AsUser(TestSetup.AdminUserContext(), Config.DefaultTimeoutDuration(), func() {
		Expect(cf.Cf("delete-security-group", securityGroupName, "-f").Wait()).To(Exit(0))
	})
}

func createDummyBuildpack() string {
	buildpack := random_name.CATSRandomName("BPK")
	buildpackZip := assets.NewAssets().SecurityGroupBuildpack

	workflowhelpers.AsUser(TestSetup.AdminUserContext(), Config.DefaultTimeoutDuration(), func() {
		Expect(cf.Cf("create-buildpack", buildpack, buildpackZip, "999").Wait()).To(Exit(0))
	})
	return buildpack
}

func deleteBuildpack(buildpack string) {
	workflowhelpers.AsUser(TestSetup.AdminUserContext(), Config.DefaultTimeoutDuration(), func() {
		Expect(cf.Cf("delete-buildpack", buildpack, "-f").Wait()).To(Exit(0))
	})
}

func getStagingOutput(appName string) func() *Session {
	return func() *Session {
		appLogsSession := logs.Tail(Config.GetUseLogCache(), appName)
		Expect(appLogsSession.Wait()).To(Exit(0))
		return appLogsSession
	}
}

func pushServerApp() (serverAppName string, privateHost string, privatePort int) {
	serverAppName = random_name.CATSRandomName("APP")
	pushApp(serverAppName, Config.GetBinaryBuildpackName())
	Expect(cf.Cf("start", serverAppName).Wait(Config.CfPushTimeoutDuration())).To(Exit(0))

	privateHost, privatePort = getAppHostIpAndPort(serverAppName)
	return
}

func pushClientApp() (clientAppName string) {
	clientAppName = random_name.CATSRandomName("APP")
	pushApp(clientAppName, Config.GetBinaryBuildpackName())
	Expect(cf.Cf("start", clientAppName).Wait(Config.CfPushTimeoutDuration())).To(Exit(0))
	return
}

func assertNetworkingPreconditions(clientAppName string, privateHost string, privatePort int) {
	By("Asserting default running security group configuration for traffic between containers")
	catnipCurlResponse := testAppConnectivity(clientAppName, privateHost, privatePort)
	Expect(catnipCurlResponse.ReturnCode).NotTo(Equal(0), "Expected default running security groups not to allow internal communication between app containers. Configure your running security groups to not allow traffic on internal networks, or disable this test by setting 'include_security_groups' to 'false' in '"+os.Getenv("CONFIG")+"'.")

	By("Asserting default running security group configuration from a running container to an external destination")
	catnipCurlResponse = testAppConnectivity(clientAppName, "www.google.com", 80)
	Expect(catnipCurlResponse.ReturnCode).To(Equal(0), "Expected default running security groups to allow external traffic from app containers. Configure your running security groups to not allow traffic on internal networks, or disable this test by setting 'include_security_groups' to 'false' in '"+os.Getenv("CONFIG")+"'.")
}

var _ = SecurityGroupsDescribe("App Instance Networking", func() {
	var serverAppName, privateHost string
	var privatePort int

	Describe("Using container-networking and running security-groups", func() {
		var serverAppName, clientAppName, privateHost, orgName, spaceName, securityGroupName string
		var privatePort int

		BeforeEach(func() {
			if !Config.GetIncludeContainerNetworking() {
				Skip(skip_messages.SkipContainerNetworkingMessage)
			}

			orgName = TestSetup.RegularUserContext().Org
			spaceName = TestSetup.RegularUserContext().Space

			serverAppName, privateHost, privatePort = pushServerApp()
			clientAppName = pushClientApp()
			assertNetworkingPreconditions(clientAppName, privateHost, privatePort)
		})

		AfterEach(func() {
			app_helpers.AppReport(serverAppName)
			Expect(cf.Cf("delete", serverAppName, "-f", "-r").Wait(Config.CfPushTimeoutDuration())).To(Exit(0))

			app_helpers.AppReport(clientAppName)
			Expect(cf.Cf("delete", clientAppName, "-f", "-r").Wait(Config.CfPushTimeoutDuration())).To(Exit(0))

			deleteSecurityGroup(securityGroupName)
		})

		It("correctly configures asgs and c2c policy independent of each other", func() {
			By("creating a wide-open ASG")
			dest := Destination{
				IP:       "0.0.0.0/0", // some random IP that isn't covered by an existing Security Group rule
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
			Expect(catnipCurlResponse.ReturnCode).NotTo(Equal(0), "no policy configured but client app can talk to server app using overlay")

			By("Testing that external connectivity to a private ip is not refused (but may be unreachable for other reasons)")
			catnipCurlResponse = testAppConnectivity(clientAppName, privateAddress, 80)
			Expect(catnipCurlResponse.Stderr).To(MatchRegexp("Connection timed out after|No route to host"), "wide-open ASG configured but app is still refused by private ip")

			By("adding policy")
			workflowhelpers.AsUser(TestSetup.AdminUserContext(), Config.DefaultTimeoutDuration(), func() {
				Expect(cf.Cf("target", "-o", orgName, "-s", spaceName).Wait()).To(Exit(0))
				Expect(string(cf.Cf("network-policies").Wait().Out.Contents())).ToNot(ContainSubstring(serverAppName))
				Expect(cf.Cf("add-network-policy", clientAppName, "--destination-app", serverAppName, "--port", fmt.Sprintf("%d", containerPort), "--protocol", "tcp").Wait(Config.CfPushTimeoutDuration())).To(Exit(0))
				Expect(string(cf.Cf("network-policies").Wait().Out.Contents())).To(ContainSubstring(serverAppName))
			})

			By("Testing that client app can connect to server app using the overlay")
			Eventually(func() int {
				catnipCurlResponse = testAppConnectivity(clientAppName, containerIp, containerPort)
				return catnipCurlResponse.ReturnCode
			}, "5s").Should(Equal(0), "policy is configured + wide-open asg but client app cannot talk to server app using overlay")

			By("unbinding the wide-open security group")
			unbindSecurityGroup(securityGroupName, orgName, spaceName)
			Expect(cf.Cf("restart", clientAppName).Wait(Config.CfPushTimeoutDuration())).To(Exit(0))

			By("restarting the app")
			Expect(cf.Cf("restart", clientAppName).Wait(Config.CfPushTimeoutDuration())).To(Exit(0))

			By("Testing that client app can still connect to server app using the overlay")
			Eventually(func() int {
				catnipCurlResponse = testAppConnectivity(clientAppName, containerIp, containerPort)
				return catnipCurlResponse.ReturnCode
			}, "5s").Should(Equal(0), "policy is configured, asgs are not but client app cannot talk to server app using overlay")

			By("Testing that external connectivity to a private ip is refused")
			catnipCurlResponse = testAppConnectivity(clientAppName, privateAddress, 80)
			Expect(catnipCurlResponse.Stderr).To(MatchRegexp("refused|No route to host|Connection timed out"))

			By("deleting policy")
			workflowhelpers.AsUser(TestSetup.AdminUserContext(), Config.DefaultTimeoutDuration(), func() {
				Expect(cf.Cf("target", "-o", orgName, "-s", spaceName).Wait()).To(Exit(0))
				Expect(string(cf.Cf("network-policies").Wait().Out.Contents())).To(ContainSubstring(serverAppName))
				Expect(cf.Cf("remove-network-policy", clientAppName, "--destination-app", serverAppName, "--port", fmt.Sprintf("%d", containerPort), "--protocol", "tcp").Wait(Config.CfPushTimeoutDuration())).To(Exit(0))
				Expect(string(cf.Cf("network-policies").Wait().Out.Contents())).ToNot(ContainSubstring(serverAppName))
			})

			By("Testing the client app cannot connect to the server app using the overlay")
			Eventually(func() int {
				catnipCurlResponse = testAppConnectivity(clientAppName, containerIp, containerPort)
				return catnipCurlResponse.ReturnCode
			}, "5s").ShouldNot(Equal(0), "no policy is configured but client app can talk to server app using overlay")
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

		It("allows external and denies internal traffic during staging based on default staging security rules", func() {
			Expect(cf.Cf("set-env", testAppName, "TESTURI", "www.google.com").Wait()).To(Exit(0))
			Expect(cf.Cf("restart", testAppName).Wait(Config.CfPushTimeoutDuration())).To(Exit(1))
			Eventually(getStagingOutput(testAppName), 5).Should(Say("CURL_EXIT=0"))
		})
	})
})
