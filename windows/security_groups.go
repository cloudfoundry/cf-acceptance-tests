package windows

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"os"

	"github.com/cloudfoundry-incubator/cf-test-helpers/cf"
	"github.com/cloudfoundry-incubator/cf-test-helpers/workflowhelpers"
	. "github.com/cloudfoundry/cf-acceptance-tests/cats_suite_helpers"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/app_helpers"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/assets"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/random_name"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gexec"

	"github.com/cloudfoundry-incubator/cf-test-helpers/generator"
	"github.com/cloudfoundry-incubator/cf-test-helpers/helpers"
)

var _ = WindowsDescribe("Security Groups", func() {
	var appName string

	BeforeEach(func() {
		appName = random_name.CATSRandomName("APP")

		Expect(cf.Cf("push",
			appName,
			"--no-start",
			"-s", Config.GetWindowsStack(),
			"-b", Config.GetHwcBuildpackName(),
			"-m", DEFAULT_MEMORY_LIMIT,
			"-p", assets.NewAssets().Nora,
			"-d", Config.GetAppsDomain()).Wait(Config.DefaultTimeoutDuration())).To(Exit(0))
		app_helpers.SetBackend(appName)
		Expect(cf.Cf("start", appName).Wait(Config.CfPushTimeoutDuration())).To(Exit(0))
		Eventually(helpers.CurlingAppRoot(Config, appName)).Should(ContainSubstring("hello i am nora"))
	})

	AfterEach(func() {
		app_helpers.AppReport(appName, Config.DefaultTimeoutDuration())

		Expect(cf.Cf("delete", appName, "-f", "-r").Wait(Config.DefaultTimeoutDuration())).Should(Exit(0))
	})

	Context("when a tcp (or udp) rule is applied", func() {
		var (
			securityGroupName string
			secureHost        string
			securePort        string
		)

		BeforeEach(func() {
			By("Asserting default running security group Configuration for traffic to private ip addresses")
			var err error
			secureAddress := Config.GetWindowsSecureAddress()
			secureHost, securePort, err = net.SplitHostPort(secureAddress)
			Expect(err).NotTo(HaveOccurred())
			Expect(noraTCPConnectResponse(appName, secureHost, securePort)).Should(Equal(1))

			By("Asserting default running security group Configuration from a running container to a public ip")
			Expect(noraTCPConnectResponse(appName, "8.8.8.8", "53")).Should(Equal(0))
		})

		AfterEach(func() {
			deleteSecurityGroup(securityGroupName)
		})

		It("allows traffic to a private ip after applying a policy and blocks it when the policy is removed", func() {
			rule := Destination{IP: secureHost, Port: securePort, Protocol: "tcp"}
			securityGroupName = createSecurityGroup(rule)
			bindSecurityGroup(securityGroupName, TestSetup.RegularUserContext().Org, TestSetup.RegularUserContext().Space)

			Expect(cf.Cf("restart", appName).Wait(Config.CfPushTimeoutDuration())).To(Exit(0))
			Eventually(helpers.CurlingAppRoot(Config, appName)).Should(ContainSubstring("hello i am nora"))

			Expect(noraTCPConnectResponse(appName, secureHost, securePort)).Should(Equal(0))

			unbindSecurityGroup(securityGroupName, TestSetup.RegularUserContext().Org, TestSetup.RegularUserContext().Space)

			Expect(cf.Cf("restart", appName).Wait(Config.CfPushTimeoutDuration())).To(Exit(0))
			Eventually(helpers.CurlingAppRoot(Config, appName)).Should(ContainSubstring("hello i am nora"))

			Expect(noraTCPConnectResponse(appName, secureHost, securePort)).Should(Equal(1))
		})
	})
})

type Destination struct {
	IP       string `json:"destination"`
	Port     string `json:"ports,omitempty"`
	Protocol string `json:"protocol"`
	Code     int    `json:"code,omitempty"`
	Type     int    `json:"type,omitempty"`
}

func createSecurityGroup(allowedDestinations ...Destination) string {
	file, _ := ioutil.TempFile(os.TempDir(), "WATS-sg-rules")
	defer os.Remove(file.Name())
	Expect(json.NewEncoder(file).Encode(allowedDestinations)).To(Succeed())

	rulesPath := file.Name()
	securityGroupName := fmt.Sprintf("WATS-SG-%s", generator.PrefixedRandomName(Config.GetNamePrefix(), "SECURITY-GROUP"))

	workflowhelpers.AsUser(TestSetup.AdminUserContext(), Config.DefaultTimeoutDuration(), func() {
		Expect(cf.Cf("create-security-group", securityGroupName, rulesPath).Wait(Config.DefaultTimeoutDuration())).To(Exit(0))
	})

	return securityGroupName
}

func bindSecurityGroup(securityGroupName, orgName, spaceName string) {
	By("Applying security group")
	workflowhelpers.AsUser(TestSetup.AdminUserContext(), Config.DefaultTimeoutDuration(), func() {
		Expect(cf.Cf("bind-security-group", securityGroupName, orgName, spaceName).Wait(Config.DefaultTimeoutDuration())).To(Exit(0))
	})
}

func unbindSecurityGroup(securityGroupName, orgName, spaceName string) {
	By("Unapplying security group")
	workflowhelpers.AsUser(TestSetup.AdminUserContext(), Config.DefaultTimeoutDuration(), func() {
		Expect(cf.Cf("unbind-security-group", securityGroupName, orgName, spaceName).Wait(Config.DefaultTimeoutDuration())).To(Exit(0))
	})
}

func deleteSecurityGroup(securityGroupName string) {
	workflowhelpers.AsUser(TestSetup.AdminUserContext(), Config.DefaultTimeoutDuration(), func() {
		Expect(cf.Cf("delete-security-group", securityGroupName, "-f").Wait(Config.DefaultTimeoutDuration())).To(Exit(0))
	})
}

type NoraTCPConnectResponse struct {
	Stdout     string
	Stderr     string
	ReturnCode int `json:"return_code"`
}

func noraTCPConnectResponse(appName, host, port string) int {
	var noraTCPConnectResponse NoraTCPConnectResponse
	resp := helpers.CurlApp(Config, appName, fmt.Sprintf("/connect/%s/%s", host, port))
	Expect(json.Unmarshal([]byte(resp), &noraTCPConnectResponse)).To(Succeed())
	return noraTCPConnectResponse.ReturnCode
}
