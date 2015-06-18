package apps

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gexec"

	"github.com/cloudfoundry-incubator/cf-test-helpers/cf"

	"strings"
)

var _ = Describe("Cloud Controller UAA connectivity", func() {
	It("User added to organization by username", func() {
		cf.AsUser(context.AdminUserContext(), DEFAULT_TIMEOUT, func() {
			orgGuid := cf.Cf("org", context.RegularUserContext().Org, "--guid").Wait(DEFAULT_TIMEOUT).Out.Contents()
			jsonBody := "{\"username\": \"" + strings.TrimSpace(context.RegularUserContext().Username) + "\"}"
			Expect(cf.Cf("curl", "/v2/organizations/"+strings.TrimSpace(string(orgGuid))+"/managers", "-X", "PUT", "-d", jsonBody).Wait(CF_PUSH_TIMEOUT)).To(Exit(0))
			Expect(cf.Cf("org-users", context.RegularUserContext().Org).Wait(DEFAULT_TIMEOUT).Out.Contents()).To(ContainSubstring(context.RegularUserContext().Username))
		})
	})
})
