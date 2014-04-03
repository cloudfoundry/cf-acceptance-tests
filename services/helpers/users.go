package helpers

import (
	. "github.com/cloudfoundry/cf-acceptance-tests/helpers"
	. "github.com/onsi/gomega"
	. "github.com/pivotal-cf-experimental/cf-test-helpers/cf"
	. "github.com/vito/cmdtest/matchers"
)

func LoginAsAdmin() {
	Expect(Cf("auth", AdminUserContext.Username, AdminUserContext.Password)).To(ExitWith(0))
}

func LoginAsUser() {
	Expect(Cf("auth", RegularUserContext.Username, RegularUserContext.Password)).To(ExitWith(0))
	Expect(Cf("target", "-o", RegularUserContext.Org, "-s", RegularUserContext.Space)).To(ExitWith(0))
}
