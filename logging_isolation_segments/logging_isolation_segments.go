package isolation_segments

import (
	"fmt"
	"strings"

	. "github.com/cloudfoundry/cf-acceptance-tests/cats_suite_helpers"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gexec"

	"github.com/cloudfoundry-incubator/cf-test-helpers/cf"
	"github.com/cloudfoundry-incubator/cf-test-helpers/helpers"
	"github.com/cloudfoundry-incubator/cf-test-helpers/workflowhelpers"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/assets"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/random_name"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/v3_helpers"
)

var _ = LoggingIsolationSegmentsDescribe("LoggingIsolationSegments", func() {
	var (
		appsDomain       string
		quotaName        string
		orgGuid, orgName string
		spaceName        string
		isoSegGuid       string
	)

	BeforeEach(func() {
		orgName = random_name.CATSRandomName("ORG")
		spaceName = random_name.CATSRandomName("SPACE")
		quotaName = random_name.CATSRandomName("QUOTA")

		appsDomain = Config.GetAppsDomain()
		isoSegName := Config.GetIsolationSegmentName()

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

			v3_helpers.EntitleOrgToIsolationSegment(orgGuid, isoSegGuid)
			v3_helpers.SetDefaultIsolationSegment(orgGuid, isoSegGuid)
		})
	})

	AfterEach(func() {
		workflowhelpers.AsUser(TestSetup.AdminUserContext(), TestSetup.ShortTimeout(), func() {
			deleteOrg := cf.Cf("delete-org", orgName, "-f").Wait()
			Expect(deleteOrg).To(Exit(0), "failed to delete org")
		})
	})

	Context("When the user-provided Isolation Segment has a logging system", func() {
		It("forwards logs to the isolated logging system", func() {
			workflowhelpers.AsUser(TestSetup.RegularUserContext(), Config.DefaultTimeoutDuration(), func() {
				target := cf.Cf("target", "-o", orgName, "-s", spaceName).Wait()
				Expect(target).To(Exit(0), "failed targeting")

				appName := random_name.CATSRandomName("APP")
				Eventually(cf.Cf(
					"push", appName,
					"-p", assets.NewAssets().Dora,
					"-m", DEFAULT_MEMORY_LIMIT,
					"-d", appsDomain),
					Config.CfPushTimeoutDuration()).Should(Exit(0))

				url := fmt.Sprintf("https://%s.%s/logspew/10", appName, appsDomain)
				curlSession := helpers.CurlSkipSSL(Config.GetSkipSSLValidation(), url)
				Eventually(curlSession).Should(Exit(0))

				session := cf.Cf("app", appName, "--guid")
				Eventually(session, Config.CfPushTimeoutDuration()).Should(Exit(0))
				isoGuid := strings.TrimSpace(string(session.Out.Contents()))

				session = cf.Cf("oauth-token")
				Eventually(session, Config.CfPushTimeoutDuration()).Should(Exit(0))
				token := strings.TrimSpace(string(session.Out.Contents()))
				authHeader := fmt.Sprintf("Authorization: %s", token)

				url = fmt.Sprintf("https://iso-log-cache.%s/api/v1/meta", appsDomain)

				Eventually(func() string {
					curlSession := helpers.CurlSkipSSL(Config.GetSkipSSLValidation(), url, "-H", authHeader)
					Eventually(curlSession).Should(Exit(0))
					return strings.TrimSpace(string(curlSession.Out.Contents()))
				}, Config.LongCurlTimeoutDuration()).Should(ContainSubstring(isoGuid))
			})
		})
	})
})
