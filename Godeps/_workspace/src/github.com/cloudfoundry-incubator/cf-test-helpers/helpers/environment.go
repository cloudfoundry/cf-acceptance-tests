package helpers

import (
	"time"

	"github.com/cloudfoundry-incubator/cf-test-helpers/cf"
	"github.com/cloudfoundry-incubator/cf-test-helpers/runner"
)

type SuiteContext interface {
	Setup()
	Teardown()
	SetRunawayQuota()

	AdminUserContext() cf.UserContext
	RegularUserContext() cf.UserContext

	ShortTimeout() time.Duration
	LongTimeout() time.Duration
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

	cf.AsUser(e.context.AdminUserContext(), e.context.ShortTimeout(), func() {
		e.setUpSpaceWithUserAccess(e.context.RegularUserContext())
	})

	e.originalCfHomeDir, e.currentCfHomeDir = cf.InitiateUserContext(e.context.RegularUserContext(), e.context.ShortTimeout())
	cf.TargetSpace(e.context.RegularUserContext(), e.context.ShortTimeout())
}

func (e *Environment) Teardown() {
	cf.RestoreUserContext(e.context.RegularUserContext(), e.context.ShortTimeout(), e.originalCfHomeDir, e.currentCfHomeDir)

	e.context.Teardown()
}

func (e *Environment) setUpSpaceWithUserAccess(uc cf.UserContext) {
	runner.NewCmdRunner(cf.Cf("create-space", "-o", uc.Org, uc.Space), e.context.ShortTimeout()).Run()
	runner.NewCmdRunner(cf.Cf("set-space-role", uc.Username, uc.Org, uc.Space, "SpaceManager"), e.context.ShortTimeout()).Run()
	runner.NewCmdRunner(cf.Cf("set-space-role", uc.Username, uc.Org, uc.Space, "SpaceDeveloper"), e.context.ShortTimeout()).Run()
	runner.NewCmdRunner(cf.Cf("set-space-role", uc.Username, uc.Org, uc.Space, "SpaceAuditor"), e.context.ShortTimeout()).Run()
}
