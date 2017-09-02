package env_test

import (
	"net/http/httptest"
	"os"

	//	. "github.com/cloudfoundry/cf-acceptance-tests/assets/catnip/env"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Env", func() {
	var (
		responseRecorder *httptest.ResponseRecorder
	)
	BeforeEach(func() {
		responseRecorder = httptest.NewRecorder()
		os.Setenv("CATNIP_ENVTEST", "Jellybean")
	})
	AfterEach(func() {
		os.Unsetenv("CATNIP_ENVTEST")
	})

	Describe("NameHandler", func() {
		It("is true", func() {

			Expect(true).To(BeTrue())
		})
	})

	Describe("JsonHandler", func() {

	})
})
