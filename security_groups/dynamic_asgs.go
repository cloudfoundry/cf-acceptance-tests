package security_groups_test

import (
	"crypto/tls"
	"fmt"
	"io"
	"net/http"
	"time"

	. "github.com/cloudfoundry/cf-acceptance-tests/cats_suite_helpers"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gexec"

	"github.com/cloudfoundry/cf-acceptance-tests/helpers/assets"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/random_name"
	"github.com/cloudfoundry/cf-test-helpers/v2/cf"
)

var _ = SecurityGroupsDescribe("ASGs", func() {
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

	It("manages whether apps can reach certain IP addresses per ASG configuration", func() {
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

		By("binding a new security group")
		securityGroupName = bindCCSecurityGroup(orgName, spaceName)

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

func assertAppRestartRequiredForConnect(client *http.Client, proxyRequestURL, appName string) {
	assertAppCannotConnect(client, proxyRequestURL)

	By("restarting the app")
	Expect(cf.Cf("restart", appName).Wait(Config.CfPushTimeoutDuration())).To(Exit(0))

	By("checking that our app can now reach cloud controller over internal address")
	assertAppCanConnect(client, proxyRequestURL)
}

func assertAppRestartRequiredForDisconnect(client *http.Client, proxyRequestURL, appName string) {
	assertAppCanConnect(client, proxyRequestURL)

	By("restarting the app")
	Expect(cf.Cf("restart", appName).Wait(Config.CfPushTimeoutDuration())).To(Exit(0))

	By("checking that our app can no longer reach cloud controller over internal address")
	assertAppCannotConnect(client, proxyRequestURL)
}

func assertAppCannotConnect(client *http.Client, proxyRequestURL string) {
	resp, err := client.Get(proxyRequestURL)
	Expect(err).NotTo(HaveOccurred())

	respBytes, err := io.ReadAll(resp.Body)
	Expect(err).ToNot(HaveOccurred())
	resp.Body.Close()
	Expect(string(respBytes)).To(MatchRegexp("i/o timeout|connection refused"))
}

func assertEventuallyAppCannotConnect(client *http.Client, proxyRequestURL string) {
	Eventually(func() string {
		resp, err := client.Get(proxyRequestURL)
		Expect(err).NotTo(HaveOccurred())

		respBytes, err := io.ReadAll(resp.Body)
		Expect(err).ToNot(HaveOccurred())
		resp.Body.Close()
		return string(respBytes)
	}, 3*time.Minute).Should(MatchRegexp("i/o timeout|refused"))
}

func assertAppCanConnect(client *http.Client, proxyRequestURL string) {
	Eventually(func() string {
		resp, err := client.Get(proxyRequestURL)
		Expect(err).NotTo(HaveOccurred())

		respBytes, err := io.ReadAll(resp.Body)
		Expect(err).ToNot(HaveOccurred())
		resp.Body.Close()
		return string(respBytes)
	}, 1*time.Minute, 1*time.Second).Should(MatchRegexp("cloud_controller_v3"))
}

func assertEventuallyAppCanConnect(client *http.Client, proxyRequestURL string) {
	Eventually(func() string {
		resp, err := client.Get(proxyRequestURL)
		Expect(err).NotTo(HaveOccurred())

		respBytes, err := io.ReadAll(resp.Body)
		Expect(err).ToNot(HaveOccurred())
		resp.Body.Close()
		return string(respBytes)
	}, 3*time.Minute, 1*time.Second).Should(MatchRegexp("cloud_controller_v3"))
}

func bindCCSecurityGroup(orgName, spaceName string) string {
	destinations := []Destination{{
		IP:       "10.0.0.0/8",
		Ports:    "9024", // internal cc port
		Protocol: "tcp",
	}, {
		IP:       "192.168.0.0/16",
		Ports:    "9024", // internal cc port
		Protocol: "tcp",
	}, {
		IP:       "172.16.0.0/12",
		Ports:    "9024", // internal cc port
		Protocol: "tcp",
	}}
	securityGroupName := createSecurityGroup(destinations...)
	bindSecurityGroup(securityGroupName, orgName, spaceName)

	return securityGroupName
}
