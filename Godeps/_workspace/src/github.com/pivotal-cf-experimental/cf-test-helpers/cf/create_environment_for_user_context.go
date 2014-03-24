package cf

import (
	. "github.com/onsi/gomega"
	. "github.com/vito/cmdtest/matchers"
)

func CreateEnvironmentForUserContext(adminContext UserContext, userContext UserContext) {
	originalCfHomeDir, currentCfHomeDir := InitiateUserContext(adminContext)
	defer func() {
		RestoreUserContext(userContext, originalCfHomeDir, currentCfHomeDir)
	}()

	createSpace(userContext)
	createRoles(userContext)
}

func createSpace(userContext UserContext) {
	Expect(Cf("target", "-o", userContext.Org)).To(ExitWith(0))
	Expect(Cf("create-space", userContext.Space)).To(ExitWith(0))
}

func createRoles(userContext UserContext) {
	Expect(Cf("set-space-role", userContext.Username, userContext.Org, userContext.Space, "SpaceDeveloper")).To(ExitWith(0))
	Expect(Cf("set-space-role", userContext.Username, userContext.Org, userContext.Space, "SpaceManager")).To(ExitWith(0))
	Expect(Cf("set-space-role", userContext.Username, userContext.Org, userContext.Space, "SpaceAuditor")).To(ExitWith(0))
}

