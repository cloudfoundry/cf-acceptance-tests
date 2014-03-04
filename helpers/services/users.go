package helpers

import (
	"os"

	. "github.com/onsi/gomega"
	. "github.com/pivotal-cf-experimental/cf-test-helpers/cf"
	. "github.com/vito/cmdtest/matchers"
)

func LoginAsAdmin() {
	Expect(Cf("login", "-u", os.Getenv("ADMIN_USER"), "-p", os.Getenv("ADMIN_PASSWORD"), "-o", os.Getenv("CF_ORG"), "-s", os.Getenv("CF_SPACE"))).To(ExitWith(0))
}

func LoginAsUser() {
	Expect(Cf("login", "-u", os.Getenv("CF_USER"), "-p", os.Getenv("CF_USER_PASSWORD"), "-o", os.Getenv("CF_ORG"), "-s", os.Getenv("CF_SPACE"))).To(ExitWith(0))
}
