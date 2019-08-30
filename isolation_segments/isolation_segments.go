package isolation_segments

import (
	"fmt"

	. "github.com/cloudfoundry/cf-acceptance-tests/cats_suite_helpers"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gbytes"
	. "github.com/onsi/gomega/gexec"

	"github.com/cloudfoundry-incubator/cf-test-helpers/cf"
	"github.com/cloudfoundry-incubator/cf-test-helpers/helpers"
	"github.com/cloudfoundry-incubator/cf-test-helpers/workflowhelpers"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/assets"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/random_name"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/v3_helpers"
)

const (
	SHARED_ISOLATION_SEGMENT_GUID = "933b4c58-120b-499a-b85d-4b6fc9e2903b"
	binaryHi                      = "Hello from a binary"
)

var _ = IsolationSegmentsDescribe("IsolationSegments", func() {
	var (
		appsDomain, isoSegDomain string
		quotaName                string
		orgGuid, orgName         string
		spaceGuid, spaceName     string
		isoSegGuid, isoSegName   string
	)

	BeforeEach(func() {
		orgName = random_name.CATSRandomName("ORG")
		spaceName = random_name.CATSRandomName("SPACE")
		quotaName = random_name.CATSRandomName("QUOTA")

		appsDomain = Config.GetAppsDomain()
		isoSegName = Config.GetIsolationSegmentName()
		isoSegDomain = Config.GetIsolationSegmentDomain()
		if isoSegDomain == "" {
			isoSegDomain = appsDomain
		}

		workflowhelpers.AsUser(TestSetup.AdminUserContext(), Config.DefaultTimeoutDuration(), func() {
			createQuota := cf.Cf("create-quota", quotaName, "-m", "10G", "-r", "1000", "-s", "5").Wait(TestSetup.ShortTimeout())
			Expect(createQuota).To(Exit(0))

			createOrg := cf.Cf("create-org", orgName).Wait()
			Expect(createOrg).To(Exit(0), "failed to create org")

			setQuota := cf.Cf("set-quota", orgName, quotaName).Wait(TestSetup.ShortTimeout())
			Expect(setQuota).To(Exit(0))

			createSpace := cf.Cf("create-space", spaceName, "-o", orgName).Wait()
			Expect(createSpace).To(Exit(0), "failed to create space")

			addSpaceDeveloper := cf.Cf("set-space-role", TestSetup.RegularUserContext().Username, orgName, spaceName, "SpaceDeveloper").Wait()
			Expect(addSpaceDeveloper).To(Exit(0), "failed to add space developer role")

			session := cf.Cf("curl", fmt.Sprintf("/v3/organizations?names=%s", orgName))
			bytes := session.Wait().Out.Contents()
			orgGuid = v3_helpers.GetGuidFromResponse(bytes)

			isoSegGuid = v3_helpers.CreateOrGetIsolationSegment(isoSegName)
		})
	})

	AfterEach(func() {
		workflowhelpers.AsUser(TestSetup.AdminUserContext(), TestSetup.ShortTimeout(), func() {
			deleteOrg := cf.Cf("delete-org", orgName, "-f").Wait()
			Expect(deleteOrg).To(Exit(0), "failed to delete org")
		})
	})

	Context("When an organization has the shared segment as its default", func() {
		BeforeEach(func() {
			workflowhelpers.AsUser(TestSetup.AdminUserContext(), TestSetup.ShortTimeout(), func() {
				v3_helpers.EntitleOrgToIsolationSegment(orgGuid, SHARED_ISOLATION_SEGMENT_GUID)
				v3_helpers.SetDefaultIsolationSegment(orgGuid, SHARED_ISOLATION_SEGMENT_GUID)
			})
		})

		It("can run an app to a space with no assigned segment", func() {
			workflowhelpers.AsUser(TestSetup.RegularUserContext(), Config.DefaultTimeoutDuration(), func() {
				target := cf.Cf("target", "-o", orgName, "-s", spaceName).Wait()
				Expect(target).To(Exit(0), "failed targeting")

				appName := random_name.CATSRandomName("APP")
				Eventually(cf.Cf(
					"push", appName,
					"-p", assets.NewAssets().Binary,
					"-m", DEFAULT_MEMORY_LIMIT,
					"-b", "binary_buildpack",
					"-c", "./app"),
					Config.CfPushTimeoutDuration()).Should(Exit(0))

				Eventually(helpers.CurlingAppRoot(Config, appName)).Should(ContainSubstring(binaryHi))
			})
		})
	})

	Context("When the user-provided Isolation Segment has an associated cell", func() {
		BeforeEach(func() {
			workflowhelpers.AsUser(TestSetup.AdminUserContext(), TestSetup.ShortTimeout(), func() {
				v3_helpers.EntitleOrgToIsolationSegment(orgGuid, isoSegGuid)
				v3_helpers.SetDefaultIsolationSegment(orgGuid, isoSegGuid)
			})
		})

		It("can run an app to an org where the default is the user-provided isolation segment", func() {
			workflowhelpers.AsUser(TestSetup.RegularUserContext(), Config.DefaultTimeoutDuration(), func() {
				target := cf.Cf("target", "-o", orgName, "-s", spaceName).Wait()
				Expect(target).To(Exit(0), "failed targeting")

				appName := random_name.CATSRandomName("APP")
				Eventually(cf.Push(
					appName,
					"-p", assets.NewAssets().Binary,
					"-m", DEFAULT_MEMORY_LIMIT,
					"-b", "binary_buildpack",
					"-d", isoSegDomain,
					"-c", "./app"),
					Config.CfPushTimeoutDuration()).Should(Exit(0))

				hostHeader := fmt.Sprintf("Host: %s.%s", appName, isoSegDomain)
				host := fmt.Sprintf("http://wildcard-path.%s", isoSegDomain)

				curlSession := helpers.CurlSkipSSL(Config.GetSkipSSLValidation(), host, "-H", hostHeader)
				Eventually(curlSession).Should(Exit(0))
				Expect(curlSession.Out).To(Say(binaryHi))
			})
		})
	})

	Context("When the Isolation Segment has no associated cells", func() {
		var (
			fakeIsoSegGuid string
		)

		BeforeEach(func() {
			workflowhelpers.AsUser(TestSetup.AdminUserContext(), TestSetup.ShortTimeout(), func() {
				fakeIsoSegGuid = v3_helpers.CreateIsolationSegment(random_name.CATSRandomName("fake-iso-seg"))
				v3_helpers.EntitleOrgToIsolationSegment(orgGuid, fakeIsoSegGuid)
				v3_helpers.SetDefaultIsolationSegment(orgGuid, fakeIsoSegGuid)
			})
		})

		AfterEach(func() {
			workflowhelpers.AsUser(TestSetup.AdminUserContext(), TestSetup.ShortTimeout(), func() {
				v3_helpers.UnsetDefaultIsolationSegment(orgGuid)
				v3_helpers.RevokeOrgEntitlementForIsolationSegment(orgGuid, fakeIsoSegGuid)
				v3_helpers.DeleteIsolationSegment(fakeIsoSegGuid)
			})
		})

		It("fails to start an app in the Isolation Segment", func() {
			workflowhelpers.AsUser(TestSetup.RegularUserContext(), Config.DefaultTimeoutDuration(), func() {
				target := cf.Cf("target", "-o", orgName, "-s", spaceName).Wait()
				Expect(target).To(Exit(0), "failed targeting")

				appName := random_name.CATSRandomName("APP")
				Eventually(cf.Push(
					appName,
					"-p", assets.NewAssets().Binary,
					"-m", DEFAULT_MEMORY_LIMIT,
					"-b", "binary_buildpack",
					"-d", isoSegDomain,
					"-c", "./app"),
					Config.CfPushTimeoutDuration()).Should(Exit(1))
			})
		})
	})

	Context("When the organization has not been entitled to the Isolation Segment", func() {
		It("fails to set the isolation segment as the default", func() {
			workflowhelpers.AsUser(TestSetup.AdminUserContext(), Config.DefaultTimeoutDuration(), func() {
				session := cf.Cf("curl",
					fmt.Sprintf("/v3/organizations/%s/relationships/default_isolation_segment", orgGuid),
					"-X",
					"PATCH",
					"-d",
					fmt.Sprintf(`{"data":{"guid":"%s"}}`, isoSegGuid)).Wait()
				Expect(session).To(Exit(0))
				Expect(session).To(Say("Ensure it has been entitled to (the|this) organization"))
			})
		})
	})

	Context("When the space has been assigned an Isolation Segment", func() {
		BeforeEach(func() {
			workflowhelpers.AsUser(TestSetup.AdminUserContext(), TestSetup.ShortTimeout(), func() {
				v3_helpers.EntitleOrgToIsolationSegment(orgGuid, isoSegGuid)
				session := cf.Cf("curl", fmt.Sprintf("/v3/spaces?names=%s", spaceName))
				bytes := session.Wait().Out.Contents()
				spaceGuid = v3_helpers.GetGuidFromResponse(bytes)
				v3_helpers.AssignIsolationSegmentToSpace(spaceGuid, isoSegGuid)
			})
		})

		AfterEach(func() {
			workflowhelpers.AsUser(TestSetup.AdminUserContext(), TestSetup.ShortTimeout(), func() {
				v3_helpers.UnassignIsolationSegmentFromSpace(spaceGuid)
			})
		})

		It("can run an app in that isolation segment", func() {
			workflowhelpers.AsUser(TestSetup.RegularUserContext(), Config.DefaultTimeoutDuration(), func() {
				target := cf.Cf("target", "-o", orgName, "-s", spaceName).Wait()
				Expect(target).To(Exit(0), "failed targeting")

				appName := random_name.CATSRandomName("APP")
				Eventually(cf.Push(
					appName,
					"-p", assets.NewAssets().Binary,
					"-m", DEFAULT_MEMORY_LIMIT,
					"-b", "binary_buildpack",
					"-d", isoSegDomain,
					"-c", "./app"),
					Config.CfPushTimeoutDuration()).Should(Exit(0))

				hostHeader := fmt.Sprintf("Host: %s.%s", appName, isoSegDomain)
				host := fmt.Sprintf("http://wildcard-path.%s", isoSegDomain)

				curlSession := helpers.CurlSkipSSL(Config.GetSkipSSLValidation(), host, "-H", hostHeader)
				Eventually(curlSession).Should(Exit(0))
				Expect(curlSession.Out).To(Say(binaryHi))
			})
		})
	})
})
