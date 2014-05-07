package helpers

import (
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gexec"

	"github.com/cloudfoundry-incubator/cf-test-helpers/cf"
)

type SuiteContext interface {
	Setup()
	Teardown()

	AdminUserContext() cf.UserContext
	RegularUserContext() cf.UserContext
}

type Environment struct {
	context           SuiteContext
	originalCfHomeDir string
	currentCfHomeDir  string
}

func NewEnvironment(context SuiteContext) *Environment {
	return &Environment{context: context}
}

func (e *Environment) Setup() {
	e.context.Setup()

	cf.AsUser(e.context.AdminUserContext(), func() {
		setUpSpaceWithUserAccess(e.context.RegularUserContext())
	})

	e.originalCfHomeDir, e.currentCfHomeDir = cf.InitiateUserContext(e.context.RegularUserContext())
	cf.TargetSpace(e.context.RegularUserContext())
}

func (e *Environment) Teardown() {
	cf.RestoreUserContext(e.context.RegularUserContext(), e.originalCfHomeDir, e.currentCfHomeDir)

	e.context.Teardown()
}

func setUpSpaceWithUserAccess(uc cf.UserContext) {
	spaceSetupTimeout := 10.0
	Eventually(cf.Cf("create-space", "-o", uc.Org, uc.Space), spaceSetupTimeout).Should(Exit(0))
	Eventually(cf.Cf("set-space-role", uc.Username, uc.Org, uc.Space, "SpaceManager"), spaceSetupTimeout).Should(Exit(0))
	Eventually(cf.Cf("set-space-role", uc.Username, uc.Org, uc.Space, "SpaceDeveloper"), spaceSetupTimeout).Should(Exit(0))
	Eventually(cf.Cf("set-space-role", uc.Username, uc.Org, uc.Space, "SpaceAuditor"), spaceSetupTimeout).Should(Exit(0))
}
