package cf_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/cloudfoundry-incubator/cf-test-helpers/cf"
)

var _ = Describe("NewUserContext", func() {

	var createUser = func() cf.UserContext {
		return cf.NewUserContext("http://FAKE_API.example.com", "FAKE_USERNAME", "FAKE_PASSWORD", "FAKE_ORG", "FAKE_SPACE", false)
	}

	It("returns a UserContext struct", func() {
		Expect(createUser()).To(BeAssignableToTypeOf(cf.UserContext{}))
	})

	It("sets UserContext.ApiUrl", func() {
		Expect(createUser().ApiUrl).To(Equal("http://FAKE_API.example.com"))
	})

	It("sets UserContext.Username", func() {
		Expect(createUser().Username).To(Equal("FAKE_USERNAME"))
	})

	It("sets UserContext.Password", func() {
		Expect(createUser().Password).To(Equal("FAKE_PASSWORD"))
	})

	It("sets UserContext.Org", func() {
		Expect(createUser().Org).To(Equal("FAKE_ORG"))
	})

	It("sets UserContext.Space", func() {
		Expect(createUser().Space).To(Equal("FAKE_SPACE"))
	})
})
