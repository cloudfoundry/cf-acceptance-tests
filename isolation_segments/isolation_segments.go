package isolation_segments

import (
	"encoding/json"
	"fmt"
	"os"

	. "github.com/cloudfoundry/cf-acceptance-tests/cats_suite_helpers"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gexec"

	"github.com/cloudfoundry-incubator/cf-test-helpers/cf"
	"github.com/cloudfoundry-incubator/cf-test-helpers/helpers"
	"github.com/cloudfoundry-incubator/cf-test-helpers/workflowhelpers"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/app_helpers"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/assets"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/config"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/random_name"
	//	"github.com/cloudfoundry-incubator/cf-test-helpers/helpers"
	//	"github.com/cloudfoundry/cf-acceptance-tests/helpers/app_helpers"
)

const (
	SHARED_ISOLATION_SEGMENT_GUID = "933b4c58-120b-499a-b85d-4b6fc9e2903b"
	binaryHi                      = "Hello from a binary"
)

func assignIsolationSegment(orgGuid, isGuid string) {
	Eventually(cf.Cf("curl",
		fmt.Sprintf("/v3/isolation_segments/%s/relationships/organizations", isGuid),
		"-X",
		"POST",
		"-d",
		fmt.Sprintf(`{"data":[{ "guid":"%s" }]}`, orgGuid)),
		Config.DefaultTimeoutDuration()).Should(Exit(0))
}

func setDefaultIsolationSegment(orgGuid, isGuid string) {
	Eventually(cf.Cf("curl",
		fmt.Sprintf("/v3/organizations/%s/relationships/default_isolation_segment", orgGuid),
		"-X",
		"PATCH",
		"-d",
		fmt.Sprintf(`{"data":{"guid":"%s"}}`, isGuid)),
		Config.DefaultTimeoutDuration()).Should(Exit(0))
}

func getV3Guid(response []byte) string {
	type resource struct {
		Guid string `json:"guid"`
	}
	var GetResponse struct {
		Resources []resource `json:"resources"`
	}

	err := json.Unmarshal(response, &GetResponse)
	Expect(err).ToNot(HaveOccurred())

	if len(GetResponse.Resources) == 0 {
		Fail("No guid found for response")
	}

	return GetResponse.Resources[0].Guid
}

func getV2Guid(response []byte) string {
	type metadata struct {
		Guid string `json:"guid"`
	}
	type resource struct {
		Metadata metadata `json:"metadata"`
	}
	var GetResponse struct {
		Resources []resource `json:"resources"`
	}

	err := json.Unmarshal(response, &GetResponse)
	Expect(err).ToNot(HaveOccurred())

	if len(GetResponse.Resources) == 0 {
		Fail("No guid found for response")
	}

	return GetResponse.Resources[0].Metadata.Guid
}

func getIsolationSegmentGuid(name string) string {
	session := cf.Cf("curl", fmt.Sprintf("/v3/isolation_segments?names=%s", name))
	bytes := session.Wait(Config.DefaultTimeoutDuration()).Out.Contents()
	return getV3Guid(bytes)
}

func isolationSegmentExists(name string) bool {
	session := cf.Cf("curl", fmt.Sprintf("/v3/isolation_segments?names=%s", name))
	bytes := session.Wait(Config.DefaultTimeoutDuration()).Out.Contents()
	type resource struct {
		Guid string `json:"guid"`
	}
	var GetResponse struct {
		Resources []resource `json:"resources"`
	}

	err := json.Unmarshal(bytes, &GetResponse)
	Expect(err).ToNot(HaveOccurred())
	return len(GetResponse.Resources) > 0
}

func createIsolationSegment(name string) string {
	session := cf.Cf("curl", "/v3/isolation_segments", "-X", "POST", "-d", fmt.Sprintf(`{"name":"%s"}`, name))
	bytes := session.Wait(Config.DefaultTimeoutDuration()).Out.Contents()

	var isolation_segment struct {
		Guid string `json:"guid"`
	}
	err := json.Unmarshal(bytes, &isolation_segment)
	Expect(err).ToNot(HaveOccurred())

	return isolation_segment.Guid
}

func deleteIsolationSegment(guid string) {
	Eventually(cf.Cf("curl", fmt.Sprintf("/v3/isolation_segments/%s", guid), "-X", "DELETE"), Config.DefaultTimeoutDuration()).Should(Exit(0))
}

var _ = IsolationSegmentsDescribe("IsolationSegments", func() {
	var orgGuid string
	var testSetup *workflowhelpers.ReproducibleTestSuiteSetup
	var isGuid string
	var shouldDeleteIsolationSegment bool

	BeforeEach(func() {
		// New up a organization since we will be assigning isolation segments.
		// This has a potential to cause other tests to fail if running in parallel mode.
		cfg, _ := config.NewCatsConfig(os.Getenv("CONFIG"))
		testSetup = workflowhelpers.NewTestSuiteSetup(cfg)
		testSetup.Setup()

		session := cf.Cf("curl", fmt.Sprintf("/v3/organizations?names=%s", testSetup.GetOrganizationName()))
		bytes := session.Wait(Config.DefaultTimeoutDuration()).Out.Contents()
		orgGuid = getV3Guid(bytes)

		shouldDeleteIsolationSegment = true
	})

	AfterEach(func() {
		testSetup.Teardown()

		if isGuid != "" && shouldDeleteIsolationSegment {
			workflowhelpers.AsUser(testSetup.AdminUserContext(), testSetup.ShortTimeout(), func() {
				deleteIsolationSegment(isGuid)
			})
		}
	})

	Context("When an organization has the shared segment as its default", func() {
		BeforeEach(func() {
			assignIsolationSegment(orgGuid, SHARED_ISOLATION_SEGMENT_GUID)
		})

		It("can push an app to a space with no assigned segment", func() {
			appName := random_name.CATSRandomName("APP")
			Eventually(cf.Cf(
				"push", appName,
				"-p", assets.NewAssets().Binary,
				"--no-start",
				"-m", DEFAULT_MEMORY_LIMIT,
				"-b", "binary_buildpack",
				"-d", Config.GetAppsDomain(),
				"-c", "./app"),
				Config.CfPushTimeoutDuration()).Should(Exit(0))

			app_helpers.EnableDiego(appName)
			Eventually(cf.Cf("start", appName), Config.CfPushTimeoutDuration()).Should(Exit(0))
			Eventually(helpers.CurlingAppRoot(Config, appName), Config.DefaultTimeoutDuration()).Should(ContainSubstring(binaryHi))
		})
	})

	Context("when the user-provided Isolation Segment has an associated cell", func() {
		BeforeEach(func() {
			workflowhelpers.AsUser(testSetup.AdminUserContext(), testSetup.ShortTimeout(), func() {
				name := Config.GetIsolationSegmentName()
				if !isolationSegmentExists(name) {
					isGuid = createIsolationSegment(name)
					shouldDeleteIsolationSegment = true
				} else {
					isGuid = getIsolationSegmentGuid(name)
					shouldDeleteIsolationSegment = false
				}
				assignIsolationSegment(orgGuid, isGuid)
				setDefaultIsolationSegment(orgGuid, isGuid)
			})
		})

		It("can push an app to an org where the default is the user-provided isolation segment", func() {
			appName := random_name.CATSRandomName("APP")
			Eventually(cf.Cf(
				"push", appName,
				"-p", assets.NewAssets().Binary,
				"--no-start",
				"-m", DEFAULT_MEMORY_LIMIT,
				"-b", "binary_buildpack",
				"-d", Config.GetAppsDomain(),
				"-c", "./app"),
				Config.CfPushTimeoutDuration()).Should(Exit(0))

			app_helpers.EnableDiego(appName)
			Eventually(cf.Cf("start", appName), Config.CfPushTimeoutDuration()).Should(Exit(0))
			Eventually(helpers.CurlingAppRoot(Config, appName), Config.DefaultTimeoutDuration()).Should(ContainSubstring(binaryHi))
		})
	})

	Context("When the Isolation Segment has no associated cells", func() {
		BeforeEach(func() {
			workflowhelpers.AsUser(testSetup.AdminUserContext(), testSetup.ShortTimeout(), func() {
				isGuid = createIsolationSegment(random_name.CATSRandomName("IsolationSegment"))
				assignIsolationSegment(orgGuid, isGuid)
				setDefaultIsolationSegment(orgGuid, isGuid)
			})
		})

		It("fails to start an app in the Isolation Segment", func() {
			appName := random_name.CATSRandomName("APP")
			Eventually(cf.Cf(
				"push", appName,
				"-p", assets.NewAssets().Binary,
				"--no-start",
				"-m", DEFAULT_MEMORY_LIMIT,
				"-b", "binary_buildpack",
				"-d", Config.GetAppsDomain(),
				"-c", "./app"),
				Config.CfPushTimeoutDuration()).Should(Exit(0))

			app_helpers.EnableDiego(appName)
			Eventually(cf.Cf("start", appName), Config.CfPushTimeoutDuration()).Should(Exit(1))
		})
	})
})
