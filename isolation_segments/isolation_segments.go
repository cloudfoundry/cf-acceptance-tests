package isolation_segments

import (
	"fmt"
	"path/filepath"

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
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/random_name"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/v3_helpers"
)

const (
	SHARED_ISOLATION_SEGMENT_GUID = "933b4c58-120b-499a-b85d-4b6fc9e2903b"
	binaryHi                      = "Hello from a binary"
	IsolationSegRouterGroupName   = "default-tcp"
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
			createQuota := cf.Cf("create-quota", quotaName, "-m", "10G", "-r", "1000", "-s", "5", "--reserved-route-ports", "20").Wait(TestSetup.ShortTimeout())
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
		TestSetup.RegularUserContext().TargetSpace()
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
				host := fmt.Sprintf(Config.Protocol()+"wildcard-path.%s", isoSegDomain)

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
			TestSetup.RegularUserContext().TargetSpace()
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
			TestSetup.RegularUserContext().TargetSpace()
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
				host := fmt.Sprintf(Config.Protocol()+"wildcard-path.%s", isoSegDomain)

				curlSession := helpers.CurlSkipSSL(Config.GetSkipSSLValidation(), host, "-H", hostHeader)
				Eventually(curlSession).Should(Exit(0))
				Expect(curlSession.Out).To(Say(binaryHi))
			})
		})
	})

	IsolatedTCPRoutingDescribe("When TCP routing is enabled", func() {

		var domainName string

		BeforeEach(func() {
			domainName = fmt.Sprintf("tcp.%s", isoSegDomain)
			workflowhelpers.AsUser(TestSetup.AdminUserContext(), Config.DefaultTimeoutDuration(), func() {
				v3_helpers.EntitleOrgToIsolationSegment(orgGuid, isoSegGuid)
				session := cf.Cf("curl", fmt.Sprintf("/v3/spaces?names=%s", spaceName))
				bytes := session.Wait().Out.Contents()
				spaceGuid = v3_helpers.GetGuidFromResponse(bytes)
				v3_helpers.AssignIsolationSegmentToSpace(spaceGuid, isoSegGuid)

				routerGroupOutput := string(cf.Cf("router-groups").Wait().Out.Contents())
				Expect(routerGroupOutput).To(
					MatchRegexp(fmt.Sprintf("%s\\s+tcp", IsolationSegRouterGroupName)),
					fmt.Sprintf("Router group %s of type tcp doesn't exist", IsolationSegRouterGroupName),
				)

				Expect(cf.Cf("create-shared-domain",
					domainName,
					"--router-group", IsolationSegRouterGroupName,
				).Wait()).To(Exit())
			})
		})

		Context("external ports", func() {
			var (
				appName            string
				tcpDropletReceiver = assets.NewAssets().TCPListener
				serverId1          = "server1"
				externalPort1      string
			)

			BeforeEach(func() {
				appName = random_name.CATSRandomName("APP")
				cmd := fmt.Sprintf("tcp-listener --serverId=%s", serverId1)

				target := cf.Cf("target", "-o", orgName, "-s", spaceName).Wait()
				Expect(target).To(Exit(0), "failed targeting")

				Expect(cf.Cf("push",
					"--no-route",
					"--no-start",
					appName,
					"-p", tcpDropletReceiver,
					"-b", Config.GetGoBuildpackName(),
					"-m", DEFAULT_MEMORY_LIMIT,
					"-f", filepath.Join(tcpDropletReceiver, "manifest.yml"),
					"-c", cmd,
				).Wait()).To(Exit(0))
				externalPort1 = MapTCPRoute(appName, domainName)
				appStart := cf.Cf("start", appName).Wait(Config.CfPushTimeoutDuration())
				Expect(appStart).To(Exit(0))
				Expect(string(appStart.Out.Contents())).To(ContainSubstring(isoSegName))
			})

			AfterEach(func() {
				app_helpers.AppReport(appName)
				Eventually(cf.Cf("delete", appName, "-f", "-r")).Should(Exit(0))
				TestSetup.RegularUserContext().TargetSpace()
			})

			It("maps a single external port to an application's container port", func() {
				resp, err := SendAndReceive(domainName, externalPort1)
				Expect(err).ToNot(HaveOccurred())
				Expect(resp).To(ContainSubstring(serverId1))
			})

			Context("with two different apps", func() {
				var (
					secondAppName string
					serverId2     = "server2"
				)

				BeforeEach(func() {
					secondAppName = random_name.CATSRandomName("APP")
					cmd := fmt.Sprintf("tcp-listener --serverId=%s", serverId2)

					target := cf.Cf("target", "-o", orgName, "-s", spaceName).Wait()
					Expect(target).To(Exit(0), "failed targeting")

					Expect(cf.Cf("push",
						"--no-route",
						"--no-start",
						secondAppName,
						"-p", tcpDropletReceiver,
						"-b", Config.GetGoBuildpackName(),
						"-m", DEFAULT_MEMORY_LIMIT,
						"-f", filepath.Join(tcpDropletReceiver, "manifest.yml"),
						"-c", cmd,
					).Wait()).To(Exit(0))

					Expect(cf.Cf("map-route",
						secondAppName, domainName, "--port", externalPort1,
					).Wait()).To(Exit(0))
					appStart := cf.Cf("start", secondAppName).Wait(Config.CfPushTimeoutDuration())
					Expect(appStart).To(Exit(0))
					Expect(string(appStart.Out.Contents())).To(ContainSubstring(isoSegName))
				})

				AfterEach(func() {
					app_helpers.AppReport(secondAppName)
					Eventually(cf.Cf("delete-route", domainName, "--port", externalPort1, "-f")).Should(Exit(0))
					Eventually(cf.Cf("delete", appName, "-f", "-r")).Should(Exit(0))
					Eventually(cf.Cf("delete", secondAppName, "-f", "-r")).Should(Exit(0))
					TestSetup.RegularUserContext().TargetSpace()
				})

				It("maps single external port to both applications", func() {
					serverResponses, err := GetNServerResponses(10, domainName, externalPort1)
					Expect(err).ToNot(HaveOccurred())
					Expect(serverResponses).To(ContainElement(ContainSubstring(serverId1)))
					Expect(serverResponses).To(ContainElement(ContainSubstring(serverId2)))
				})
			})

			Context("with a second external port", func() {
				var externalPort2 string

				BeforeEach(func() {
					externalPort2 = MapTCPRoute(appName, domainName)
				})

				It("maps both ports to the same application", func() {
					resp1, err := SendAndReceive(domainName, externalPort1)
					Expect(err).ToNot(HaveOccurred())
					Expect(resp1).To(ContainSubstring(serverId1))

					resp2, err := SendAndReceive(domainName, externalPort2)
					Expect(err).ToNot(HaveOccurred())
					Expect(resp2).To(ContainSubstring(serverId1))
				})
			})
		})
	})
})
