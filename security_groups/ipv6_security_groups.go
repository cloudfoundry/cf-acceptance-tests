package security_groups_test

import (
	"net"
	"strings"

	. "github.com/cloudfoundry/cf-acceptance-tests/cats_suite_helpers"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gexec"

	"github.com/cloudfoundry/cf-acceptance-tests/helpers/app_helpers"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/assets"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/random_name"
	"github.com/cloudfoundry/cf-test-helpers/v2/cf"
	"github.com/cloudfoundry/cf-test-helpers/v2/helpers"
)

var _ = IPv6SecurityGroupsDescribe("IPv6 Security Group", func() {
	var (
		appName           string
		securityGroupName string
		orgName           string
		spaceName         string
	)

	BeforeEach(func() {
		appName = random_name.CATSRandomName("APP-IPv6")
		orgName = TestSetup.RegularUserContext().Org
		spaceName = TestSetup.RegularUserContext().Space
		securityGroupName = "ipv6_public_networks"

		By("pushing simple python app")
		Expect(cf.Cf(
			"push", appName,
			"-p", assets.NewAssets().Python,
			"-m", DEFAULT_MEMORY_LIMIT,
		).Wait(Config.CfPushTimeoutDuration())).To(Exit(0))
	})

	AfterEach(func() {
		app_helpers.AppReport(appName)
		Expect(cf.Cf("delete", appName, "-f", "-r").Wait()).To(Exit(0))
	})

	assertAppCanConnect := func() {
		response := helpers.CurlAppWithStatusCode(Config, appName, "/ipv4-test")
		responseParts := strings.Split(response, "\n")
		ipAddress := responseParts[0]
		statusCode := responseParts[1]

		parsedIP := net.ParseIP(ipAddress)
		Expect(parsedIP).NotTo(BeNil(), "Expected a valid IP address")
		Expect(statusCode).To(Equal("200"))
	}

	assertAppCanNotConnect := func() {
		response := helpers.CurlAppWithStatusCode(Config, appName, "/ipv4-test")
		responseParts := strings.Split(response, "\n")
		ipAddress := responseParts[0]
		statusCode := responseParts[1]

		bodyResponce := net.ParseIP(ipAddress)
		Expect(bodyResponce).To(BeNil(), "Expected a non-valid IP address")
		Expect(bodyResponce).To(ContainSubstring("not supported by protocol"))
		Expect(statusCode).To(Equal("500"))
	}

	Describe("Default IPv6 security groups are working", func() {
		It("validates IPv6 with security groups enabled", func() {
			assertAppCanConnect()
		})

		It("unbinds the wide-open security group", func() {
			By("unbinding the wide-open security group")
			unbindSecurityGroup(securityGroupName, orgName, spaceName)
		})

		It("validates IPv6 with security groups disabled", func() {
			assertAppCanNotConnect()
		})

		It("binds the wide-open security group", func() {
			By("binding the wide-open security group")
			bindSecurityGroup(securityGroupName, orgName, spaceName)
		})

		It("validates IPv6 with security groups enabled", func() {
			assertAppCanConnect()
		})

	})
})
