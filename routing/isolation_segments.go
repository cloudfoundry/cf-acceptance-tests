package routing

import (
	"fmt"
	"net/http"
	"os"
	"strings"

	"code.cloudfoundry.org/cf-routing-test-helpers/helpers"
	"github.com/cloudfoundry-incubator/cf-test-helpers/cf"
	"github.com/cloudfoundry-incubator/cf-test-helpers/workflowhelpers"
	. "github.com/cloudfoundry/cf-acceptance-tests/cats_suite_helpers"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/assets"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/config"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/random_name"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/v3_helpers"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
)

const (
	sharedIsoSegGUID = "933b4c58-120b-499a-b85d-4b6fc9e2903b"
)

var _ = RoutingIsolationSegmentsDescribe("Routing Isolation Segments", func() {
	var (
		app1              string
		helloRoutingAsset = assets.NewAssets().HelloRouting
		orgName           string
		orgGUID           string
		spaceName         string
		isoOrgName        string
		isoOrgGUID        string
		isoSpaceName      string
		isoSegGUID        string
		isoSegName        string
		isoSegDomain      string
		testSetup         *workflowhelpers.ReproducibleTestSuiteSetup
		newIsoSeg         bool
	)

	BeforeEach(func() {
		cfg, _ := config.NewCatsConfig(os.Getenv("CONFIG"))
		isoSegName = Config.GetRoutingIsolationSegmentName()
		isoSegDomain = Config.GetRoutingIsolationSegmentDomain()
		Expect(isoSegName).NotTo(Equal(""), "RoutingIsolationSegmentName must be provided")
		Expect(isoSegDomain).NotTo(Equal(""), "RoutingIsolationSegmentDomain must be provided")
		Expect(Config.GetBackend()).To(Equal("diego"), "Backend must be diego")

		testSetup = workflowhelpers.NewTestSuiteSetup(cfg)
		testSetup.Setup()

		orgName = testSetup.RegularUserContext().Org
		spaceName = testSetup.RegularUserContext().Space

		isoOrgName = random_name.CATSRandomName("IsoOrg")
		workflowhelpers.AsUser(testSetup.AdminUserContext(), testSetup.ShortTimeout(), func() {
			Eventually(cf.Cf("create-org", isoOrgName), Config.DefaultTimeoutDuration()).Should(gexec.Exit(0))
			session := cf.Cf("org", isoOrgName, "--guid")
			bytes := session.Wait(Config.DefaultTimeoutDuration()).Out.Contents()
			isoOrgGUID = strings.TrimSpace(string(bytes))

			session = cf.Cf("org", orgName, "--guid")
			bytes = session.Wait(Config.DefaultTimeoutDuration()).Out.Contents()
			orgGUID = strings.TrimSpace(string(bytes))
		})
	})

	AfterEach(func() {
		testSetup.Teardown()
	})

	Context("with an app deployed in a shared segment", func() {
		BeforeEach(func() {
			workflowhelpers.AsUser(testSetup.AdminUserContext(), testSetup.ShortTimeout(), func() {
				v3_helpers.EntitleOrgToIsolationSegment(orgGUID, sharedIsoSegGUID)
			})
			app1 = random_name.CATSRandomName("APP")
			helpers.PushApp(app1, helloRoutingAsset, Config.GetRubyBuildpackName(), Config.GetAppsDomain(), Config.CfPushTimeoutDuration(), DEFAULT_MEMORY_LIMIT)
		})
		It("the app in shared responds with 200", func() {
			req, _ := http.NewRequest("GET", fmt.Sprintf("http://%s.%s", app1, Config.GetAppsDomain()), nil)

			resp, err := http.DefaultClient.Do(req)
			defer resp.Body.Close()

			Expect(err).NotTo(HaveOccurred())
			Expect(resp.StatusCode).To(Equal(200))
		})
		It("the app is not reachable from the isolation segment", func() {
			req, _ := http.NewRequest("GET", fmt.Sprintf("http://iso-router.%s", isoSegDomain), nil)
			req.Host = fmt.Sprintf("%s.%s", app1, Config.GetAppsDomain())

			resp, err := http.DefaultClient.Do(req)
			defer resp.Body.Close()

			Expect(err).NotTo(HaveOccurred())
			Expect(resp.StatusCode).To(Equal(404))
		})
	})
	Context("with an app deployed in an isolation segment", func() {
		BeforeEach(func() {
			isoSpaceName = random_name.CATSRandomName("IsoSpace")
			workflowhelpers.AsUser(testSetup.AdminUserContext(), testSetup.ShortTimeout(), func() {
				newIsoSeg = false
				if !v3_helpers.IsolationSegmentExists(isoSegName) {
					newIsoSeg = true
					v3_helpers.CreateIsolationSegment(isoSegName)
				}
				isoSegGUID = v3_helpers.GetIsolationSegmentGuid(isoSegName)

				Expect(v3_helpers.IsolationSegmentExists(isoSegName)).To(BeTrue())
				v3_helpers.EntitleOrgToIsolationSegment(isoOrgGUID, isoSegGUID)
				Eventually(cf.Cf("target", "-o", isoOrgName), Config.DefaultTimeoutDuration()).Should(gexec.Exit(0))
				Eventually(cf.Cf("create-space", "-o", isoOrgName, isoSpaceName), Config.DefaultTimeoutDuration()).Should(gexec.Exit(0))
				session := cf.Cf("space", isoSpaceName, "--guid")
				bytes := session.Wait(Config.DefaultTimeoutDuration()).Out.Contents()
				isoSpaceGUID := strings.TrimSpace(string(bytes))

				v3_helpers.AssignIsolationSegmentToSpace(isoSpaceGUID, isoSegGUID)
				//Add test user to space
				Eventually(cf.Cf("set-space-role", testSetup.RegularUserContext().TestUser.Username(), isoOrgName, isoSpaceName, "SpaceDeveloper"), Config.DefaultTimeoutDuration()).Should(gexec.Exit(0))
			})

			app1 = random_name.CATSRandomName("APP")
			Eventually(cf.Cf("target", "-o", isoOrgName, "-s", isoSpaceName), Config.DefaultTimeoutDuration()).Should(gexec.Exit(0))

			helpers.PushApp(app1, helloRoutingAsset, Config.GetRubyBuildpackName(), Config.GetAppsDomain(), Config.CfPushTimeoutDuration(), DEFAULT_MEMORY_LIMIT)

		})
		AfterEach(func() {
			helpers.AppReport(app1, Config.DefaultTimeoutDuration())
			helpers.DeleteApp(app1, Config.DefaultTimeoutDuration())
			workflowhelpers.AsUser(testSetup.AdminUserContext(), testSetup.ShortTimeout(), func() {
				Eventually(cf.Cf("delete-space", "-f", "-o", isoOrgName, isoSpaceName), Config.DefaultTimeoutDuration()).Should(gexec.Exit(0))
				v3_helpers.RevokeOrgEntitlementForIsolationSegment(isoOrgGUID, isoSegGUID)
				Eventually(cf.Cf("delete-org", "-f", isoOrgName), Config.DefaultTimeoutDuration()).Should(gexec.Exit(0))
				if newIsoSeg {
					v3_helpers.DeleteIsolationSegment(isoSegGUID)
				}
			})
		})

		It("the app in IS responds with 200", func() {
			req, _ := http.NewRequest("GET", fmt.Sprintf("http://iso-router.%s", isoSegDomain), nil)
			req.Host = fmt.Sprintf("%s.%s", app1, Config.GetAppsDomain())

			resp, err := http.DefaultClient.Do(req)
			defer resp.Body.Close()

			Expect(err).NotTo(HaveOccurred())
			Expect(resp.StatusCode).To(Equal(200))
		})

		It("the app is not reachable outside the isolation segment", func() {
			req, _ := http.NewRequest("GET", fmt.Sprintf("http://shared-router.%s", Config.GetAppsDomain()), nil)
			req.Host = fmt.Sprintf("%s.%s", app1, Config.GetAppsDomain())

			resp, err := http.DefaultClient.Do(req)
			defer resp.Body.Close()

			Expect(err).NotTo(HaveOccurred())
			Expect(resp.StatusCode).To(Equal(404), `The shared gorouter must be configured to "shared_and_segments" to perform this validation.`)
		})
	})
})
