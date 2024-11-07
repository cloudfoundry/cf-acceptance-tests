package security_groups_test

import (
	"crypto/tls"
	"fmt"
	"net/http"

	. "github.com/cloudfoundry/cf-acceptance-tests/cats_suite_helpers"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gexec"

	"github.com/cloudfoundry/cf-acceptance-tests/helpers/assets"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/random_name"
	"github.com/cloudfoundry/cf-test-helpers/v2/cf"
)

var _ = CommaDelimitedSecurityGroupsDescribe("Comma Delimited ASGs", func() {
	var (
		orgName           string
		spaceName         string
		appName           string
		securityGroupName string
	)

	BeforeEach(func() {
		orgName = TestSetup.RegularUserContext().Org
		spaceName = TestSetup.RegularUserContext().Space
		appName = random_name.CATSRandomName("APP")

		By("pushing a proxy app")
		Expect(cf.Cf(
			"push", appName,
			"-b", Config.GetGoBuildpackName(),
			"-p", assets.NewAssets().Proxy,
			"-f", assets.NewAssets().Proxy+"/manifest.yml",
		).Wait(Config.CfPushTimeoutDuration())).To(Exit(0))
	})

	AfterEach(func() {
		Expect(cf.Cf("delete", appName, "-f", "-r").Wait(Config.CfPushTimeoutDuration())).To(Exit(0))
		deleteSecurityGroup(securityGroupName)
	})

	It("manages whether apps can reach certain IP addresses per ASG configuration with commas", func() {
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

		By("binding a new security group with commas")
		securityGroupName = bindCCSecurityGroupWithCommas(orgName, spaceName)

		if Config.GetDynamicASGsEnabled() {
			By("checking that our app can eventually reach cloud controller over internal address")
			assertEventuallyAppCanConnect(client, proxyRequestURL)
		} else {
			By("because dynamic asgs are not enabled, validating an app restart is required")
			assertAppRestartRequiredForConnect(client, proxyRequestURL, appName)
		}

		By("unbinding the security group")
		unbindSecurityGroup(securityGroupName, orgName, spaceName)

		if Config.GetDynamicASGsEnabled() {
			By("checking that our app eventually cannot reach cloud controller over internal address")
			assertEventuallyAppCannotConnect(client, proxyRequestURL)
		} else {
			By("because dynamic asgs are not enabled, validating an app restart is required")
			assertAppRestartRequiredForDisconnect(client, proxyRequestURL, appName)
		}
	})
})

func bindCCSecurityGroupWithCommas(orgName, spaceName string) string {
	destinations := []Destination{{
		IP:       "10.0.0.0/8,192.168.0.0/16,172.16.0.0/12",
		Ports:    "9024", // internal cc port
		Protocol: "tcp",
	}}
	securityGroupName := createSecurityGroup(destinations...)
	bindSecurityGroup(securityGroupName, orgName, spaceName)

	return securityGroupName
}
