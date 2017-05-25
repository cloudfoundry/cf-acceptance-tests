package isolation_segments

import (
	"fmt"
	"io/ioutil"
	"os"

	. "github.com/cloudfoundry/cf-acceptance-tests/cats_suite_helpers"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gbytes"
	. "github.com/onsi/gomega/gexec"

	"github.com/cloudfoundry-incubator/cf-test-helpers/cf"
	"github.com/cloudfoundry-incubator/cf-test-helpers/helpers"
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
)

func setDefaultIsolationSegment(orgGuid, isoSegGuid string) {
	Eventually(cf.Cf("curl",
		fmt.Sprintf("/v3/organizations/%s/relationships/default_isolation_segment", orgGuid),
		"-X",
		"PATCH",
		"-d",
		fmt.Sprintf(`{"data":{"guid":"%s"}}`, isoSegGuid)),
		Config.DefaultTimeoutDuration()).Should(Exit(0))
}

func getDefaultIsolationSegment(orgGuid string) string {
	session := cf.Cf("curl", fmt.Sprintf("/v3/organizations/%s/relationships/default_isolation_segment", orgGuid))
	bytes := session.Wait(Config.DefaultTimeoutDuration()).Out.Contents()
	return v3_helpers.GetIsolationSegmentGuidFromResponse(bytes)
}

var _ = IsolationSegmentsDescribe("IsolationSegments", func() {
	var appsDomain string
	var orgGuid, orgName string
	var spaceGuid, spaceName string
	var isoSegGuid, isoSegName, isoSegDomain, defaultIsoSegGuid string
	var testSetup *workflowhelpers.ReproducibleTestSuiteSetup
	var created bool
	var originallyEntitled bool

	BeforeEach(func() {
		// New up a organization since we will be assigning isolation segments.
		// This has a potential to cause other tests to fail if running in parallel mode.
		cfg, _ := config.NewCatsConfig(os.Getenv("CONFIG"))
		testSetup = workflowhelpers.NewTestSuiteSetup(cfg)
		testSetup.Setup()

		appsDomain = Config.GetAppsDomain()
		orgName = testSetup.RegularUserContext().Org
		spaceName = testSetup.RegularUserContext().Space
		isoSegName = Config.GetIsolationSegmentName()
		isoSegDomain = Config.GetIsolationSegmentDomain()
		if isoSegDomain == "" {
			isoSegDomain = appsDomain
		}

		session := cf.Cf("curl", fmt.Sprintf("/v3/organizations?names=%s", orgName))
		bytes := session.Wait(Config.DefaultTimeoutDuration()).Out.Contents()
		orgGuid = v3_helpers.GetGuidFromResponse(bytes)
		defaultIsoSegGuid = getDefaultIsolationSegment(orgGuid)
		workflowhelpers.AsUser(testSetup.AdminUserContext(), testSetup.ShortTimeout(), func() {
			originallyEntitled = v3_helpers.OrgEntitledToIsolationSegment(orgGuid, isoSegName)
		})
	})

	AfterEach(func() {
		workflowhelpers.AsUser(testSetup.AdminUserContext(), testSetup.ShortTimeout(), func() {
			setDefaultIsolationSegment(orgGuid, defaultIsoSegGuid)
			if !originallyEntitled && isoSegGuid != "" {
				v3_helpers.RevokeOrgEntitlementForIsolationSegment(orgGuid, isoSegGuid)
			}
		})
		testSetup.Teardown()
	})

	Context("When an organization has the shared segment as its default", func() {
		BeforeEach(func() {
			workflowhelpers.AsUser(testSetup.AdminUserContext(), testSetup.ShortTimeout(), func() {
				v3_helpers.EntitleOrgToIsolationSegment(orgGuid, SHARED_ISOLATION_SEGMENT_GUID)
				setDefaultIsolationSegment(orgGuid, SHARED_ISOLATION_SEGMENT_GUID)
			})
		})

		It("can run an app to a space with no assigned segment", func() {
			appName := random_name.CATSRandomName("APP")
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

			Eventually(helpers.CurlingAppRoot(Config, appName), Config.DefaultTimeoutDuration()).Should(ContainSubstring(binaryHi))
		})

		It("is not reachable from the isolation segment router", func() {
			if Config.GetIsolationSegmentDomain() == "" {
				Skip(`Skipping this routing isolation segment test because Config.IsolationSegmentDomain is not set.
NOTE: Ensure that you have configured a load balancer to point to the isolation segment domain and deployed Gorouter with an isolation segment before running this test.`)
			}
			appName := random_name.CATSRandomName("APP")
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

			//send a request to app in the shared domain, but through the isolation segment router
			resp := v3_helpers.SendRequestWithHost(fmt.Sprintf("%s.%s", appName, appsDomain), isoSegDomain)
			defer resp.Body.Close()

			Expect(resp.StatusCode).To(Equal(404))
		})
	})

	Context("When the user-provided Isolation Segment has an associated cell", func() {
		BeforeEach(func() {
			workflowhelpers.AsUser(testSetup.AdminUserContext(), testSetup.ShortTimeout(), func() {
				isoSegGuid, created = v3_helpers.CreateOrGetIsolationSegment(isoSegName)
				v3_helpers.EntitleOrgToIsolationSegment(orgGuid, isoSegGuid)
				setDefaultIsolationSegment(orgGuid, isoSegGuid)
			})
		})

		AfterEach(func() {
			if created {
				v3_helpers.DeleteIsolationSegment(isoSegGuid)
			}
		})

		It("can run an app to an org where the default is the user-provided isolation segment", func() {
			appName := random_name.CATSRandomName("APP")
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

			resp := v3_helpers.SendRequestWithHost(fmt.Sprintf("%s.%s", appName, isoSegDomain), isoSegDomain)
			defer resp.Body.Close()

			Expect(resp.StatusCode).To(Equal(200))
			htmlData, err := ioutil.ReadAll(resp.Body)
			Expect(err).NotTo(HaveOccurred())
			Expect(string(htmlData)).To(ContainSubstring(binaryHi))
		})
	})

	Context("When the Isolation Segment has no associated cells", func() {
		BeforeEach(func() {
			workflowhelpers.AsUser(testSetup.AdminUserContext(), testSetup.ShortTimeout(), func() {
				isoSegGuid = v3_helpers.CreateIsolationSegment(random_name.CATSRandomName("fake-iso-seg"))
				v3_helpers.EntitleOrgToIsolationSegment(orgGuid, isoSegGuid)
				setDefaultIsolationSegment(orgGuid, isoSegGuid)
			})
		})

		AfterEach(func() {
			v3_helpers.DeleteIsolationSegment(isoSegGuid)
		})

		It("fails to start an app in the Isolation Segment", func() {
			appName := random_name.CATSRandomName("APP")
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
			Eventually(cf.Cf("start", appName), Config.CfPushTimeoutDuration()).Should(Exit(1))
		})
	})

	Context("When the organization has not been entitled to the Isolation Segment", func() {
		BeforeEach(func() {
			workflowhelpers.AsUser(testSetup.AdminUserContext(), testSetup.ShortTimeout(), func() {
				isoSegGuid, created = v3_helpers.CreateOrGetIsolationSegment(isoSegName)
			})
		})

		AfterEach(func() {
			if created {
				v3_helpers.DeleteIsolationSegment(isoSegGuid)
			}
		})

		It("fails to set the isolation segment as the default", func() {
			workflowhelpers.AsUser(TestSetup.AdminUserContext(), Config.DefaultTimeoutDuration(), func() {
				session := cf.Cf("curl",
					fmt.Sprintf("/v3/organizations/%s/relationships/default_isolation_segment", orgGuid),
					"-X",
					"PATCH",
					"-d",
					fmt.Sprintf(`{"data":{"guid":"%s"}}`, isoSegGuid)).Wait(Config.DefaultTimeoutDuration())
				Expect(session).To(Exit(0))
				Expect(session).To(Say("Ensure it has been entitled to (the|this) organization"))
			})
		})
	})

	Context("When the space has been assigned an Isolation Segment", func() {
		BeforeEach(func() {
			workflowhelpers.AsUser(testSetup.AdminUserContext(), testSetup.ShortTimeout(), func() {
				isoSegGuid, created = v3_helpers.CreateOrGetIsolationSegment(isoSegName)
				v3_helpers.EntitleOrgToIsolationSegment(orgGuid, isoSegGuid)
				session := cf.Cf("curl", fmt.Sprintf("/v3/spaces?names=%s", spaceName))
				bytes := session.Wait(Config.DefaultTimeoutDuration()).Out.Contents()
				spaceGuid = v3_helpers.GetGuidFromResponse(bytes)
				v3_helpers.AssignIsolationSegmentToSpace(spaceGuid, isoSegGuid)
			})
		})

		AfterEach(func() {
			if created {
				v3_helpers.DeleteIsolationSegment(isoSegGuid)
			}
		})

		It("can run an app in that isolation segment", func() {
			appName := random_name.CATSRandomName("APP")
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

			resp := v3_helpers.SendRequestWithHost(fmt.Sprintf("%s.%s", appName, isoSegDomain), isoSegDomain)
			defer resp.Body.Close()

			Expect(resp.StatusCode).To(Equal(200))
			htmlData, err := ioutil.ReadAll(resp.Body)
			Expect(err).NotTo(HaveOccurred())
			Expect(string(htmlData)).To(ContainSubstring(binaryHi))

		})

		It("the app is not reachable outside the isolation segment", func() {
			if Config.GetIsolationSegmentDomain() == "" {
				Skip(`Skipping this routing isolation segment test because Config.IsolationSegmentDomain is not set.
NOTE: Ensure that you have configured a load balancer to point to the isolation segment domain and deployed Gorouter with an isolation segment before running this test.`)
			}
			appName := random_name.CATSRandomName("APP")
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

			resp := v3_helpers.SendRequestWithHost(fmt.Sprintf("%s.%s", appName, isoSegDomain), appsDomain)
			defer resp.Body.Close()

			Expect(resp.StatusCode).To(Equal(404))
		})
	})
})
