package helpers

import (
	"fmt"
	"testing"

	. "github.com/onsi/ginkgo"
	ginkgoconfig "github.com/onsi/ginkgo/config"
	"github.com/onsi/ginkgo/reporters"
	. "github.com/onsi/gomega"
	"github.com/vito/cmdtest"
	. "github.com/vito/cmdtest/matchers"

	"github.com/pivotal-cf-experimental/cf-test-helpers/cf"
)

func setUpSpaceWithUserAccess(uc cf.UserContext, sname string) {
	Expect(cf.Cf("create-space", "-o", uc.Org, sname)).To(ExitWith(0))
	Expect(cf.Cf("set-space-role", uc.Username, uc.Org, sname, "SpaceManager")).To(ExitWith(0))
	Expect(cf.Cf("set-space-role", uc.Username, uc.Org, sname, "SpaceDeveloper")).To(ExitWith(0))
	Expect(cf.Cf("set-space-role", uc.Username, uc.Org, sname, "SpaceAuditor")).To(ExitWith(0))
}

var RegularUserContext cf.UserContext

func GinkgoBootstrap(t *testing.T, suiteName string) {
	RegisterFailHandler(Fail)

	adminContext := AdminUserContext
	defer func() {
		cf.Cf("delete-user", "-f", RegularUserContext.Username)
		cf.Cf("delete-space", "-f", RegularUserContext.Space)
	}()

	var originalCfHomeDir, currentCfHomeDir string

	BeforeEach(func() {
		RegularUserContext = NewRegularUserContext()
		cf.AsUser(adminContext, func() {
			Expect(cf.Cf("create-user", RegularUserContext.Username, RegularUserContext.Password)).To(SayBranches(
				cmdtest.ExpectBranch{"OK", func() {}},
				cmdtest.ExpectBranch{"scim_resource_already_exists", func() {}},
			))
			Expect(cf.Cf("create-org", RegularUserContext.Org)).To(ExitWith(0))

			setUpSpaceWithUserAccess(RegularUserContext, RegularUserContext.Space)
			setUpSpaceWithUserAccess(RegularUserContext, "persistent-space")
		})

		originalCfHomeDir, currentCfHomeDir = cf.InitiateUserContext(RegularUserContext)
		cf.TargetSpace(RegularUserContext)
	})

	AfterEach(func() {
		cf.RestoreUserContext(RegularUserContext, originalCfHomeDir, currentCfHomeDir)
	})

	outputFile := fmt.Sprintf("../results/%s-junit_%d.xml", suiteName, ginkgoconfig.GinkgoConfig.ParallelNode)
	RunSpecsWithDefaultAndCustomReporters(t, suiteName, []Reporter{reporters.NewJUnitReporter(outputFile)})
}
