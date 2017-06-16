package routing_isolation_segments

import (
	"fmt"
	"io/ioutil"
	"os"

	. "github.com/cloudfoundry/cf-acceptance-tests/cats_suite_helpers"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gexec"

	"github.com/cloudfoundry-incubator/cf-test-helpers/cf"
	"github.com/cloudfoundry-incubator/cf-test-helpers/workflowhelpers"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/app_helpers"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/assets"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/config"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/random_name"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/v3_helpers"
)

const (
	SHARED_ISOLATION_SEGMENT_GUID = "933b4c58-120b-499a-b85d-4b6fc9e2903b"
	binaryHi                      = "Hello from a binary"
	SHARED_ISOLATION_SEGMENT_NAME = "shared"
)

var _ = RoutingIsolationSegmentsDescribe("RoutingIsolationSegments", func() {
	var appsDomain string
	var orgGuid, orgName string
	var spaceGuid, spaceName string
	var isoSegGuid, isoSegName, isoSegDomain string
	var testSetup *workflowhelpers.ReproducibleTestSuiteSetup
	var created bool
	var originallyEntitledToShared bool

	BeforeEach(func() {
		// New up a organization since we will be assigning isolation segments.
		// This has a potential to cause other tests to fail if running in parallel mode.
		cfg, _ := config.NewCatsConfig(os.Getenv("CONFIG"))
		testSetup = workflowhelpers.NewTestSuiteSetup(cfg)
		testSetup.Setup()

		appsDomain = Config.GetAppsDomain()
		orgName = testSetup.RegularUserContext().Org
		spaceName = testSetup.RegularUserContext().Space
		spaceGuid = v3_helpers.GetSpaceGuidFromName(spaceName)
		isoSegName = Config.GetIsolationSegmentName()
		isoSegDomain = Config.GetIsolationSegmentDomain()

		session := cf.Cf("curl", fmt.Sprintf("/v3/organizations?names=%s", orgName))
		bytes := session.Wait(Config.DefaultTimeoutDuration()).Out.Contents()
		orgGuid = v3_helpers.GetGuidFromResponse(bytes)
		workflowhelpers.AsUser(testSetup.AdminUserContext(), testSetup.ShortTimeout(), func() {
			originallyEntitledToShared = v3_helpers.OrgEntitledToIsolationSegment(orgGuid, SHARED_ISOLATION_SEGMENT_NAME)
		})
	})

	AfterEach(func() {
		testSetup.Teardown()
		if !originallyEntitledToShared && Config.GetUseExistingOrganization() {
			workflowhelpers.AsUser(testSetup.AdminUserContext(), testSetup.ShortTimeout(), func() {
				v3_helpers.RevokeOrgEntitlementForIsolationSegment(orgGuid, SHARED_ISOLATION_SEGMENT_GUID)
			})
		}
	})

	Context("When an app is pushed to a space assigned the shared isolation segment", func() {
		var appName string

		BeforeEach(func() {
			workflowhelpers.AsUser(testSetup.AdminUserContext(), testSetup.ShortTimeout(), func() {
				v3_helpers.EntitleOrgToIsolationSegment(orgGuid, SHARED_ISOLATION_SEGMENT_GUID)
				v3_helpers.AssignIsolationSegmentToSpace(spaceGuid, SHARED_ISOLATION_SEGMENT_GUID)
				appName = random_name.CATSRandomName("APP")
			})
			Eventually(cf.Cf(
				"push", appName,
				"-p", assets.NewAssets().Binary,
				"--no-start",
				"-m", DEFAULT_MEMORY_LIMIT,
				"-b", "binary_buildpack",
				"-d", appsDomain,
				"-c", "./app"),
				Config.CfPushTimeoutDuration()).Should(Exit(0))

			app_helpers.EnableDiego(appName)
			Eventually(cf.Cf("start", appName), Config.CfPushTimeoutDuration()).Should(Exit(0))
		})

		It("is reachable from the shared router", func() {
			resp := v3_helpers.SendRequestWithSpoofedHeader(fmt.Sprintf("%s.%s", appName, appsDomain), appsDomain)
			defer resp.Body.Close()

			Expect(resp.StatusCode).To(Equal(200))
			htmlData, err := ioutil.ReadAll(resp.Body)
			Expect(err).NotTo(HaveOccurred())
			Expect(string(htmlData)).To(ContainSubstring(binaryHi))
		})

		It("is not reachable from the isolation segment router", func() {
			//send a request to app in the shared domain, but through the isolation segment router
			resp := v3_helpers.SendRequestWithSpoofedHeader(fmt.Sprintf("%s.%s", appName, appsDomain), isoSegDomain)
			defer resp.Body.Close()

			Expect(resp.StatusCode).To(Equal(404))
		})
	})

	Context("When an app is pushed to a space that has been assigned an Isolation Segment", func() {
		var appName string

		BeforeEach(func() {
			workflowhelpers.AsUser(testSetup.AdminUserContext(), testSetup.ShortTimeout(), func() {
				isoSegGuid, created = v3_helpers.CreateOrGetIsolationSegment(isoSegName)
				v3_helpers.EntitleOrgToIsolationSegment(orgGuid, isoSegGuid)
				session := cf.Cf("curl", fmt.Sprintf("/v3/spaces?names=%s", spaceName))
				bytes := session.Wait(Config.DefaultTimeoutDuration()).Out.Contents()
				spaceGuid = v3_helpers.GetGuidFromResponse(bytes)
				v3_helpers.AssignIsolationSegmentToSpace(spaceGuid, isoSegGuid)
			})
			appName = random_name.CATSRandomName("APP")
			Eventually(cf.Cf(
				"push", appName,
				"-p", assets.NewAssets().Binary,
				"--no-start",
				"-m", DEFAULT_MEMORY_LIMIT,
				"-b", "binary_buildpack",
				"-d", isoSegDomain,
				"-c", "./app"),
				Config.CfPushTimeoutDuration()).Should(Exit(0))

			app_helpers.EnableDiego(appName)
			Eventually(cf.Cf("start", appName), Config.CfPushTimeoutDuration()).Should(Exit(0))
		})

		AfterEach(func() {
			if created {
				workflowhelpers.AsUser(testSetup.AdminUserContext(), testSetup.ShortTimeout(), func() {
					v3_helpers.UnassignIsolationSegmentFromSpace(spaceGuid)
					v3_helpers.RevokeOrgEntitlementForIsolationSegment(orgGuid, isoSegGuid)
					v3_helpers.DeleteIsolationSegment(isoSegGuid)
				})
			}
		})
		It("the app is reachable from the isolated router", func() {
			resp := v3_helpers.SendRequestWithSpoofedHeader(fmt.Sprintf("%s.%s", appName, isoSegDomain), isoSegDomain)
			defer resp.Body.Close()

			Expect(resp.StatusCode).To(Equal(200))
			htmlData, err := ioutil.ReadAll(resp.Body)
			Expect(err).NotTo(HaveOccurred())
			Expect(string(htmlData)).To(ContainSubstring(binaryHi))
		})

		It("the app is not reachable from the shared router", func() {

			resp := v3_helpers.SendRequestWithSpoofedHeader(fmt.Sprintf("%s.%s", appName, isoSegDomain), appsDomain)
			defer resp.Body.Close()

			Expect(resp.StatusCode).To(Equal(404))
		})
	})
})
