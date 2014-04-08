package helpers

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
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

func SetupEnvironment(context SuiteContext) {
	var originalCfHomeDir, currentCfHomeDir string

	BeforeEach(func() {
		AdminUserContext = context.AdminUserContext()
		RegularUserContext = context.RegularUserContext()

		context.Setup()

		cf.AsUser(AdminUserContext, func() {
			setUpSpaceWithUserAccess(RegularUserContext)
		})

		originalCfHomeDir, currentCfHomeDir = cf.InitiateUserContext(RegularUserContext)
		cf.TargetSpace(RegularUserContext)
	})

	AfterEach(func() {
		cf.RestoreUserContext(RegularUserContext, originalCfHomeDir, currentCfHomeDir)

		context.Teardown()
	})
}

func setUpSpaceWithUserAccess(uc cf.UserContext) {
	Expect(cf.Cf("create-space", "-o", uc.Org, uc.Space)).To(ExitWith(0))
	Expect(cf.Cf("set-space-role", uc.Username, uc.Org, uc.Space, "SpaceManager")).To(ExitWith(0))
	Expect(cf.Cf("set-space-role", uc.Username, uc.Org, uc.Space, "SpaceDeveloper")).To(ExitWith(0))
	Expect(cf.Cf("set-space-role", uc.Username, uc.Org, uc.Space, "SpaceAuditor")).To(ExitWith(0))
}
