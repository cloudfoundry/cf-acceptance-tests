package security_groups_test

import (
	"crypto/tls"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
	"time"

	. "github.com/cloudfoundry/cf-acceptance-tests/cats_suite_helpers"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gexec"

	"github.com/cloudfoundry/cf-acceptance-tests/helpers/assets"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/random_name"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/skip_messages"
	"github.com/cloudfoundry/cf-test-helpers/v2/cf"
)

var _ = Describe("Dynamic ASGs", func() {
	var (
		orgName           string
		spaceName         string
		appName           string
		securityGroupName string
	)

	BeforeEach(func() {
		if !Config.GetIncludeSecurityGroups() {
			Skip(skip_messages.SkipSecurityGroupsMessage)
		}

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

	It("applies ASGs wihout app restart", func() {
		endpointHostPortPath := fmt.Sprintf("%s:%d%s", Config.GetDynamicASGTestConfig().EndpointHost, Config.GetDynamicASGTestConfig().EndpointPort, Config.GetDynamicASGTestConfig().EndpointPath)

		proxyRequestURL := fmt.Sprintf("%s%s.%s/https_proxy/%s", Config.Protocol(), appName, Config.GetAppsDomain(), endpointHostPortPath)

		client := &http.Client{
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{
					InsecureSkipVerify: Config.GetSkipSSLValidation(),
				},
			},
		}

		By(fmt.Sprintf("checking that our app can't initially reach %s", endpointHostPortPath))
		resp, err := client.Get(proxyRequestURL)
		Expect(err).NotTo(HaveOccurred())

		respBytes, err := ioutil.ReadAll(resp.Body)
		Expect(err).ToNot(HaveOccurred())
		resp.Body.Close()
		Expect(respBytes).To(MatchRegexp("refused"))

		By("binding a new security group")
		dest := Destination{
			IP:       Config.GetDynamicASGTestConfig().EndpointAllowIPRange,
			Ports:    strconv.Itoa(Config.GetDynamicASGTestConfig().EndpointPort),
			Protocol: "tcp",
		}
		securityGroupName = createSecurityGroup(dest)
		bindSecurityGroup(securityGroupName, orgName, spaceName)

		By(fmt.Sprintf("checking that our app can now reach %s", endpointHostPortPath))
		Eventually(func() []byte {
			resp, err = client.Get(proxyRequestURL)
			Expect(err).NotTo(HaveOccurred())

			respBytes, err = ioutil.ReadAll(resp.Body)
			Expect(err).ToNot(HaveOccurred())
			resp.Body.Close()
			return respBytes
		}, 3*time.Minute).Should(MatchRegexp(Config.GetDynamicASGTestConfig().ExpectedResponseRegex))

		By("unbinding the security group")
		unbindSecurityGroup(securityGroupName, orgName, spaceName)

		By(fmt.Sprintf("checking that our app can no longer reach %s", endpointHostPortPath))
		Eventually(func() []byte {
			resp, err = client.Get(proxyRequestURL)
			Expect(err).NotTo(HaveOccurred())

			respBytes, err = ioutil.ReadAll(resp.Body)
			Expect(err).ToNot(HaveOccurred())
			resp.Body.Close()
			return respBytes
		}, 3*time.Minute).Should(MatchRegexp("refused"))
	})
})
