package cf_test

import (
	"os"
	"os/exec"
	"time"

	"github.com/cloudfoundry-incubator/cf-test-helpers/cf"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
)

var _ = Describe("AsUser", func() {
	var (
		timeout               = 1 * time.Second
		FakeThingsToRunAsUser = func() {}
		FakeCfCalls           = [][]string{}
	)

	var FakeCf = func(args ...string) *gexec.Session {
		FakeCfCalls = append(FakeCfCalls, args)
		var session, _ = gexec.Start(exec.Command("echo", "nothing"), nil, nil)
		return session
	}
	var user cf.UserContext

	BeforeEach(func() {
		FakeCfCalls = [][]string{}
		cf.Cf = FakeCf
		user = cf.NewUserContext("http://FAKE_API.example.com", "FAKE_USERNAME", "FAKE_PASSWORD", "FAKE_ORG", "FAKE_SPACE", true)
	})

	It("calls cf api", func() {
		cf.AsUser(user, timeout, FakeThingsToRunAsUser)

		Expect(FakeCfCalls[0]).To(Equal([]string{"api", "http://FAKE_API.example.com", "--skip-ssl-validation"}))
	})

	It("calls cf auth", func() {
		cf.AsUser(user, timeout, FakeThingsToRunAsUser)

		Expect(FakeCfCalls[1]).To(Equal([]string{"auth", "FAKE_USERNAME", "FAKE_PASSWORD"}))
	})

	Describe("calling cf target", func() {
		Context("when org is set and space is set", func() {
			It("includes flags to set org and space", func() {
				cf.AsUser(user, timeout, FakeThingsToRunAsUser)

				Expect(FakeCfCalls[2]).To(Equal([]string{"target", "-o", "FAKE_ORG", "-s", "FAKE_SPACE"}))
			})
		})

		Context("when org is set and space is NOT set", func() {
			BeforeEach(func() {
				user.Space = ""
			})

			It("includes a flag to set org but NOT for space", func() {
				cf.AsUser(user, timeout, FakeThingsToRunAsUser)

				Expect(FakeCfCalls[2]).To(Equal([]string{"target", "-o", "FAKE_ORG"}))
			})
		})

		Context("when org is NOT set and space is NOT set", func() {
			BeforeEach(func() {
				user.Org = ""
				user.Space = ""
			})

			It("does not call cf target", func() {
				cf.AsUser(user, timeout, FakeThingsToRunAsUser)

				for _, call := range FakeCfCalls {
					Expect(call).ToNot(ContainElement("target"))
				}
			})
		})

		Context("when org is NOT set and space is set", func() {
			BeforeEach(func() {
				user.Org = ""
			})

			It("does not call cf target", func() {
				cf.AsUser(user, timeout, FakeThingsToRunAsUser)

				for _, call := range FakeCfCalls {
					Expect(call).ToNot(ContainElement("target"))
				}
			})
		})
	})

	It("calls cf logout", func() {
		cf.AsUser(user, timeout, FakeThingsToRunAsUser)

		Expect(FakeCfCalls[len(FakeCfCalls)-1]).To(Equal([]string{"logout"}))
	})

	It("logs out even if actions contain a failing expectation", func() {
		RegisterFailHandler(func(message string, callerSkip ...int) {})
		cf.AsUser(user, timeout, func() { Expect(1).To(Equal(2)) })
		RegisterFailHandler(Fail)

		Expect(FakeCfCalls[len(FakeCfCalls)-1]).To(Equal([]string{"logout"}))
	})

	It("calls the passed function", func() {
		called := false
		cf.AsUser(user, timeout, func() { called = true })

		Expect(called).To(BeTrue())
	})

	It("sets a unique CF_HOME value", func() {
		var (
			firstHome  string
			secondHome string
		)

		cf.AsUser(user, timeout, func() {
			firstHome = os.Getenv("CF_HOME")
		})

		cf.AsUser(user, timeout, func() {
			secondHome = os.Getenv("CF_HOME")
		})

		Expect(firstHome).NotTo(Equal(secondHome))
	})

	It("returns CF_HOME to its original value", func() {
		os.Setenv("CF_HOME", "some-crazy-value")
		cf.AsUser(user, timeout, func() {})

		Expect(os.Getenv("CF_HOME")).To(Equal("some-crazy-value"))
	})
})
