package helpers

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/vito/cmdtest"
	. "github.com/vito/cmdtest/matchers"

	"github.com/pivotal-cf-experimental/cf-test-helpers/cf"
)

var AdminUserContext cf.UserContext
var RegularUserContext cf.UserContext

type SuiteContext interface {
	Setup()
	Teardown()

	AdminUserContext() cf.UserContext
	RegularUserContext() cf.UserContext
}

func SetupEnvironment(t *testing.T, context SuiteContext) {
	var originalCfHomeDir, currentCfHomeDir string

	BeforeEach(func() {
		AdminUserContext = context.AdminUserContext()
		RegularUserContext = context.RegularUserContext()

		context.Setup()

		cf.AsUser(AdminUserContext, func() {
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

		Expect(cf.Cf("delete-user", "-f", RegularUserContext.Username)).To(ExitWith(0))
		Expect(cf.Cf("delete-org", "-f", RegularUserContext.Org)).To(ExitWith(0))

		context.Teardown()
	})
}

func setUpSpaceWithUserAccess(uc cf.UserContext, sname string) {
	Expect(cf.Cf("create-space", "-o", uc.Org, sname)).To(ExitWith(0))
	Expect(cf.Cf("set-space-role", uc.Username, uc.Org, sname, "SpaceManager")).To(ExitWith(0))
	Expect(cf.Cf("set-space-role", uc.Username, uc.Org, sname, "SpaceDeveloper")).To(ExitWith(0))
	Expect(cf.Cf("set-space-role", uc.Username, uc.Org, sname, "SpaceAuditor")).To(ExitWith(0))
}
