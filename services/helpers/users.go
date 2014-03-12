package helpers

import (
	. "github.com/cloudfoundry/cf-acceptance-tests/helpers"
	. "github.com/onsi/gomega"
	. "github.com/pivotal-cf-experimental/cf-test-helpers/cf"
	. "github.com/vito/cmdtest/matchers"
)

func LoginAsAdmin() {
	adminUserContext := NewAdminUserContext()
	Expect(Cf("login", "-u", adminUserContext.Username, "-p", adminUserContext.Password, "-o", adminUserContext.Org, "-s", adminUserContext.Space)).To(ExitWith(0))
}

func LoginAsUser() {
	regularUserContext := NewRegularUserContext()
	Expect(Cf("login", "-u", regularUserContext.Username, "-p", regularUserContext.Password, "-o", regularUserContext.Org, "-s", regularUserContext.Space)).To(ExitWith(0))
}
