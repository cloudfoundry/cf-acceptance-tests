package routing_isolation_segments

import (
	"fmt"
	"io/ioutil"

	. "github.com/cloudfoundry/cf-acceptance-tests/cats_suite_helpers"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gexec"

	"github.com/cloudfoundry-incubator/cf-test-helpers/cf"
	"github.com/cloudfoundry-incubator/cf-test-helpers/workflowhelpers"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/app_helpers"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/assets"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/random_name"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/v3_helpers"
)

const (
	SHARED_ISOLATION_SEGMENT_GUID = "933b4c58-120b-499a-b85d-4b6fc9e2903b"
	binaryHi                      = "Hello from a binary"
)

var _ = RoutingIsolationSegmentsDescribe("RoutingIsolationSegments", func() {
	var (
		appsDomain                           string
		quotaName                            string
		orgGuid, orgName                     string
		spaceGuid, spaceName                 string
		isoSegGuid, isoSegName, isoSegDomain string
	)

	BeforeEach(func() {
		orgName = random_name.CATSRandomName("ORG")
		spaceName = random_name.CATSRandomName("SPACE")
		quotaName = random_name.CATSRandomName("QUOTA")

		appsDomain = Config.GetAppsDomain()
		isoSegName = Config.GetIsolationSegmentName()
		isoSegDomain = Config.GetIsolationSegmentDomain()

		workflowhelpers.AsUser(TestSetup.AdminUserContext(), Config.DefaultTimeoutDuration(), func() {
			createQuota := cf.Cf("create-quota", quotaName, "-m", "10G", "-r", "1000", "-s", "5").Wait(TestSetup.ShortTimeout())
			Expect(createQuota).To(Exit(0))

			createOrg := cf.Cf("create-org", orgName).Wait(Config.DefaultTimeoutDuration())
			Expect(createOrg).To(Exit(0), "failed to create org")

			setQuota := cf.Cf("set-quota", orgName, quotaName).Wait(TestSetup.ShortTimeout())
			Expect(setQuota).To(Exit(0))

			createSpace := cf.Cf("create-space", spaceName, "-o", orgName).Wait(Config.DefaultTimeoutDuration())
			Expect(createSpace).To(Exit(0), "failed to create space")

			session := cf.Cf("curl", fmt.Sprintf("/v3/organizations?names=%s", orgName))
			bytes := session.Wait(Config.DefaultTimeoutDuration()).Out.Contents()
			orgGuid = v3_helpers.GetGuidFromResponse(bytes)

			isoSegGuid = v3_helpers.CreateOrGetIsolationSegment(isoSegName)
		})
	})

	AfterEach(func() {
		workflowhelpers.AsUser(TestSetup.AdminUserContext(), TestSetup.ShortTimeout(), func() {
			deleteOrg := cf.Cf("delete-org", orgName, "-f").Wait(Config.DefaultTimeoutDuration())
			Expect(deleteOrg).To(Exit(0), "failed to delete org")
		})
	})

	Context("When an app is pushed to a space assigned the shared isolation segment", func() {
		var appName string

		BeforeEach(func() {
			workflowhelpers.AsUser(TestSetup.AdminUserContext(), TestSetup.ShortTimeout(), func() {
				v3_helpers.EntitleOrgToIsolationSegment(orgGuid, SHARED_ISOLATION_SEGMENT_GUID)
				v3_helpers.AssignIsolationSegmentToSpace(spaceGuid, SHARED_ISOLATION_SEGMENT_GUID)
				target := cf.Cf("target", "-o", orgName, "-s", spaceName).Wait(Config.DefaultTimeoutDuration())
				Expect(target).To(Exit(0), "failed targeting")
				appName = random_name.CATSRandomName("APP")

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
			workflowhelpers.AsUser(TestSetup.AdminUserContext(), TestSetup.ShortTimeout(), func() {
				v3_helpers.EntitleOrgToIsolationSegment(orgGuid, isoSegGuid)
				session := cf.Cf("curl", fmt.Sprintf("/v3/spaces?names=%s", spaceName))
				bytes := session.Wait(Config.DefaultTimeoutDuration()).Out.Contents()
				spaceGuid = v3_helpers.GetGuidFromResponse(bytes)
				v3_helpers.AssignIsolationSegmentToSpace(spaceGuid, isoSegGuid)

				target := cf.Cf("target", "-o", orgName, "-s", spaceName).Wait(Config.DefaultTimeoutDuration())
				Expect(target).To(Exit(0), "failed targeting")
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
