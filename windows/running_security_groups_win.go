package windows

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"

	. "github.com/cloudfoundry/cf-acceptance-tests/cats_suite_helpers"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gexec"

	"github.com/cloudfoundry-incubator/cf-test-helpers/cf"
	"github.com/cloudfoundry-incubator/cf-test-helpers/helpers"
	"github.com/cloudfoundry-incubator/cf-test-helpers/workflowhelpers"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/app_helpers"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/assets"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/random_name"
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

type NoraCurlResponse struct {
	Stdout     string
	Stderr     string
	ReturnCode int `json:"return_code"`
}

func pushApp(appName, buildpack string) {
	Expect(cf.Cf("push",
		appName,
		"-s", Config.GetWindowsStack(),
		"-b", buildpack,
		"-m", DEFAULT_WINDOWS_MEMORY_LIMIT,
		"-p", assets.NewAssets().Nora,
		"-d", Config.GetAppsDomain()).Wait(Config.CfPushTimeoutDuration())).To(Exit(0))

	warmUpRequest(appName)
}

// When a .NET app running via HWC buildpack receives its first HTTP request,
// it has to do just-in-time compilation, which can take ~10 seconds.
func warmUpRequest(appName string) {
	Expect(helpers.CurlAppRoot(Config, appName)).To(ContainSubstring("hello i am nora running on"))
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

func testAppConnectivity(clientAppName string, privateHost string, privatePort int) NoraCurlResponse {
	var noraCurlResponse NoraCurlResponse
	uri := helpers.AppUri(clientAppName, fmt.Sprintf("/curl/%s/%d", privateHost, privatePort), Config)
	curlResponse := helpers.CurlSkipSSL(Config.GetSkipSSLValidation(), uri).Wait()
	json.Unmarshal([]byte(curlResponse.Out.Contents()), &noraCurlResponse)
	return noraCurlResponse
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

func pushServerApp() (serverAppName string, privateHost string, privatePort int) {
	serverAppName = random_name.CATSRandomName("APP")
	pushApp(serverAppName, Config.GetHwcBuildpackName())

	privateHost, privatePort = getAppHostIpAndPort(serverAppName)
	return
}

func pushClientApp() (clientAppName string) {
	clientAppName = random_name.CATSRandomName("APP")
	pushApp(clientAppName, Config.GetHwcBuildpackName())
	return
}

func restartClientApp(clientAppName string) {
	Expect(cf.Cf("restart", clientAppName).Wait(Config.CfPushTimeoutDuration())).To(Exit(0))

	warmUpRequest(clientAppName)
}

func assertNetworkingPreconditions(clientAppName string, privateHost string, privatePort int) {
	By("Asserting default running security group configuration for traffic between containers")
	noraCurlResponse := testAppConnectivity(clientAppName, privateHost, privatePort)
	Expect(noraCurlResponse.ReturnCode).NotTo(Equal(0), "Expected default running security groups not to allow internal communication between app containers. Configure your running security groups to not allow traffic on internal networks, or disable this test by setting 'include_security_groups' to 'false' in '"+os.Getenv("CONFIG")+"'.")
}

var _ = WindowsDescribe("WINDOWS: App Instance Networking", func() {
	SecurityGroupsDescribe("WINDOWS: Using container-networking and running security-groups", func() {
		var serverAppName, clientAppName, privateHost, orgName, spaceName, securityGroupName string
		var privatePort int

		BeforeEach(func() {
			orgName = TestSetup.RegularUserContext().Org
			spaceName = TestSetup.RegularUserContext().Space
			serverAppName, privateHost, privatePort = pushServerApp()
			clientAppName = pushClientApp()

			if Config.GetWindowsStack() == "windows" {
				assertNetworkingPreconditions(clientAppName, privateHost, privatePort)
			}
		})

		AfterEach(func() {
			app_helpers.AppReport(serverAppName)
			Expect(cf.Cf("delete", serverAppName, "-f", "-r").Wait(Config.CfPushTimeoutDuration())).To(Exit(0))

			app_helpers.AppReport(clientAppName)
			Expect(cf.Cf("delete", clientAppName, "-f", "-r").Wait(Config.CfPushTimeoutDuration())).To(Exit(0))

			deleteSecurityGroup(securityGroupName)
		})

		It("WINDOWS: correctly configures asgs", func() {
			By("creating a wide-open ASG")
			dest := Destination{
				IP:       "0.0.0.0/0", // some random IP that isn't covered by an existing Security Group rule
				Protocol: "all",
			}
			securityGroupName = createSecurityGroup(dest)
			privateAddress := Config.GetUnallocatedIPForSecurityGroup()

			By("binding new security group")
			bindSecurityGroup(securityGroupName, orgName, spaceName)

			restartClientApp(clientAppName)

			By("Testing that external connectivity to a private ip is not refused (but may be unreachable for other reasons)")

			noraCurlResponse := testAppConnectivity(clientAppName, privateAddress, 80)
			Expect(noraCurlResponse.Stderr).To(ContainSubstring("The operation has timed out"), "wide-open ASG configured but app is still refused by private ip")

			By("unbinding the wide-open security group")
			unbindSecurityGroup(securityGroupName, orgName, spaceName)
			restartClientApp(clientAppName)

			By("restarting the app")
			restartClientApp(clientAppName)

			By("Testing that external connectivity to a private ip is refused")
			Eventually(func() string {
				return testAppConnectivity(clientAppName, privateAddress, 80).Stderr
			}).Should(ContainSubstring("Unable to connect to the remote server"))
		})

		It("allows traffic to the public internet by default", func() {
			if !Config.GetIncludeInternetDependent() {
				Skip("skipping internet dependent test as 'include_internet_dependent' is not set")
			}

			By("Asserting default running security group configuration from a running container to an external destination")
			noraCurlResponse := testAppConnectivity(clientAppName, "www.google.com", 80)

			Expect(noraCurlResponse.ReturnCode).To(Equal(0), "Expected default running security groups to allow external traffic from app containers. Configure your running security groups to not allow traffic on internal networks, or disable this test by setting 'include_security_groups' to 'false' in '"+os.Getenv("CONFIG")+"'.")
		})
	})
})
