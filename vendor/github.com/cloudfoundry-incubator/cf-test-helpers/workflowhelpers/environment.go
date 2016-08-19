package workflowhelpers

import (
	"time"

	"github.com/cloudfoundry-incubator/cf-test-helpers/cf"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gexec"
)

type SuiteContext interface {
	Setup()
	Teardown()
	SetRunawayQuota()

	AdminUserContext() UserContext
	RegularUserContext() UserContext

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

	AsUser(e.context.AdminUserContext(), e.context.ShortTimeout(), func() {
		e.setUpSpaceWithUserAccess(e.context.RegularUserContext())
	})

	e.originalCfHomeDir, e.currentCfHomeDir = InitiateUserContext(e.context.RegularUserContext(), e.context.ShortTimeout())
	TargetSpace(e.context.RegularUserContext(), e.context.ShortTimeout())
}

func (e *Environment) Teardown() {
	RestoreUserContext(e.context.RegularUserContext(), e.context.ShortTimeout(), e.originalCfHomeDir, e.currentCfHomeDir)

	e.context.Teardown()
}

func (e *Environment) setUpSpaceWithUserAccess(uc UserContext) {
	EventuallyWithOffset(1, cf.Cf("create-space", "-o", uc.Org, uc.Space), e.context.ShortTimeout()).Should(Exit(0))
	EventuallyWithOffset(1, cf.Cf("set-space-role", uc.Username, uc.Org, uc.Space, "SpaceManager"), e.context.ShortTimeout()).Should(Exit(0))
	EventuallyWithOffset(1, cf.Cf("set-space-role", uc.Username, uc.Org, uc.Space, "SpaceDeveloper"), e.context.ShortTimeout()).Should(Exit(0))
	EventuallyWithOffset(1, cf.Cf("set-space-role", uc.Username, uc.Org, uc.Space, "SpaceAuditor"), e.context.ShortTimeout()).Should(Exit(0))
}
