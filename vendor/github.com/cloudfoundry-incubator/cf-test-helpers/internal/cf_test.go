package internal_test

import (
	"fmt"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gexec"

	"github.com/cloudfoundry-incubator/cf-test-helpers/commandreporter"
	"github.com/cloudfoundry-incubator/cf-test-helpers/internal"
	"github.com/cloudfoundry-incubator/cf-test-helpers/internal/fakes"
)

var _ = Describe("Cf", func() {
	var starter *fakes.FakeCmdStarter
	BeforeEach(func() {
		starter = fakes.NewFakeCmdStarter()
	})

	It("calls the cf cli with the correct command and args", func() {
		Eventually(internal.Cf(starter, "app", "my-app"), 1*time.Second).Should(Exit(0))

		Expect(starter.CalledWith[0].Executable).To(Equal("cf"))
		Expect(starter.CalledWith[0].Args).To(Equal([]string{"app", "my-app"}))
	})

	It("uses a default reporter", func() {
		Eventually(internal.Cf(starter, "app", "my-app"), 1*time.Second).Should(Exit(0))
		Expect(starter.CalledWith[0].Reporter).To(BeAssignableToTypeOf(commandreporter.NewCommandReporter()))
	})

	Context("when the exit code is non-zero", func() {
		BeforeEach(func() {
			starter.ToReturn[0].ExitCode = 42
		})

		It("returns the exit code anyway", func() {
			Eventually(internal.Cf(starter, "app", "my-app"), 1*time.Second).Should(Exit(42))
		})
	})

	Context("when there is an error", func() {
		BeforeEach(func() {
			starter.ToReturn[0].Err = fmt.Errorf("failing now")
		})

		It("panics", func() {
			Expect(func() {
				internal.Cf(starter, "fail")
			}).To(Panic())
		})
	})
})
