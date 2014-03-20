package helpers

import (
	. "github.com/cloudfoundry/cf-acceptance-tests/helpers"
	. "github.com/onsi/gomega"
	. "github.com/pivotal-cf-experimental/cf-test-helpers/cf"
	. "github.com/vito/cmdtest/matchers"
)

func LoginAsAdmin() {
	Expect(Cf("login",
		"-u", AdminUserContext.Username,
		"-p", AdminUserContext.Password,
		"-o", AdminUserContext.Org,
		"-s", AdminUserContext.Space,
		AdminUserContext.LoginFlags,
	)).To(ExitWith(0))
}

func LoginAsUser() {
	Expect(Cf("login",
		"-u", RegularUserContext.Username,
		"-p", RegularUserContext.Password,
		"-o", RegularUserContext.Org,
		"-s", RegularUserContext.Space,
		RegularUserContext.LoginFlags,
	)).To(ExitWith(0))
}
