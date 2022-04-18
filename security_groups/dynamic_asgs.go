package security_groups_test

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	. "github.com/cloudfoundry/cf-acceptance-tests/cats_suite_helpers"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gexec"

	"github.com/cloudfoundry-incubator/cf-test-helpers/cf"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/assets"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/random_name"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/skip_messages"
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
		proxyRequestURL := fmt.Sprintf("http://%s.%s/https_proxy/cloud-controller-ng.service.cf.internal:9024/v2/info", appName, Config.GetAppsDomain())

		By("checking that our app can't initially reach cloud controller over internal address")
		resp, err := http.Get(proxyRequestURL)
		Expect(err).NotTo(HaveOccurred())

		respBytes, err := ioutil.ReadAll(resp.Body)
		Expect(err).ToNot(HaveOccurred())
		resp.Body.Close()
		Expect(respBytes).To(MatchRegexp("refused"))

		By("binding a new security group")
		dest := Destination{
			IP:       "10.0.0.0/0",
			Ports:    "9024", // internal cc port
			Protocol: "tcp",
		}
		securityGroupName = createSecurityGroup(dest)
		bindSecurityGroup(securityGroupName, orgName, spaceName)

		By("checking that our app can now reach cloud controller over internal address")
		Eventually(func() []byte {
			resp, err = http.Get(proxyRequestURL)
			Expect(err).NotTo(HaveOccurred())

			respBytes, err = ioutil.ReadAll(resp.Body)
			Expect(err).ToNot(HaveOccurred())
			resp.Body.Close()
			return respBytes
		}, 3*time.Minute).Should(MatchRegexp("api_version"))

		By("unbinding the security group")
		unbindSecurityGroup(securityGroupName, orgName, spaceName)

		By("checking that our app can no longer reach cloud controller over internal address")
		Eventually(func() []byte {
			resp, err = http.Get(proxyRequestURL)
			Expect(err).NotTo(HaveOccurred())

			respBytes, err = ioutil.ReadAll(resp.Body)
			Expect(err).ToNot(HaveOccurred())
			resp.Body.Close()
			return respBytes
		}, 3*time.Minute).Should(MatchRegexp("refused"))
	})
})
