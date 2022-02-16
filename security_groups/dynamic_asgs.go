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
)

type AppsResponse struct {
	Resources []struct {
		Links struct {
			Processes struct {
				Href string
			}
		}
	}
}

type StatsResponse struct {
	Resources []struct {
		Host           string
		Instance_ports []struct {
			External int
		}
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
	).Wait()).To(Exit(0))
}

func getAppHostIpAndPort(appName string) (string, int) {
	appGUID := app_helpers.GetAppGuid(appName)

	cfResponse := cf.Cf("curl", fmt.Sprintf("/v3/apps/%s/processes/web/stats", appGUID)).Wait().Out.Contents()

	var statsResponse StatsResponse
	json.Unmarshal(cfResponse, &statsResponse)

	return statsResponse.Resources[0].Host, statsResponse.Resources[0].Instance_ports[0].External
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
		Expect(cf.Cf("bind-security-group", securityGroupName, orgName, "--space", spaceName).Wait()).To(Exit(0))
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
		appLogsSession := logs.Recent(appName)
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

	Describe("Dynamic ASGs", func() {

		var serverAppName, clientAppName, privateHost, orgName, spaceName, securityGroupName string
		var privatePort int

		BeforeEach(func() {
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

		Describe("Dynamic ASGs Enabled", func() {
			BeforeEach(func() {
				if !Config.GetIncludeDynamicASGs() {
					Skip(skip_messages.SkipDynamicASGsEnabledMessage)
				}
			})
		
			It("applies security group without restart", func() {
				By("Testing that external connectivity to a private ip is refused")
				privateAddress := Config.GetUnallocatedIPForSecurityGroup()
				catnipCurlResponse := testAppConnectivity(clientAppName, privateAddress, 80)
				Expect(catnipCurlResponse.Stderr).To(MatchRegexp("refused|No route to host|Connection timed out"))

				By("creating a wide-open ASG")
				dest := Destination{
					IP:       "0.0.0.0/0", // some random IP that isn't covered by an existing Security Group rule
					Protocol: "all",
				}
				securityGroupName = createSecurityGroup(dest)

				By("binding new security group")
				bindSecurityGroup(securityGroupName, orgName, spaceName)

				//	Expect(cf.Cf("restart", clientAppName).Wait(Config.CfPushTimeoutDuration())).To(Exit(0))

				By("Testing that external connectivity to a private ip is not refused (but may be unreachable for other reasons)")
				catnipCurlResponse = testAppConnectivity(clientAppName, privateAddress, 80)
				Expect(catnipCurlResponse.Stderr).ToNot(MatchRegexp("refused"))
				Expect(catnipCurlResponse.Stderr).To(MatchRegexp("Connection timed out after|No route to host"), "wide-open ASG configured but app is still refused by private ip")

				By("unbinding the wide-open security group")
				unbindSecurityGroup(securityGroupName, orgName, spaceName)

				//			Expect(cf.Cf("restart", clientAppName).Wait(Config.CfPushTimeoutDuration())).To(Exit(0))

				By("Testing that external connectivity to a private ip is refused")
				catnipCurlResponse = testAppConnectivity(clientAppName, privateAddress, 80)
				Expect(catnipCurlResponse.Stderr).To(MatchRegexp("refused|No route to host|Connection timed out"))
			})
		})
		Describe("Dynamic ASGs Disabled", func() {
			
			BeforeEach(func() {
				if Config.GetIncludeDynamicASGs() {
					Skip(skip_messages.SkipDynamicASGsDisabledMessage)
				}
			})
			It("does not apply ASGs", func() {
				By("Testing that external connectivity to a private ip is refused")
				privateAddress := Config.GetUnallocatedIPForSecurityGroup()
				catnipCurlResponse := testAppConnectivity(clientAppName, privateAddress, 80)
				Expect(catnipCurlResponse.Stderr).To(MatchRegexp("refused|No route to host|Connection timed out"))

				By("creating a wide-open ASG")
				dest := Destination{
					IP:       "0.0.0.0/0", // some random IP that isn't covered by an existing Security Group rule
					Protocol: "all",
				}
				securityGroupName = createSecurityGroup(dest)

				By("binding new security group")
				bindSecurityGroup(securityGroupName, orgName, spaceName)


				By("Testing that external connectivity to a private ip is still refused (but may be unreachable for other reasons)")
				catnipCurlResponse = testAppConnectivity(clientAppName, privateAddress, 80)
				Expect(catnipCurlResponse.Stderr).To(MatchRegexp("refused|No route to host|Connection timed out"))
		})
	})
})
