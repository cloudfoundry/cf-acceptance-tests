package cf_test

import (
	"os/exec"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/pivotal-cf-experimental/cf-test-helpers/cf"
	"github.com/vito/cmdtest"
)

var _ = Describe("CreateEnvironmentForUserContext", func() {
	var FakeCfCalls = [][]string{}

	var FakeCf = func(args ...string) *cmdtest.Session {
		FakeCfCalls = append(FakeCfCalls, args)
		var session, _ = cmdtest.Start(exec.Command("echo", "nothing"))
		return session
	}
	var user = cf.NewUserContext("http://FAKE_API.example.com", "FAKE_USERNAME", "FAKE_PASSWORD", "FAKE_ORG", "FAKE_SPACE", false)
	var admin = cf.NewUserContext("http://FAKE_API.example.com", "FAKE_ADMIN_USERNAME", "FAKE_ADMIN_PASSWORD", "FAKE_ADMIN_ORG", "FAKE_ADMIN_SPACE", true)

	BeforeEach(func() {
		FakeCfCalls = [][]string{}
		cf.Cf = FakeCf
	})

	It("calls cf api", func() {
		cf.CreateEnvironmentForUserContext(admin, user)

		Expect(FakeCfCalls[0]).To(Equal([]string{"api", "http://FAKE_API.example.com", "--skip-ssl-validation"}))
	})

	It("calls cf auth with admin credentials", func() {
		cf.CreateEnvironmentForUserContext(admin, user)

		Expect(FakeCfCalls[1]).To(Equal([]string{"auth", "FAKE_ADMIN_USERNAME", "FAKE_ADMIN_PASSWORD"}))
	})

	It("calls cf logout", func() {
		cf.CreateEnvironmentForUserContext(admin, user)

		Expect(FakeCfCalls[len(FakeCfCalls)-1]).To(Equal([]string{"logout"}))
	})

	It("targets the user org", func() {
		cf.CreateEnvironmentForUserContext(admin, user)

		Expect(FakeCfCalls[2]).To(Equal([]string{"target", "-o", "FAKE_ORG"}))
	})

	It("calls cf create-space with user space", func() {
		cf.CreateEnvironmentForUserContext(admin, user)

		Expect(FakeCfCalls[3]).To(Equal([]string{"create-space", "FAKE_SPACE"}))
	})

	It("sets up the required space roles", func() {
		cf.CreateEnvironmentForUserContext(admin, user)

		Expect(FakeCfCalls[4]).To(Equal([]string{"set-space-role", "FAKE_USERNAME", "FAKE_ORG", "FAKE_SPACE", "SpaceDeveloper"}))
		Expect(FakeCfCalls[5]).To(Equal([]string{"set-space-role", "FAKE_USERNAME", "FAKE_ORG", "FAKE_SPACE", "SpaceManager"}))
		Expect(FakeCfCalls[6]).To(Equal([]string{"set-space-role", "FAKE_USERNAME", "FAKE_ORG", "FAKE_SPACE", "SpaceAuditor"}))
	})
})
